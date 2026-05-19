package usage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akira-io/billing-sdk-go/license"
)

func TestTrackerFlush(t *testing.T) {
	buf := NewMemoryBuffer()
	var syncedDeltas map[string]uint64
	var syncedSerial uint64
	refreshed := 0

	tracker, err := NewTracker(TrackerOptions{
		Buffer: buf,
		Sync: func(_ context.Context, deltas map[string]uint64, serial uint64) (*license.SyncUsageResponse, error) {
			syncedDeltas = deltas
			syncedSerial = serial
			return &license.SyncUsageResponse{Applied: deltas, Serial: serial + 1}, nil
		},
		Serial:    func(context.Context) (uint64, error) { return 42, nil },
		OnRefresh: func(context.Context, *license.SyncUsageResponse) error { refreshed++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := tracker.TrackDelta(ctx, "requests_per_day", 3); err != nil {
		t.Fatal(err)
	}
	if err := tracker.TrackDelta(ctx, "requests_per_day", 2); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Flush(ctx); err != nil {
		t.Fatal(err)
	}

	if syncedDeltas["requests_per_day"] != 5 || syncedSerial != 42 || refreshed != 1 {
		t.Fatalf("unexpected flush state deltas=%v serial=%d refreshed=%d", syncedDeltas, syncedSerial, refreshed)
	}
}

func TestTrackerRollbackOnSyncError(t *testing.T) {
	buf := NewMemoryBuffer()
	syncErr := errors.New("boom")

	tracker, _ := NewTracker(TrackerOptions{
		Buffer: buf,
		Sync: func(context.Context, map[string]uint64, uint64) (*license.SyncUsageResponse, error) {
			return nil, syncErr
		},
	})

	ctx := context.Background()
	_ = tracker.TrackDelta(ctx, "f", 4)
	if err := tracker.Flush(ctx); !errors.Is(err, syncErr) {
		t.Fatalf("want syncErr got %v", err)
	}

	deltas, _ := buf.Drain(ctx)
	if deltas["f"] != 4 {
		t.Fatalf("rollback failed, deltas=%v", deltas)
	}
}

func TestTrackerStartStop(t *testing.T) {
	buf := NewMemoryBuffer()
	done := make(chan struct{}, 4)

	tracker, _ := NewTracker(TrackerOptions{
		Buffer:        buf,
		FlushInterval: 10 * time.Millisecond,
		Sync: func(_ context.Context, deltas map[string]uint64, _ uint64) (*license.SyncUsageResponse, error) {
			select {
			case done <- struct{}{}:
			default:
			}
			return &license.SyncUsageResponse{Applied: deltas}, nil
		},
	})

	ctx := context.Background()
	_ = tracker.TrackDelta(ctx, "f", 1)
	tracker.Start(ctx)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("flusher did not run")
	}

	if err := tracker.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
