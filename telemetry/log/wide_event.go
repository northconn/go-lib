package log

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"
)

type wideEventCtx struct{}

var wideEventKey wideEventCtx

type WideEvent struct {
	mu sync.Mutex

	logger *slog.Logger

	// fixed-ish metadata
	name      string
	start     time.Time
	level     slog.Level
	message   string
	committed bool

	// canonical fields (accumulated)
	attrs []slog.Attr
}

// New creates a WideEvent with a base name and optional initial attrs.
// It automatically adds common canonical fields.
func New(name string, initial ...slog.Attr) *WideEvent {
	we := &WideEvent{
		logger:  Logger(),
		name:    name,
		start:   time.Now(),
		level:   slog.LevelInfo,
		message: name,
	}

	we.attrs = append(we.attrs,
		slog.String("event", name),
		slog.String("event_id", newEventID()),
		slog.Time("event_start", we.start),
	)

	if len(initial) > 0 {
		we.attrs = append(we.attrs, initial...)
	}

	return we
}

// WithContext injects this WideEvent into a context.
func (we *WideEvent) WithContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, wideEventKey, we)
}

// WideEventFromContext extracts the WideEvent from context, if present.
func WideEventFromContext(ctx context.Context) (*WideEvent, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(wideEventKey)
	if v == nil {
		return nil, false
	}
	we, ok := v.(*WideEvent)
	return we, ok
}

// EnsureWideEventFromContext returns an existing WideEvent from context or creates a new one,
// injects it back, and returns (newCtx, we).
func EnsureWideEventFromContext(ctx context.Context, name string, initial ...slog.Attr) (context.Context, *WideEvent) {
	if we, ok := WideEventFromContext(ctx); ok && we != nil {
		return ctx, we
	}
	we := New(name, initial...)
	return we.WithContext(ctx), we
}

// Add appends attributes to the canonical event.
// It is safe to call from multiple goroutines.
func (we *WideEvent) Add(attrs ...slog.Attr) {
	if len(attrs) == 0 {
		return
	}
	we.mu.Lock()
	defer we.mu.Unlock()

	if we.committed {
		// After commit we intentionally ignore further mutation.
		return
	}

	we.attrs = append(we.attrs, attrs...)
}

// AddKVs convenience: Add key/value pairs (even length; odd last is ignored).
func (we *WideEvent) AddKVs(kv ...any) {
	if len(kv) == 0 {
		return
	}
	attrs := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok || k == "" {
			continue
		}
		attrs = append(attrs, slog.Any(k, kv[i+1]))
	}
	we.Add(attrs...)
}

// SetLevel changes the level that will be used on Commit (default INFO).
func (we *WideEvent) SetLevel(level slog.Level) {
	we.mu.Lock()
	defer we.mu.Unlock()
	if we.committed {
		return
	}
	we.level = level
}

// SetMessage changes the final message used on Commit (default is name).
func (we *WideEvent) SetMessage(msg string) {
	if msg == "" {
		return
	}
	we.mu.Lock()
	defer we.mu.Unlock()
	if we.committed {
		return
	}
	we.message = msg
}

// Commit emits the canonical log as a single record with all accumulated attrs.
// Safe to call multiple times; only the first commit logs.
func (we *WideEvent) Commit(ctx context.Context, extra ...slog.Attr) {
	we.mu.Lock()
	if we.committed {
		we.mu.Unlock()
		return
	}
	we.committed = true

	// Copy attrs so we can unlock before logging.
	attrs := make([]slog.Attr, 0, len(we.attrs)+8+len(extra))
	attrs = append(attrs, we.attrs...)
	we.mu.Unlock()

	end := time.Now()
	duration := end.Sub(we.start)

	attrs = append(attrs,
		slog.Time("event_end", end),
		slog.Duration("event_duration", duration),
	)

	if len(extra) > 0 {
		attrs = append(attrs, extra...)
	}

	// Emit one single canonical log entry.
	we.logger.LogAttrs(ctx, we.level, we.message, attrs...)
}

// CommitError is a helper that marks the event as errored, adds error fields,
// and commits at ERROR level.
func (we *WideEvent) CommitError(ctx context.Context, err error, extra ...slog.Attr) {
	if err != nil {
		we.Add(
			slog.Bool("error", true),
			slog.String("error_message", err.Error()),
		)
	} else {
		we.Add(slog.Bool("error", true))
	}

	we.SetLevel(slog.LevelError)
	we.Commit(ctx, extra...)
}

// Cancel is an optional semantic helper: mark as cancelled and commit.
// Useful for early returns/timeouts.
func (we *WideEvent) Cancel(ctx context.Context, reason string, extra ...slog.Attr) {
	we.Add(
		slog.Bool("cancelled", true),
	)
	if reason != "" {
		we.Add(slog.String("cancel_reason", reason))
	}
	we.SetLevel(slog.LevelWarn)
	we.Commit(ctx, extra...)
}

func newEventID() string {
	// 16 bytes -> 32 hex chars; collision-resistant enough for request-scoped logs.
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
