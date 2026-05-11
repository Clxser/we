package edit

import (
	"testing"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
)

func TestPasteClipboardConsumingKeepsRotatedDenseEntriesInPlaceOrder(t *testing.T) {
	cb := &Clipboard{OriginDir: cube.North}
	for x := 0; x < 2; x++ {
		for y := 0; y < 1; y++ {
			for z := 0; z < 3; z++ {
				cb.Entries = append(cb.Entries, bufferEntry{
					Offset: cube.Pos{x, y, z},
					Block:  mcblock.Stone{},
				})
			}
		}
	}

	w := world.New()
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("close world: %v", err)
		}
	}()
	<-w.Exec(func(tx *world.Tx) {
		if err := PasteClipboardConsuming(tx, cb, cube.Pos{}, cube.East, false, nil); err != nil {
			t.Fatalf("PasteClipboardConsuming: %v", err)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{-2, 0, 0}), mcblock.Stone{}) {
			t.Fatalf("rotated dense paste missed expected block at (-2,0,0)")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{0, 0, 1}), mcblock.Stone{}) {
			t.Fatalf("rotated dense paste missed expected block at (0,0,1)")
		}
	})

	layout, ok := makeDenseBuffer(cb.Entries)
	if !ok {
		t.Fatal("rotated consuming clipboard is no longer dense")
	}
	if len(layout.ordered) == 0 || &layout.ordered[0] != &cb.Entries[0] {
		t.Fatal("rotated consuming clipboard was not reordered in place for dense zero-copy paste")
	}
}
