package edit

import (
	"testing"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/we/parse"
)

func TestMakeRotatedDenseBufferMapsOriginalDenseOrder(t *testing.T) {
	blocks := []bufferEntry{
		{Offset: cube.Pos{0, 0, 0}, Block: mcblock.Stone{}},
		{Offset: cube.Pos{0, 0, 1}, Block: mcblock.Dirt{}},
		{Offset: cube.Pos{0, 0, 2}, Block: mcblock.Gold{}},
		{Offset: cube.Pos{1, 0, 0}, Block: mcblock.Diamond{}},
		{Offset: cube.Pos{1, 0, 1}, Block: mcblock.Emerald{}},
		{Offset: cube.Pos{1, 0, 2}, Block: mcblock.Iron{}},
	}

	layout, ok := makeRotatedDenseBuffer(blocks, 1)
	if !ok {
		t.Fatal("dense source was not accepted")
	}
	if layout.min != (cube.Pos{-2, 0, 0}) {
		t.Fatalf("min = %v, want (-2,0,0)", layout.min)
	}
	if layout.dims != [3]int{3, 1, 2} {
		t.Fatalf("dims = %v, want [3 1 2]", layout.dims)
	}
	structure := rotatedBufferDenseStructure{min: layout.min, d: layout.dims, source: layout.source, turns: layout.turns}

	// A right rotation maps original (x,z) to (-z,x). Local x is shifted by
	// min.x=-2, so original (1,0,2) lands at local (0,0,1).
	got, _ := structure.At(0, 0, 1, nil)
	if !parse.SameBlock(got, mcblock.Iron{}) {
		t.Fatalf("rotated At(0,0,1) = %T, want Iron", got)
	}
	got, _ = structure.At(2, 0, 0, nil)
	if !parse.SameBlock(got, mcblock.Stone{}) {
		t.Fatalf("rotated At(2,0,0) = %T, want Stone", got)
	}
}
