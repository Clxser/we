package edit

import (
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

// traceEnabled is read on every trace call. Toggle via SetPasteTracing or
// the WE_PASTE_TRACE=1 environment variable (read once at package init).
var traceEnabled atomic.Bool

func init() {
	if v := os.Getenv("WE_PASTE_TRACE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil && b {
			traceEnabled.Store(true)
		}
	}
}

// SetPasteTracing toggles per-step paste/import tracing. When enabled,
// each phase logs duration + heap delta via slog (default destination
// is stderr). Useful for diagnosing OOMs and freezes on huge schematics.
func SetPasteTracing(enabled bool) { traceEnabled.Store(enabled) }

// PasteTracingEnabled reports whether tracing is currently on.
func PasteTracingEnabled() bool { return traceEnabled.Load() }

// traceStep holds the start state for a single timed phase. Use startTrace
// + step.end(). When tracing is off the calls are near-zero cost (one
// atomic load each).
type traceStep struct {
	name      string
	start     time.Time
	startHeap uint64
	startSys  uint64
}

func startTrace(name string) traceStep {
	if !traceEnabled.Load() {
		return traceStep{}
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("paste.trace.start",
		"step", name,
		"heap_mb", m.HeapAlloc/1024/1024,
		"sys_mb", m.Sys/1024/1024,
		"goroutines", runtime.NumGoroutine(),
	)
	return traceStep{name: name, start: time.Now(), startHeap: m.HeapAlloc, startSys: m.Sys}
}

func (t traceStep) end() {
	if t.name == "" {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("paste.trace.end",
		"step", t.name,
		"duration", time.Since(t.start).String(),
		"heap_mb", m.HeapAlloc/1024/1024,
		"sys_mb", m.Sys/1024/1024,
		"heap_delta_mb", deltaMB(t.startHeap, m.HeapAlloc),
		"sys_delta_mb", deltaMB(t.startSys, m.Sys),
		"goroutines", runtime.NumGoroutine(),
	)
}

func (t traceStep) endf(suffix string) {
	if t.name == "" {
		return
	}
	t.name = t.name + " " + suffix
	t.end()
}

func deltaMB(before, after uint64) int64 {
	d := int64(after) - int64(before)
	return d / 1024 / 1024
}

// traceAnnotate prints a one-shot status line with no duration — useful
// inside long-running loops to mark progress (e.g. "wrote 10M cells so far").
func traceAnnotate(msg string, kv ...any) {
	if !traceEnabled.Load() {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	args := []any{
		"heap_mb", m.HeapAlloc / 1024 / 1024,
		"sys_mb", m.Sys / 1024 / 1024,
	}
	args = append(args, kv...)
	slog.Info("paste.trace.note "+msg, args...)
}
