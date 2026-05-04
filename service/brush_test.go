package service_test

import (
	"strings"
	"testing"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
	"github.com/df-mc/we/service"
)

func TestApplyBrushRejectsLargeBrush(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		batch := history.NewBatch(true)
		err := service.ApplyBrush(tx, service.BrushActor{}, cube.Pos{0, 0, 0}, service.BrushConfig{
			Type:   "cube",
			Length: 2,
			Width:  1,
			Height: 1,
		}, edit.DefaultSchematicStore(), guardrail.Limits{MaxBrushVolume: 1}, batch)
		if err == nil || !strings.Contains(err.Error(), "brush volume 2 exceeds limit 1") {
			t.Fatalf("ApplyBrush error = %v, want brush limit error", err)
		}
		if batch.Len() != 0 {
			t.Fatalf("batch Len = %d, want 0", batch.Len())
		}
	})
}

func TestApplyBrushAndRecordUsesBrushHistory(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		target := cube.Pos{0, 0, 0}
		err := service.ApplyBrushAndRecord(tx, s, service.BrushActor{}, target, service.BrushConfig{
			Type:   service.BrushCube,
			Length: 1,
			Width:  1,
			Height: 1,
		}, edit.DefaultSchematicStore(), guardrail.Limits{})
		if err != nil {
			t.Fatalf("ApplyBrushAndRecord error = %v", err)
		}
		if !parse.SameBlock(tx.Block(target), mcblock.Stone{}) {
			t.Fatal("brush did not place the expected block")
		}
		if err := service.Undo(tx, s, true); err != nil {
			t.Fatalf("brush undo error = %v", err)
		}
		if !parse.IsAir(tx.Block(target)) {
			t.Fatal("brush edit was not recorded on the brush history stack")
		}
	})
}

func TestBrushAnchorFromSurfaceExtendsShapeOutward(t *testing.T) {
	cfg := service.BrushConfig{Type: service.BrushSphere, Radius: 3, Height: 7}
	tests := []struct {
		name    string
		surface cube.Pos
		face    cube.Face
		want    cube.Pos
	}{
		{name: "up", surface: cube.Pos{10, 65, 10}, face: cube.FaceUp, want: cube.Pos{10, 68, 10}},
		{name: "down", surface: cube.Pos{10, 63, 10}, face: cube.FaceDown, want: cube.Pos{10, 60, 10}},
		{name: "east", surface: cube.Pos{11, 64, 10}, face: cube.FaceEast, want: cube.Pos{14, 64, 10}},
		{name: "west", surface: cube.Pos{9, 64, 10}, face: cube.FaceWest, want: cube.Pos{6, 64, 10}},
		{name: "south", surface: cube.Pos{10, 64, 11}, face: cube.FaceSouth, want: cube.Pos{10, 64, 14}},
		{name: "north", surface: cube.Pos{10, 64, 9}, face: cube.FaceNorth, want: cube.Pos{10, 64, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.BrushAnchorFromSurface(tt.surface, tt.face, cfg); got != tt.want {
				t.Fatalf("BrushAnchorFromSurface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrushAnchorFromSurfaceLeavesLineAtSurface(t *testing.T) {
	cfg := service.BrushConfig{Type: service.BrushLine, Radius: 3, Height: 7}
	surface := cube.Pos{10, 65, 10}
	if got := service.BrushAnchorFromSurface(surface, cube.FaceUp, cfg); got != surface {
		t.Fatalf("BrushAnchorFromSurface() = %v, want %v", got, surface)
	}
}

func TestBrushAnchorFromSurfacePaintTargetsHitBlock(t *testing.T) {
	surface := cube.Pos{10, 63, 10}
	want := cube.Pos{10, 64, 10}
	got := service.BrushAnchorFromSurface(surface, cube.FaceDown, service.BrushConfig{Type: service.BrushPaint, Radius: 1, Height: 3})
	if got != want {
		t.Fatalf("BrushAnchorFromSurface() = %v, want hit block %v", got, want)
	}
}

func TestPaintBrushCanDrawOnUnderside(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		target := cube.Pos{0, 0, 0}
		tx.SetBlock(target, mcblock.Stone{}, nil)
		batch := history.NewBatch(true)
		err := service.ApplyBrush(tx, service.BrushActor{}, target, service.BrushConfig{
			Type:     service.BrushPaint,
			Shape:    service.BrushSphere,
			Radius:   1,
			Height:   3,
			Strength: 1,
			Blocks:   service.StatesFromBlocks([]world.Block{mcblock.Gold{}}),
		}, edit.DefaultSchematicStore(), guardrail.Limits{}, batch)
		if err != nil {
			t.Fatalf("ApplyBrush: %v", err)
		}
		if !parse.SameBlock(tx.Block(target), mcblock.Gold{}) {
			t.Fatalf("paint brush left underside target as %#v, want gold", tx.Block(target))
		}
	})
}

func TestBrushVolumeBounds(t *testing.T) {
	area, ok := service.BrushVolumeBounds(cube.Pos{10, 20, 30}, service.BrushConfig{
		Type:   service.BrushCube,
		Length: 3,
		Width:  5,
		Height: 2,
	})
	if !ok {
		t.Fatal("BrushVolumeBounds ok = false, want true")
	}
	if area.Min != (cube.Pos{9, 20, 28}) || area.Max != (cube.Pos{11, 21, 32}) {
		t.Fatalf("BrushVolumeBounds = %v-%v", area.Min, area.Max)
	}
	if _, ok := service.BrushVolumeBounds(cube.Pos{10, 20, 30}, service.BrushConfig{Type: service.BrushLine}); ok {
		t.Fatal("BrushVolumeBounds ok = true for line brush, want false")
	}
}
