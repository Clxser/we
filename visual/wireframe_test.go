package visual

import (
	"image/color"
	"sync"
	"testing"
	"time"

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

type blockingRenderer struct {
	entered    chan struct{}
	exited     chan struct{}
	removed    chan struct{}
	release    chan struct{}
	addOnce    sync.Once
	removeOnce sync.Once
}

func (r *blockingRenderer) AddDebugShape(debug.Shape) {
	r.addOnce.Do(func() { close(r.entered) })
	<-r.release
	close(r.exited)
}

func (r *blockingRenderer) RemoveDebugShape(debug.Shape) {
	r.removeOnce.Do(func() { close(r.removed) })
}
func (r *blockingRenderer) VisibleDebugShapes() []debug.Shape {
	return nil
}
func (r *blockingRenderer) RemoveAllDebugShapes() {}

func TestWireframeAsyncDoesNotBlockCaller(t *testing.T) {
	r := &blockingRenderer{
		entered: make(chan struct{}),
		exited:  make(chan struct{}),
		removed: make(chan struct{}),
		release: make(chan struct{}),
	}
	var w Wireframe

	done := make(chan struct{})
	go func() {
		w.DrawAsync(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}}, color.RGBA{A: 255})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DrawAsync blocked caller")
	}

	select {
	case <-r.entered:
	case <-time.After(time.Second):
		t.Fatal("DrawAsync did not reach renderer")
	}

	removeDone := make(chan struct{})
	go func() {
		w.RemoveAsync(r)
		close(removeDone)
	}()
	select {
	case <-removeDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RemoveAsync blocked caller")
	}
	close(r.release)
	select {
	case <-r.exited:
	case <-time.After(time.Second):
		t.Fatal("DrawAsync renderer call did not exit after release")
	}
	select {
	case <-r.removed:
	case <-time.After(time.Second):
		t.Fatal("RemoveAsync did not remove drawn line")
	}
}
