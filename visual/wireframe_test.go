package visual

import (
	"image/color"
	"math"
	"sync"
	"sync/atomic"
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

type concurrentReadRenderer struct {
	stop    chan struct{}
	done    chan struct{}
	removed atomic.Int32
	sink    atomic.Uint64
}

func (r *concurrentReadRenderer) AddDebugShape(shape debug.Shape) {
	line := shape.(*debug.Line)
	go func() {
		defer close(r.done)
		for {
			select {
			case <-r.stop:
				return
			default:
				r.sink.Add(math.Float64bits(line.Position[0]) ^ math.Float64bits(line.EndPosition[0]) ^ uint64(line.Colour.A))
			}
		}
	}()
}

func (r *concurrentReadRenderer) RemoveDebugShape(debug.Shape) {
	r.removed.Store(1)
	close(r.stop)
}

func (*concurrentReadRenderer) VisibleDebugShapes() []debug.Shape { return nil }
func (*concurrentReadRenderer) RemoveAllDebugShapes()             {}

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
	if len(r.removed) != 2 {
		t.Fatalf("removed after second draw = %d, want 2", len(r.removed))
	}

	w.Remove(r)
	if len(r.removed) != 3 {
		t.Fatalf("removed after remove = %d, want 3", len(r.removed))
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
	if len(r.removed) != 1 {
		t.Fatalf("removed after colour change = %d, want 1", len(r.removed))
	}
}

func TestWireframeDrawDoesNotMutateQueuedLines(t *testing.T) {
	r := &concurrentReadRenderer{stop: make(chan struct{}), done: make(chan struct{})}
	var w Wireframe

	w.Draw(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}}, color.RGBA{A: 255})
	// Use a different renderer so r keeps reading the first queued line while
	// the wireframe swaps to a fresh line for the next draw.
	w.Draw(&debugRenderer{}, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{2, 0, 0}}}, color.RGBA{A: 255})
	w.Remove(r)
	if r.removed.Load() == 0 {
		t.Fatal("renderer did not observe remove")
	}
	select {
	case <-r.done:
	case <-time.After(time.Second):
		t.Fatal("renderer read did not finish")
	}
}

type blockingRenderer struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *blockingRenderer) AddDebugShape(debug.Shape) {
	r.once.Do(func() { close(r.entered) })
	<-r.release
}

func (*blockingRenderer) RemoveDebugShape(debug.Shape)      {}
func (*blockingRenderer) VisibleDebugShapes() []debug.Shape { return nil }
func (*blockingRenderer) RemoveAllDebugShapes()             {}

func TestWireframeDrawAsyncDoesNotBlockOnRenderer(t *testing.T) {
	r := &blockingRenderer{entered: make(chan struct{}), release: make(chan struct{})}
	var w Wireframe

	returned := make(chan struct{})
	go func() {
		w.DrawAsync(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}}, color.RGBA{A: 255})
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		close(r.release)
		t.Fatal("DrawAsync blocked on renderer")
	}

	select {
	case <-r.entered:
		close(r.release)
	case <-time.After(time.Second):
		t.Fatal("DrawAsync did not reach renderer")
	}
}

type controlledRenderer struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	events  chan wireframeEvent
}

type wireframeEvent struct {
	kind string
	end  mgl64.Vec3
}

func (r *controlledRenderer) AddDebugShape(shape debug.Shape) {
	r.once.Do(func() {
		close(r.entered)
		<-r.release
	})
	if line, ok := shape.(*debug.Line); ok {
		r.events <- wireframeEvent{kind: "draw", end: line.EndPosition}
		return
	}
	r.events <- wireframeEvent{kind: "draw"}
}

func (r *controlledRenderer) RemoveDebugShape(debug.Shape) {
	r.events <- wireframeEvent{kind: "remove"}
}

func (*controlledRenderer) VisibleDebugShapes() []debug.Shape { return nil }
func (*controlledRenderer) RemoveAllDebugShapes()             {}

func TestWireframeAsyncRemoveWinsWhileDrawBlocked(t *testing.T) {
	r := &controlledRenderer{entered: make(chan struct{}), release: make(chan struct{}), events: make(chan wireframeEvent, 16)}
	var w Wireframe

	w.DrawAsync(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}}, color.RGBA{A: 255})
	select {
	case <-r.entered:
	case <-time.After(time.Second):
		t.Fatal("DrawAsync did not reach renderer")
	}
	w.RemoveAsync(r)
	close(r.release)

	select {
	case got := <-r.events:
		if got.kind != "draw" || got.end != (mgl64.Vec3{1, 0, 0}) {
			t.Fatalf("first event = %v, want draw", got)
		}
	case <-time.After(time.Second):
		t.Fatal("draw did not complete")
	}
	select {
	case got := <-r.events:
		if got.kind != "remove" {
			t.Fatalf("second event = %v, want remove", got)
		}
	case <-time.After(time.Second):
		t.Fatal("remove did not run after blocked draw")
	}
}

func TestWireframeAsyncCoalescesWhileRendererBlocks(t *testing.T) {
	r := &controlledRenderer{entered: make(chan struct{}), release: make(chan struct{}), events: make(chan wireframeEvent, 16)}
	var w Wireframe

	w.DrawAsync(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{1, 0, 0}}}, color.RGBA{A: 255})
	select {
	case <-r.entered:
	case <-time.After(time.Second):
		t.Fatal("DrawAsync did not reach renderer")
	}

	for i := 0; i < 1000; i++ {
		w.DrawAsync(r, []Segment{{Start: mgl64.Vec3{}, End: mgl64.Vec3{float64(i), 0, 0}}}, color.RGBA{A: 255})
	}
	close(r.release)

	wantEvents := []wireframeEvent{
		{kind: "draw", end: mgl64.Vec3{1, 0, 0}},
		{kind: "remove"},
		{kind: "draw", end: mgl64.Vec3{999, 0, 0}},
	}
	for _, want := range wantEvents {
		select {
		case got := <-r.events:
			if got != want {
				t.Fatalf("event = %v, want %v", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %v", want)
		}
	}
	select {
	case got := <-r.events:
		t.Fatalf("unexpected non-coalesced event %v", got)
	case <-time.After(50 * time.Millisecond):
	}

	deadline := time.After(time.Second)
	for {
		w.asyncMu.Lock()
		running := w.asyncRunning
		w.asyncMu.Unlock()
		if !running {
			return
		}
		select {
		case <-deadline:
			t.Fatal("async worker did not exit after queue drained")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
