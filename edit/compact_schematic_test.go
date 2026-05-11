package edit

import (
	"path/filepath"
	"testing"

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
