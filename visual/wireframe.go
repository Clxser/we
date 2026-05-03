package visual

import (
	"image/color"
	"slices"
	"sync"

	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/go-gl/mathgl/mgl64"
)

// Segment is one line in a wireframe preview.
type Segment struct {
	Start, End mgl64.Vec3
}

// LineSegment returns a single wireframe segment from start to end.
func LineSegment(start, end mgl64.Vec3) Segment {
	return Segment{Start: start, End: end}
}

// Wireframe manages a reusable set of debug lines. It is the generic visual
// primitive for selection outlines, paste previews, and other predicted shapes:
// callers provide whatever segments describe the thing they want to preview.
//
// A Wireframe must not be copied after first use.
type Wireframe struct {
	mu           sync.Mutex
	lines        []*debug.Line
	segments     []Segment
	colour       color.RGBA
	asyncMu      sync.Mutex
	asyncOp      *wireframeOp
	asyncRunning bool
}

type wireframeOp struct {
	renderer debug.Renderer
	segments []Segment
	colour   color.RGBA
	remove   bool
}

// Draw draws or updates the wireframe on r. If fewer segments are supplied than
// in the previous draw, stale lines are removed automatically.
func (w *Wireframe) Draw(r debug.Renderer, segments []Segment, colour color.RGBA) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.drawLocked(r, segments, colour)
}

// DrawAsync draws or updates the wireframe without blocking the caller on the
// renderer. Calls are coalesced through one ephemeral worker goroutine per
// Wireframe, so only the latest pending draw/remove is kept while a renderer is
// blocked. The renderer must remain valid until the worker drains. The caller
// may reuse or mutate segments after this returns.
func (w *Wireframe) DrawAsync(r debug.Renderer, segments []Segment, colour color.RGBA) {
	segments = append([]Segment(nil), segments...)
	w.enqueue(wireframeOp{renderer: r, segments: segments, colour: colour})
}

func (w *Wireframe) drawLocked(r debug.Renderer, segments []Segment, colour color.RGBA) {
	if w.sameDraw(segments, colour) {
		return
	}
	w.removeLocked(r)
	w.segments = append(w.segments[:0], segments...)
	w.colour = colour
	for _, segment := range segments {
		line := &debug.Line{Colour: colour, Position: segment.Start, EndPosition: segment.End}
		w.lines = append(w.lines, line)
		r.AddDebugShape(line)
	}
}

func (w *Wireframe) sameDraw(segments []Segment, colour color.RGBA) bool {
	return len(w.lines) > 0 && w.colour == colour && slices.Equal(w.segments, segments)
}

// Remove removes all lines in the wireframe from r. It is safe to call even when
// the wireframe has not been drawn yet.
func (w *Wireframe) Remove(r debug.Renderer) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.removeLocked(r)
}

// RemoveAsync removes all lines in the wireframe without blocking the caller on
// the renderer. Calls are coalesced with pending DrawAsync calls, so the latest
// requested state wins. The renderer must remain valid until the worker drains.
func (w *Wireframe) RemoveAsync(r debug.Renderer) {
	w.enqueue(wireframeOp{renderer: r, remove: true})
}

func (w *Wireframe) enqueue(op wireframeOp) {
	w.asyncMu.Lock()
	w.asyncOp = &op
	if w.asyncRunning {
		w.asyncMu.Unlock()
		return
	}
	w.asyncRunning = true
	w.asyncMu.Unlock()
	go w.runAsync()
}

func (w *Wireframe) runAsync() {
	for {
		w.asyncMu.Lock()
		op := w.asyncOp
		w.asyncOp = nil
		if op == nil {
			w.asyncRunning = false
			w.asyncMu.Unlock()
			return
		}
		w.asyncMu.Unlock()
		if op.remove {
			w.Remove(op.renderer)
			continue
		}
		w.Draw(op.renderer, op.segments, op.colour)
	}
}

func (w *Wireframe) removeLocked(r debug.Renderer) {
	if len(w.lines) == 0 {
		return
	}
	for _, line := range w.lines {
		r.RemoveDebugShape(line)
	}
	w.lines = nil
	w.segments = nil
}
