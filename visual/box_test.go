package visual

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/we/geo"
	"github.com/go-gl/mathgl/mgl64"
)

func TestAreaBoxUsesExclusiveMaxEdge(t *testing.T) {
	box := AreaBox(geo.NewArea(1, 2, 3, 4, 5, 6))
	if want := (mgl64.Vec3{1, 2, 3}); box.Min != want {
		t.Fatalf("Min = %v, want %v", box.Min, want)
	}
	if want := (mgl64.Vec3{5, 6, 7}); box.Max != want {
		t.Fatalf("Max = %v, want %v", box.Max, want)
	}
}

func TestBlockBoxNormalisesCorners(t *testing.T) {
	box := BlockBox(cube.Pos{4, 5, 6}, cube.Pos{1, 2, 3})
	if want := (mgl64.Vec3{1, 2, 3}); box.Min != want {
		t.Fatalf("Min = %v, want %v", box.Min, want)
	}
	if want := (mgl64.Vec3{5, 6, 7}); box.Max != want {
		t.Fatalf("Max = %v, want %v", box.Max, want)
	}
}

func TestBoxSegmentsReturnsWireframeEdges(t *testing.T) {
	segments := BoxSegments(Box{Min: mgl64.Vec3{1, 2, 3}, Max: mgl64.Vec3{5, 6, 7}})
	if len(segments) != 12 {
		t.Fatalf("segment count = %d, want 12", len(segments))
	}
	if segments[0] != (Segment{Start: mgl64.Vec3{1, 2, 3}, End: mgl64.Vec3{5, 2, 3}}) {
		t.Fatalf("first segment = %v", segments[0])
	}
	if segments[11] != (Segment{Start: mgl64.Vec3{1, 2, 7}, End: mgl64.Vec3{1, 6, 7}}) {
		t.Fatalf("last segment = %v", segments[11])
	}
}
