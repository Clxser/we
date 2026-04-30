package history_test

import (
	"testing"
	_ "unsafe"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
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

func TestBrushHistoryIsIsolatedFromMainHistory(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 0, 0}
		h := history.NewHistory(10)
		mainBatch := history.NewBatch(false)
		mainBatch.SetBlock(tx, pos, mcblock.Stone{})
		h.Record(mainBatch)
		brushBatch := history.NewBatch(true)
		brushBatch.SetBlock(tx, pos, mcblock.Gold{})
		h.Record(brushBatch)

		if !h.Undo(tx, false) {
			t.Fatal("main undo returned false")
		}
		if !parse.SameBlock(tx.Block(pos), mcblock.Air{}) {
			t.Fatal("main undo did not skip brush stack and restore main before-state")
		}
		if !h.Undo(tx, true) {
			t.Fatal("brush undo returned false")
		}
		if !parse.SameBlock(tx.Block(pos), mcblock.Stone{}) {
			t.Fatal("brush undo did not restore brush before-state")
		}
	})
}
