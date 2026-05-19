# Usage Tracker

Buffers feature deltas in memory (or behind a custom store), flushes them on a `time.Ticker`, and feeds the refreshed snapshot back to the caller. Pairs with `Client.LicenseSyncUsage` for the `offline_snapshot` licensing mode.

```go
import billing "github.com/akira-io/billing-sdk-go"
```

For `online_realtime` mode, skip the tracker and call `Client.TrackUsage` directly with `Action = "check"` then `Action = "increment"`.

## UsageBuffer interface

```go
type UsageBuffer interface {
    Add(ctx context.Context, feature string, delta uint64) error
    Drain(ctx context.Context) (map[string]uint64, error)
    Restore(ctx context.Context, deltas map[string]uint64) error
}
```

Contract:

- `Add` — accumulate `delta` for the feature.
- `Drain` — return all pending deltas and **clear** the buffer atomically.
- `Restore` — merge deltas back on flush failure. Add to existing pending count, do not overwrite.

Implement a custom buffer for durability (sqlite, badger, etc.). `MemoryBuffer` is fine for short-lived processes.

## MemoryBuffer

```go
type MemoryBuffer struct { /* sync.Mutex + map[string]uint64 */ }

func NewMemoryBuffer() *MemoryBuffer
```

In-process implementation. Concurrent `Add`/`Drain`/`Restore` are mutex-guarded.

## TrackerOptions

```go
type TrackerOptions struct {
    Buffer        UsageBuffer
    Sync          SyncUsageFunc
    Serial        SerialProvider
    OnRefresh     RefreshHandler
    FlushInterval time.Duration   // default 5 minutes when ≤ 0
}

type SyncUsageFunc func(ctx context.Context,
    deltas map[string]uint64,
    serial uint64,
) (*LicenseSyncUsageResponse, error)

type SerialProvider func(ctx context.Context) (uint64, error)
type RefreshHandler func(ctx context.Context, resp *LicenseSyncUsageResponse) error
```

| Field | Required | Purpose |
|-------|----------|---------|
| `Buffer` | yes | Persists pending deltas. |
| `Sync` | yes | Calls `Client.LicenseSyncUsage`. |
| `Serial` | no | Reads the local snapshot's `Serial`. Omit to send `0`. |
| `OnRefresh` | no | Persist the refreshed snapshot. |
| `FlushInterval` | no | Auto-flush cadence; default 5 minutes. |

## UsageTracker

```go
type UsageTracker struct { /* … */ }

func NewUsageTracker(opts TrackerOptions) (*UsageTracker, error)

func (t *UsageTracker) Track(ctx context.Context, feature string, delta uint64) error
func (t *UsageTracker) Flush(ctx context.Context) error
func (t *UsageTracker) Start(ctx context.Context)
func (t *UsageTracker) Stop(ctx context.Context) error
```

- `NewUsageTracker` validates `Buffer` and `Sync`. Returns an error otherwise.
- `Track` — appends to the buffer. No-op when `delta == 0`.
- `Flush` — drains the buffer, calls `Serial` + `Sync`, fires `OnRefresh`. Empty buffer is a no-op. On `Serial`/`Sync` error, restores the buffer; propagates the error.
- `Start` — launches a background goroutine running `time.NewTicker(FlushInterval)`. Idempotent — second call is a no-op (atomic CAS guards the `running` flag).
- `Stop` — cancels the goroutine context, waits for it to finish, runs one final `Flush`.

## Rollback semantics

```
Flush(ctx)
  ├── Buffer.Drain(ctx)         ── empty? return nil
  ├── Serial?(ctx)              ── err? Buffer.Restore(deltas); return err
  ├── Sync(ctx, deltas, serial) ── err? Buffer.Restore(deltas); return err
  └── OnRefresh?(ctx, resp)
```

Restore is best-effort (`_ = t.opts.Buffer.Restore(...)`) — a subsequent `Add` collision with a restore is acceptable because both paths add to the existing count.

The auto-flush goroutine swallows errors (`_ = t.Flush(ctx)`) and keeps ticking. Surface failures via your own observability inside `Sync`.

## Concurrency

- The `*http.Client` used by `Client.LicenseSyncUsage` is shared and safe for concurrent use.
- `MemoryBuffer` is mutex-guarded.
- The tracker uses `atomic.Bool` for the `running` flag and a `context.CancelFunc` to stop the goroutine.

Tracker-level concurrency is bounded by the auto-flush goroutine; manual `Flush()` calls can race with the auto-flush. In normal use, prefer `Start` + `Track` and let the goroutine handle flushing.

## Worked example — agent runtime

```go
tracker, err := billing.NewUsageTracker(billing.TrackerOptions{
    Buffer: billing.NewMemoryBuffer(),
    Sync: func(ctx context.Context, deltas map[string]uint64, serial uint64) (*billing.LicenseSyncUsageResponse, error) {
        return client.LicenseSyncUsage(ctx, billing.LicenseSyncUsagePayload{
            Product:     "unified-dev",
            Fingerprint: fp,
            Serial:      serial,
            Deltas:      deltas,
        })
    },
    Serial: func(ctx context.Context) (uint64, error) {
        return store.ReadSerial(ctx)
    },
    OnRefresh: func(ctx context.Context, resp *billing.LicenseSyncUsageResponse) error {
        if err := store.WriteLicense(ctx, resp.License); err != nil {
            return err
        }
        return store.WriteSerial(ctx, resp.Serial)
    },
    FlushInterval: 5 * time.Minute,
})
if err != nil {
    return err
}

tracker.Start(ctx)
defer tracker.Stop(context.Background())

// Application code
tracker.Track(ctx, "agent_run", 1)
```

`Stop` takes its own context so shutdown can have a deadline distinct from the request context.

---

Navigation: [← Gate](31-gate.md) · **Usage** · [Lifecycle →](33-lifecycle.md)
