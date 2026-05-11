package edit

import (
	"path/filepath"
	"testing"

	"github.com/Velvet-MC/s2d/translate"
	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
)

func TestCompactSchematicDeduplicatesPaletteAndPastes(t *testing.T) {
	cs, err := NewCompactSchematic(2, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range []cube.Pos{{0, 0, 0}, {0, 0, 1}, {1, 0, 0}} {
		if err := cs.AddBlock(pos, mcblock.Stone{}, nil); err != nil {
			t.Fatalf("AddBlock stone at %v: %v", pos, err)
		}
	}
	if err := cs.AddBlock(cube.Pos{1, 0, 1}, mcblock.Dirt{}, nil); err != nil {
		t.Fatal(err)
	}

	if cs.PaletteLen() != 2 {
		t.Fatalf("palette len = %d, want 2", cs.PaletteLen())
	}
	if cs.Volume() != 4 {
		t.Fatalf("volume = %d, want 4", cs.Volume())
	}

	withCompactTx(t, func(tx *world.Tx) {
		if err := PasteCompactSchematicNoUndo(tx, cs, cube.Pos{10, 0, 10}, cube.North); err != nil {
			t.Fatalf("PasteCompactSchematicNoUndo: %v", err)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{10, 0, 10}), mcblock.Stone{}) {
			t.Fatalf("expected stone at paste origin, got %T", tx.Block(cube.Pos{10, 0, 10}))
		}
		if !parse.SameBlock(tx.Block(cube.Pos{11, 0, 11}), mcblock.Dirt{}) {
			t.Fatalf("expected dirt at far corner, got %T", tx.Block(cube.Pos{11, 0, 11}))
		}
	})
}

func TestCompactSchematicDirectPaletteIndexes(t *testing.T) {
	cs, err := NewCompactSchematic(2, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	stone := cs.AppendPaletteState(mcblock.Stone{}, nil)
	dirt := cs.AppendPaletteState(mcblock.Dirt{}, nil)
	if stone != 0 || dirt != 1 {
		t.Fatalf("palette indexes = %d/%d, want 0/1", stone, dirt)
	}
	if err := cs.SetPaletteIndex(cube.Pos{0, 0, 0}, stone); err != nil {
		t.Fatal(err)
	}
	if err := cs.SetPaletteIndex(cube.Pos{1, 0, 0}, dirt); err != nil {
		t.Fatal(err)
	}

	if cs.PaletteLen() != 2 {
		t.Fatalf("palette len = %d, want 2", cs.PaletteLen())
	}
	withCompactTx(t, func(tx *world.Tx) {
		if err := PasteCompactSchematicNoUndo(tx, cs, cube.Pos{}, cube.North); err != nil {
			t.Fatal(err)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{0, 0, 0}), mcblock.Stone{}) {
			t.Fatalf("first block = %T, want stone", tx.Block(cube.Pos{0, 0, 0}))
		}
		if !parse.SameBlock(tx.Block(cube.Pos{1, 0, 0}), mcblock.Dirt{}) {
			t.Fatalf("second block = %T, want dirt", tx.Block(cube.Pos{1, 0, 0}))
		}
	})
}

func TestCompactSchematicRotatesPasteAndDirectionalBlocks(t *testing.T) {
	cs, err := NewCompactSchematic(1, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := cs.AddBlock(cube.Pos{0, 0, 0}, mcblock.Furnace{Facing: cube.North}, nil); err != nil {
		t.Fatal(err)
	}
	if err := cs.AddBlock(cube.Pos{0, 0, 1}, mcblock.Dirt{}, nil); err != nil {
		t.Fatal(err)
	}

	withCompactTx(t, func(tx *world.Tx) {
		if err := PasteCompactSchematicNoUndo(tx, cs, cube.Pos{20, 0, 20}, cube.East); err != nil {
			t.Fatalf("PasteCompactSchematicNoUndo: %v", err)
		}
		furnace, ok := tx.Block(cube.Pos{20, 0, 20}).(mcblock.Furnace)
		if !ok || furnace.Facing != cube.East {
			t.Fatalf("rotated furnace = %T/%v, want east-facing furnace", tx.Block(cube.Pos{20, 0, 20}), ok)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{19, 0, 20}), mcblock.Dirt{}) {
			t.Fatalf("rotated dirt missed expected position, got %T", tx.Block(cube.Pos{19, 0, 20}))
		}
	})
}

func TestCompactSchematicPastesLiquidLayerOnNonLiquidBlock(t *testing.T) {
	cs, err := NewCompactSchematic(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	seagrass := translate.NewStateBlock(translate.BedrockState{
		Name:       "minecraft:seagrass",
		Properties: map[string]any{"sea_grass_type": "default"},
	})
	water := mcblock.Water{Depth: 8, Still: true}
	if err := cs.AddBlock(cube.Pos{}, seagrass, water); err != nil {
		t.Fatal(err)
	}

	withCompactTx(t, func(tx *world.Tx) {
		pos := cube.Pos{5, 0, 5}
		if err := PasteCompactSchematicNoUndo(tx, cs, pos, cube.North); err != nil {
			t.Fatalf("PasteCompactSchematicNoUndo: %v", err)
		}
		name, _ := tx.Block(pos).EncodeBlock()
		if name != "minecraft:seagrass" {
			t.Fatalf("block at %v = %s, want minecraft:seagrass", pos, name)
		}
		liq, ok := tx.Liquid(pos)
		if !parse.SameLiquid(liq, ok, water, true) {
			t.Fatalf("liquid at %v = %T/%v, want water", pos, liq, ok)
		}
	})
}

func TestImportJavaCompactSchematic(t *testing.T) {
	cs, report, err := ImportJavaCompactSchematic(filepath.Join("testdata", "single_stone.schem"))
	if err != nil {
		t.Fatal(err)
	}
	if report.Width != 1 || report.Height != 1 || report.Length != 1 {
		t.Fatalf("dimensions = %dx%dx%d, want 1x1x1", report.Width, report.Height, report.Length)
	}
	if cs.Volume() != 1 {
		t.Fatalf("volume = %d, want 1", cs.Volume())
	}
	if cs.PaletteLen() != 1 {
		t.Fatalf("palette len = %d, want 1", cs.PaletteLen())
	}

	withCompactTx(t, func(tx *world.Tx) {
		if err := PasteCompactSchematicNoUndo(tx, cs, cube.Pos{3, 0, 0}, cube.North); err != nil {
			t.Fatalf("PasteCompactSchematicNoUndo: %v", err)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{3, 0, 0}), mcblock.Stone{}) {
			t.Fatalf("imported compact schematic pasted %T, want stone", tx.Block(cube.Pos{3, 0, 0}))
		}
	})
}

func withCompactTx(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	w := world.New()
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("close world: %v", err)
		}
	}()
	<-w.Exec(f)
}
