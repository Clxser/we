package edit_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	_ "unsafe"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

//go:linkname finaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func finaliseBlockRegistry()

func withTx(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	finaliseBlockRegistry()
	w := world.New()
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("close world: %v", err)
		}
	}()
	<-w.Exec(f)
}

func TestFillUndoRedoBatch(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 1, 0, 1)
		area.Range(func(x, y, z int) {
			tx.SetBlock(cube.Pos{x, y, z}, mcblock.Dirt{}, nil)
		})
		tx.SetBlock(cube.Pos{2, 0, 0}, mcblock.Gold{}, nil)

		h := history.NewHistory(10)
		batch := history.NewBatch(false)
		edit.FillArea(tx, area, []world.Block{mcblock.Stone{}}, batch)
		if got := h.Record(batch); got != 1 {
			t.Fatalf("Record() = %d, want 1", got)
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Stone{}) {
				t.Fatalf("fill missed %v", cube.Pos{x, y, z})
			}
		})
		if !parse.SameBlock(tx.Block(cube.Pos{2, 0, 0}), mcblock.Gold{}) {
			t.Fatal("fill changed block outside selection")
		}
		if !h.Undo(tx, false) {
			t.Fatal("Undo returned false")
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Dirt{}) {
				t.Fatalf("undo did not restore %v", cube.Pos{x, y, z})
			}
		})
		if !h.Redo(tx, false) {
			t.Fatal("Redo returned false")
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Stone{}) {
				t.Fatalf("redo did not reapply %v", cube.Pos{x, y, z})
			}
		})
	})
}

func TestClipboardPasteNoAirKeepsExistingBlocks(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Air{}, nil)
		tx.SetBlock(cube.Pos{11, 0, 0}, mcblock.Dirt{}, nil)

		cb := edit.CopySelection(tx, geo.NewArea(0, 0, 0, 1, 0, 0), cube.Pos{0, 0, 0}, cube.North, edit.BlockMask{All: true, IncludeAir: true}, false)
		batch := history.NewBatch(false)
		if err := edit.PasteClipboard(tx, cb, cube.Pos{10, 0, 0}, cube.North, true, batch); err != nil {
			t.Fatalf("PasteClipboard: %v", err)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{10, 0, 0}), mcblock.Stone{}) {
			t.Fatal("non-air clipboard block was not pasted")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{11, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("-a paste overwrote existing block with air")
		}
	})
}

func TestHollowCubeDoesNotOverwriteInterior(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		anchor := cube.Pos{0, 0, 0}
		interior := cube.Pos{0, 1, 0}
		tx.SetBlock(interior, mcblock.Dirt{}, nil)

		batch := history.NewBatch(false)
		edit.ApplyShape(tx, anchor, edit.ShapeSpec{Kind: edit.ShapeCube, Length: 3, Width: 3, Height: 3, Hollow: true}, []world.Block{mcblock.Stone{}}, batch)
		if !parse.SameBlock(tx.Block(interior), mcblock.Dirt{}) {
			t.Fatal("hollow shape overwrote interior block")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{-1, 0, -1}), mcblock.Stone{}) {
			t.Fatal("hollow shape did not place shell block")
		}
	})
}

func TestBiomeChangesAreUndoable(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 0, 0}
		before := tx.Biome(pos)
		var after world.Biome
		for _, b := range world.Biomes() {
			if !parse.SameBiome(before, b) {
				after = b
				break
			}
		}
		if after == nil {
			t.Fatal("expected at least two registered biomes")
		}
		h := history.NewHistory(10)
		batch := history.NewBatch(false)
		batch.SetBiome(tx, pos, after)
		h.Record(batch)
		if !parse.SameBiome(tx.Biome(pos), after) {
			t.Fatal("biome was not set")
		}
		if !h.Undo(tx, false) {
			t.Fatal("Undo returned false")
		}
		if !parse.SameBiome(tx.Biome(pos), before) {
			t.Fatal("undo did not restore biome")
		}
	})
}

func TestSchematicRoundTrip(t *testing.T) {
	store := edit.NewFileSchematicStore(filepath.Join(t.TempDir(), "schematics"))

	withTx(t, func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		cb := edit.CopySelection(tx, geo.NewArea(0, 0, 0, 0, 0, 0), cube.Pos{0, 0, 0}, cube.East, edit.BlockMask{All: true, IncludeAir: true}, false)
		if err := store.Save("one", cb); err != nil {
			t.Fatalf("SaveSchematic: %v", err)
		}
		loaded, err := store.Load("one")
		if err != nil {
			t.Fatalf("LoadSchematic: %v", err)
		}
		batch := history.NewBatch(false)
		if err := edit.PasteClipboard(tx, loaded, cube.Pos{5, 0, 0}, cube.East, false, batch); err != nil {
			t.Fatalf("PasteClipboard: %v", err)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{5, 0, 0}), mcblock.Stone{}) {
			t.Fatal("loaded schematic did not paste expected block")
		}
	})
}

func TestReplaceMaskCanExplicitlyTargetAir(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 1, 0, 0)
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Air{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Dirt{}, nil)
		mask, err := edit.ParseMask("air")
		if err != nil {
			t.Fatalf("ParseMask: %v", err)
		}
		batch := history.NewBatch(false)
		edit.ReplaceArea(tx, area, mask, []world.Block{mcblock.Stone{}}, batch)
		if !parse.SameBlock(tx.Block(cube.Pos{0, 0, 0}), mcblock.Stone{}) {
			t.Fatal("explicit air mask did not replace air")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{1, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("explicit air mask replaced non-air")
		}
	})
}

func TestLineThicknessUsesRequestedWidth(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		for _, thickness := range []int{1, 2, 3, 4} {
			batch := history.NewBatch(false)
			edit.Line(tx, cube.Pos{thickness * 10, 0, 0}, cube.Pos{thickness * 10, 0, 0}, thickness, []world.Block{mcblock.Stone{}}, batch)
			want := thickness * thickness * thickness
			if got := batch.Len(); got != want {
				t.Fatalf("thickness %d changed %d blocks, want %d", thickness, got, want)
			}
		}
	})
}

func TestFileSchematicStoreListAndDelete(t *testing.T) {
	store := edit.NewFileSchematicStore(filepath.Join(t.TempDir(), "schematics"))

	withTx(t, func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		cb := edit.CopySelection(tx, geo.NewArea(0, 0, 0, 0, 0, 0), cube.Pos{0, 0, 0}, cube.North, edit.BlockMask{All: true, IncludeAir: true}, false)
		if err := store.Save("beta", cb); err != nil {
			t.Fatalf("Save beta: %v", err)
		}
		if err := store.Save("alpha", cb); err != nil {
			t.Fatalf("Save alpha: %v", err)
		}
	})

	if err := os.WriteFile(filepath.Join(store.Dir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write non-schematic file: %v", err)
	}
	names, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"alpha", "beta"}) {
		t.Fatalf("names = %v, want [alpha beta]", names)
	}
	if err := store.Delete("alpha"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	names, err = store.List()
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"beta"}) {
		t.Fatalf("names after delete = %v, want [beta]", names)
	}
}

func TestFileSchematicStoreRejectsUnsafeNames(t *testing.T) {
	store := edit.NewFileSchematicStore(t.TempDir())
	if _, err := store.Load("../escape"); err == nil {
		t.Fatal("Load accepted unsafe schematic name")
	}
	if err := store.Delete("bad/name"); err == nil {
		t.Fatal("Delete accepted unsafe schematic name")
	}
}
