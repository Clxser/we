package visual

import (
	"image/color"
	"testing"

	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/go-gl/mathgl/mgl64"
)

type debugRenderer struct {
	added   []debug.Shape
	removed []debug.Shape
	visible []debug.Shape
}

func (r *debugRenderer) AddDebugShape(shape debug.Shape) {
	r.added = append(r.added, shape)
}

func (r *debugRenderer) RemoveDebugShape(shape debug.Shape) {
	r.removed = append(r.removed, shape)
}

func (r *debugRenderer) VisibleDebugShapes() []debug.Shape {
	return r.visible
}

func (r *debugRenderer) RemoveAllDebugShapes() {}

func TestWireframeDrawCreatesAndShrinksLines(t *testing.T) {
	r := &debugRenderer{}
	var w Wireframe

	w.Draw(r, []Segment{
		{Start: mgl64.Vec3{0, 0, 0}, End: mgl64.Vec3{1, 0, 0}},
		{Start: mgl64.Vec3{0, 0, 0}, End: mgl64.Vec3{0, 1, 0}},
	}, color.RGBA{R: 255, A: 255})
	if len(r.added) != 2 {
		t.Fatalf("added after first draw = %d, want 2", len(r.added))
	}
	if len(r.removed) != 0 {
		t.Fatalf("removed after first draw = %d, want 0", len(r.removed))
	}

	w.Draw(r, []Segment{{Start: mgl64.Vec3{0, 0, 0}, End: mgl64.Vec3{0, 0, 1}}}, color.RGBA{G: 255, A: 255})
	if len(r.added) != 3 {
		t.Fatalf("added after second draw = %d, want 3", len(r.added))
	}
	if len(r.removed) != 1 {
		t.Fatalf("removed after second draw = %d, want 1", len(r.removed))
	}

	w.Remove(r)
	if len(r.removed) != 2 {
		t.Fatalf("removed after remove = %d, want 2", len(r.removed))
	}
}

func TestWireframeDrawSkipsUnchangedSegments(t *testing.T) {
	r := &debugRenderer{}
	var w Wireframe
	segments := []Segment{{Start: mgl64.Vec3{0, 0, 0}, End: mgl64.Vec3{1, 0, 0}}}
	colour := color.RGBA{R: 255, A: 255}

	w.Draw(r, segments, colour)
	w.Draw(r, segments, colour)
	if len(r.added) != 1 {
		t.Fatalf("added after unchanged redraw = %d, want 1", len(r.added))
	}
	if len(r.removed) != 0 {
		t.Fatalf("removed after unchanged redraw = %d, want 0", len(r.removed))
	}

	w.Draw(r, segments, color.RGBA{G: 255, A: 255})
	if len(r.added) != 2 {
		t.Fatalf("added after colour change = %d, want 2", len(r.added))
	}
	if len(r.removed) != 0 {
		t.Fatalf("removed after colour change = %d, want 0", len(r.removed))
	}
}

func TestWireframeDrawColourChangeReusesLines(t *testing.T) {
	r := &debugRenderer{}
	var w Wireframe
	segment := Segment{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}

	w.Draw(r, []Segment{segment}, color.RGBA{R: 255, A: 255})
	w.Draw(r, []Segment{segment}, color.RGBA{G: 255, A: 255})
	if len(r.added) != 2 {
		t.Fatalf("added after colour change = %d, want 2", len(r.added))
	}
	if len(r.removed) != 0 {
		t.Fatalf("removed after colour change = %d, want 0", len(r.removed))
	}
	if r.added[0] != r.added[1] {
		t.Fatal("colour change allocated a new debug shape")
	}
	if r.added[0].ShapeID() != r.added[1].ShapeID() {
		t.Fatal("colour change changed debug shape ID")
	}
	line := r.added[1].(*debug.Line)
	if line.Colour != (color.RGBA{G: 255, A: 255}) {
		t.Fatalf("updated line colour = %v, want green", line.Colour)
	}
}

func TestWireframeDrawUpdatesExistingLines(t *testing.T) {
	r := &debugRenderer{}
	var w Wireframe

	w.Draw(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}}, color.RGBA{A: 255})
	w.Draw(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{2, 0, 0}}}, color.RGBA{A: 255})
	if len(r.added) != 2 {
		t.Fatalf("added after update = %d, want 2", len(r.added))
	}
	if len(r.removed) != 0 {
		t.Fatalf("removed after update = %d, want 0", len(r.removed))
	}
	if r.added[0] != r.added[1] {
		t.Fatal("draw update allocated a new debug shape")
	}
	if r.added[0].ShapeID() != r.added[1].ShapeID() {
		t.Fatal("draw update changed debug shape ID")
	}
	line := r.added[1].(*debug.Line)
	if line.EndPosition != (mgl64.Vec3{2, 0, 0}) {
		t.Fatalf("updated line end = %v, want [2 0 0]", line.EndPosition)
	}
}
