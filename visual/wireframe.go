package visual

import (
	"image/color"

	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/go-gl/mathgl/mgl64"
)

// Segment is one line in a wireframe preview.
type Segment struct {
	Start, End mgl64.Vec3
}

// Wireframe manages a reusable set of debug lines. It is the generic visual
// primitive for selection outlines, paste previews, and other predicted shapes:
// callers provide whatever segments describe the thing they want to preview.
type Wireframe struct {
	lines []*debug.Line
}

// Draw draws or updates the wireframe on r. If fewer segments are supplied than
// in the previous draw, stale lines are removed automatically.
func (w *Wireframe) Draw(r debug.Renderer, segments []Segment, colour color.RGBA) {
	w.removeExtra(r, len(segments))

	for i, segment := range segments {
		line := w.line(i)
		line.Colour = colour
		line.Position = segment.Start
		line.EndPosition = segment.End
		r.AddDebugShape(line)
	}
}

// Remove removes all lines in the wireframe from r. It is safe to call even when
// the wireframe has not been drawn yet.
func (w *Wireframe) Remove(r debug.Renderer) {
	w.removeExtra(r, 0)
}

func (w *Wireframe) line(index int) *debug.Line {
	for len(w.lines) <= index {
		w.lines = append(w.lines, &debug.Line{})
	}
	return w.lines[index]
}

func (w *Wireframe) removeExtra(r debug.Renderer, keep int) {
	if keep >= len(w.lines) {
		return
	}
	for _, line := range w.lines[keep:] {
		r.RemoveDebugShape(line)
	}
	w.lines = w.lines[:keep]
}
