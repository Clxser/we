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
// A Wireframe must not be copied after first use. Draw and Remove calls should
// originate from the world tick goroutine, such as a player handler or world.Tx
// callback, because Dragonfly drains debug-shape updates on that goroutine.
type Wireframe struct {
	mu       sync.Mutex
	lines    []*debug.Line
	segments []Segment
	colour   color.RGBA
}

// Draw draws or updates the wireframe on r. Existing line slots are upserted
// with stable debug shape IDs. If fewer segments are supplied than in the
// previous draw, stale lines are removed automatically.
func (w *Wireframe) Draw(r debug.Renderer, segments []Segment, colour color.RGBA) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.drawLocked(r, segments, colour)
}

func (w *Wireframe) drawLocked(r debug.Renderer, segments []Segment, colour color.RGBA) {
	if w.sameDraw(segments, colour) {
		return
	}
	for len(w.lines) > len(segments) {
		i := len(w.lines) - 1
		r.RemoveDebugShape(w.lines[i])
		w.lines = w.lines[:i]
	}
	for i, segment := range segments {
		if i == len(w.lines) {
			line := &debug.Line{Colour: colour, Position: segment.Start, EndPosition: segment.End}
			w.lines = append(w.lines, line)
			r.AddDebugShape(line)
			continue
		}
		if w.colour == colour && i < len(w.segments) && w.segments[i] == segment {
			continue
		}
		line := w.lines[i]
		line.Colour = colour
		line.Position = segment.Start
		line.EndPosition = segment.End
		r.AddDebugShape(line)
	}
	w.segments = append(w.segments[:0], segments...)
	w.colour = colour
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
