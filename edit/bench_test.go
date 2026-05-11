package edit_test

import (
	"path/filepath"
	"testing"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
)

func newBenchWorld(b *testing.B) *world.World {
	b.Helper()
	w := world.New()
	b.Cleanup(func() {
		if err := w.Close(); err != nil {
			b.Fatalf("close world: %v", err)
		}
	})
	return w
}

func execBenchTx(b *testing.B, w *world.World, f func(tx *world.Tx)) {
	b.Helper()
	<-w.Exec(f)
}

func fillBenchArea(tx *world.Tx, area geo.Area) {
	area.Range(func(x, y, z int) {
		tx.SetBlock(cube.Pos{x, y, z}, mcblock.Stone{}, nil)
	})
}

func BenchmarkFillArea(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 15, 7, 15)
	blocks := []world.Block{mcblock.Stone{}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			batch := history.NewBatch(false)
			edit.FillArea(tx, area, blocks, batch)
		})
	}
}

func BenchmarkFillAreaNoHistory(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 15, 7, 15)
	blocks := []world.Block{mcblock.Stone{}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			edit.FillArea(tx, area, blocks, nil)
		})
	}
}

func BenchmarkFillAreaNoHistoryMultiBlock(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 15, 7, 15)
	blocks := []world.Block{mcblock.Stone{}, mcblock.Dirt{}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			edit.FillArea(tx, area, blocks, nil)
		})
	}
}

func BenchmarkCopySelection(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 7, 3, 7)
	execBenchTx(b, w, func(tx *world.Tx) { fillBenchArea(tx, area) })

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			_ = edit.CopySelection(tx, area, area.Min, cube.North, edit.BlockMask{All: true, IncludeAir: true}, false)
		})
	}
}

func BenchmarkPasteClipboard(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 7, 3, 7)
	var cb *edit.Clipboard
	execBenchTx(b, w, func(tx *world.Tx) {
		fillBenchArea(tx, area)
		cb = edit.CopySelection(tx, area, area.Min, cube.North, edit.BlockMask{All: true, IncludeAir: true}, false)
	})

	b.ReportAllocs()
	b.ResetTimer()
	var pasteErr error
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			batch := history.NewBatch(false)
			pasteErr = edit.PasteClipboard(tx, cb, cube.Pos{16, 0, 0}, cube.North, false, batch)
		})
		if pasteErr != nil {
			b.Fatal(pasteErr)
		}
	}
}

func BenchmarkReplaceAreaMask(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 15, 7, 15)
	execBenchTx(b, w, func(tx *world.Tx) {
		area.Range(func(x, y, z int) {
			block := world.Block(mcblock.Stone{})
			if (x+y+z)%3 == 0 {
				block = mcblock.Dirt{}
			}
			tx.SetBlock(cube.Pos{x, y, z}, block, nil)
		})
	})
	mask := edit.BlockMask{Blocks: []world.Block{mcblock.Stone{}, mcblock.Dirt{}}}
	to := []world.Block{mcblock.Gold{}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			batch := history.NewBatch(false)
			edit.ReplaceArea(tx, area, mask, to, batch)
		})
	}
}

func BenchmarkStack(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 7, 3, 7)
	execBenchTx(b, w, func(tx *world.Tx) { fillBenchArea(tx, area) })

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			batch := history.NewBatch(false)
			edit.Stack(tx, area, cube.Pos{1, 0, 0}, 4, false, batch)
		})
	}
}

func BenchmarkApplyShapeSphere(b *testing.B) {
	w := newBenchWorld(b)
	spec := edit.ShapeSpec{Kind: edit.ShapeSphere, Radius: 6, Height: 13}
	blocks := []world.Block{mcblock.Stone{}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execBenchTx(b, w, func(tx *world.Tx) {
			batch := history.NewBatch(false)
			edit.ApplyShape(tx, cube.Pos{0, 64, 0}, spec, blocks, batch)
		})
	}
}

func BenchmarkSchematicSaveLoad(b *testing.B) {
	w := newBenchWorld(b)
	area := geo.NewArea(0, 0, 0, 5, 3, 5)
	var cb *edit.Clipboard
	execBenchTx(b, w, func(tx *world.Tx) {
		fillBenchArea(tx, area)
		cb = edit.CopySelection(tx, area, area.Min, cube.North, edit.BlockMask{All: true, IncludeAir: true}, false)
	})
	store := edit.NewFileSchematicStore(filepath.Join(b.TempDir(), "schematics"))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.Save("bench", cb); err != nil {
			b.Fatal(err)
		}
		loaded, err := store.Load("bench")
		if err != nil {
			b.Fatal(err)
		}
		if len(loaded.Entries) != len(cb.Entries) {
			b.Fatalf("loaded %d entries, want %d", len(loaded.Entries), len(cb.Entries))
		}
	}
}
