package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/akira-io/billing-sdk-go/client"
	"github.com/akira-io/billing-sdk-go/license"
)

type Payload struct {
	Product    string `json:"product"`
	Feature    string `json:"feature"`
	Date       string `json:"date"`
	DeviceFP   string `json:"device_fp"`
	Action     string `json:"action"`
	Count      int    `json:"count,omitempty"`
	Platform   string `json:"platform,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
}

type Response struct {
	Count   int     `json:"count"`
	Limit   *int    `json:"limit"`
	Period  *string `json:"period,omitempty"`
	Allowed bool    `json:"allowed"`
}

func Track(ctx context.Context, c *client.Client, payload Payload) (*Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &Response{}
	if err := c.Do(ctx, "POST", "/api/me/usage", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

func TrackAnonymous(ctx context.Context, c *client.Client, payload Payload) (*Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &Response{}
	if err := c.Do(ctx, "POST", "/api/v1/usage/anonymous", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

type Buffer interface {
	Add(ctx context.Context, feature string, delta uint64) error
	Drain(ctx context.Context) (map[string]uint64, error)
	Restore(ctx context.Context, deltas map[string]uint64) error
}

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

type SyncFunc func(ctx context.Context, deltas map[string]uint64, serial uint64) (*license.SyncUsageResponse, error)

type SerialProvider func(ctx context.Context) (uint64, error)

type RefreshHandler func(ctx context.Context, resp *license.SyncUsageResponse) error

type TrackerOptions struct {
	Buffer        Buffer
	Sync          SyncFunc
	Serial        SerialProvider
	OnRefresh     RefreshHandler
	FlushInterval time.Duration
}

type Tracker struct {
	opts    TrackerOptions
	running atomic.Bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewTracker(opts TrackerOptions) (*Tracker, error) {
	if opts.Buffer == nil {
		return nil, fmt.Errorf("billing: tracker requires Buffer")
	}
	if opts.Sync == nil {
		return nil, fmt.Errorf("billing: tracker requires Sync")
	}
	if opts.FlushInterval <= 0 {
		opts.FlushInterval = 5 * time.Minute
	}
	return &Tracker{opts: opts}, nil
}

func (t *Tracker) TrackDelta(ctx context.Context, feature string, delta uint64) error {
	if delta == 0 {
		return nil
	}
	return t.opts.Buffer.Add(ctx, feature, delta)
}

func (t *Tracker) Flush(ctx context.Context) error {
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

func (t *Tracker) Start(ctx context.Context) {
	if !t.running.CompareAndSwap(false, true) {
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.done = make(chan struct{})
	go t.loop(loopCtx)
}

func (t *Tracker) Stop(ctx context.Context) error {
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

func (t *Tracker) loop(ctx context.Context) {
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
