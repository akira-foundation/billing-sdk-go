package billing

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// UsageBuffer is the local store for pending counter deltas.
type UsageBuffer interface {
	Add(ctx context.Context, feature string, delta uint64) error
	Drain(ctx context.Context) (map[string]uint64, error)
	Restore(ctx context.Context, deltas map[string]uint64) error
}

// MemoryBuffer is an in-memory UsageBuffer suitable for tests or stateless apps.
type MemoryBuffer struct {
	mu    sync.Mutex
	state map[string]uint64
}

func NewMemoryBuffer() *MemoryBuffer {
	return &MemoryBuffer{state: map[string]uint64{}}
}

func (b *MemoryBuffer) Add(_ context.Context, feature string, delta uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state[feature] += delta
	return nil
}

func (b *MemoryBuffer) Drain(_ context.Context) (map[string]uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.state
	b.state = map[string]uint64{}
	return out, nil
}

func (b *MemoryBuffer) Restore(_ context.Context, deltas map[string]uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range deltas {
		b.state[k] += v
	}
	return nil
}

// SyncUsageFunc dispatches deltas to the server. Returns the refreshed license
// + applied counts so the caller can persist the new snapshot.
type SyncUsageFunc func(ctx context.Context, deltas map[string]uint64, serial uint64) (*LicenseSyncUsageResponse, error)

// SerialProvider returns the current cached license serial.
type SerialProvider func(ctx context.Context) (uint64, error)

// RefreshHandler receives the freshly synced license envelope.
type RefreshHandler func(ctx context.Context, resp *LicenseSyncUsageResponse) error

// TrackerOptions configures a UsageTracker.
type TrackerOptions struct {
	Buffer        UsageBuffer
	Sync          SyncUsageFunc
	Serial        SerialProvider
	OnRefresh     RefreshHandler
	FlushInterval time.Duration
}

// UsageTracker buffers deltas and flushes them in the background.
type UsageTracker struct {
	opts    TrackerOptions
	running atomic.Bool
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewUsageTracker validates options and returns a ready tracker.
func NewUsageTracker(opts TrackerOptions) (*UsageTracker, error) {
	if opts.Buffer == nil {
		return nil, fmt.Errorf("billing: tracker requires Buffer")
	}
	if opts.Sync == nil {
		return nil, fmt.Errorf("billing: tracker requires Sync")
	}
	if opts.FlushInterval <= 0 {
		opts.FlushInterval = 5 * time.Minute
	}
	return &UsageTracker{opts: opts}, nil
}

// Track adds delta to the buffer for feature.
func (t *UsageTracker) Track(ctx context.Context, feature string, delta uint64) error {
	if delta == 0 {
		return nil
	}
	return t.opts.Buffer.Add(ctx, feature, delta)
}

// Flush drains the buffer and pushes a single sync call. On error the deltas
// are restored so the next flush retries them.
func (t *UsageTracker) Flush(ctx context.Context) error {
	deltas, err := t.opts.Buffer.Drain(ctx)
	if err != nil {
		return err
	}
	if len(deltas) == 0 {
		return nil
	}

	var serial uint64
	if t.opts.Serial != nil {
		s, err := t.opts.Serial(ctx)
		if err != nil {
			_ = t.opts.Buffer.Restore(ctx, deltas)
			return err
		}
		serial = s
	}

	resp, err := t.opts.Sync(ctx, deltas, serial)
	if err != nil {
		_ = t.opts.Buffer.Restore(ctx, deltas)
		return err
	}

	if t.opts.OnRefresh != nil && resp != nil {
		if err := t.opts.OnRefresh(ctx, resp); err != nil {
			return err
		}
	}
	return nil
}

// Start launches a background flusher. Calling twice is a no-op.
func (t *UsageTracker) Start(ctx context.Context) {
	if !t.running.CompareAndSwap(false, true) {
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.done = make(chan struct{})
	go t.loop(loopCtx)
}

// Stop halts the flusher and performs one final flush.
func (t *UsageTracker) Stop(ctx context.Context) error {
	if !t.running.CompareAndSwap(true, false) {
		return nil
	}
	if t.cancel != nil {
		t.cancel()
	}
	if t.done != nil {
		<-t.done
	}
	return t.Flush(ctx)
}

func (t *UsageTracker) loop(ctx context.Context) {
	defer close(t.done)
	ticker := time.NewTicker(t.opts.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = t.Flush(ctx)
		}
	}
}
