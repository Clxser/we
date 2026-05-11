package history_test

import (
	"testing"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

func withTx(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	w := world.New()
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("close world: %v", err)
		}
	}()
	<-w.Exec(f)
}

func TestDefaultUndoRedoUsesMostRecentCommandOrBrushBatch(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		mainPos := cube.Pos{0, 0, 0}
		brushPos := cube.Pos{1, 0, 0}
		h := history.NewHistory(10)
		mainBatch := history.NewBatch(false)
		mainBatch.SetBlock(tx, mainPos, mcblock.Stone{})
		h.Record(mainBatch)
		brushBatch := history.NewBatch(true)
		brushBatch.SetBlock(tx, brushPos, mcblock.Gold{})
		h.Record(brushBatch)

		if !h.Undo(tx, false) {
			t.Fatal("default undo returned false")
		}
		if !parse.SameBlock(tx.Block(brushPos), mcblock.Air{}) {
			t.Fatal("default undo did not undo latest brush batch")
		}
		if !parse.SameBlock(tx.Block(mainPos), mcblock.Stone{}) {
			t.Fatal("default undo changed older command batch first")
		}
		if !h.Undo(tx, false) {
			t.Fatal("second default undo returned false")
		}
		if !parse.SameBlock(tx.Block(mainPos), mcblock.Air{}) {
			t.Fatal("second default undo did not undo command batch")
		}
		if !h.Redo(tx, false) {
			t.Fatal("default redo returned false")
		}
		if !parse.SameBlock(tx.Block(mainPos), mcblock.Stone{}) {
			t.Fatal("default redo did not redo most recently undone command batch")
		}
		if !h.Redo(tx, false) {
			t.Fatal("second default redo returned false")
		}
		if !parse.SameBlock(tx.Block(brushPos), mcblock.Gold{}) {
			t.Fatal("second default redo did not redo brush batch")
		}
	})
}

func TestExplicitBrushUndoStillTargetsBrushStack(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		mainPos := cube.Pos{0, 0, 0}
		brushPos := cube.Pos{1, 0, 0}
		h := history.NewHistory(10)
		brushBatch := history.NewBatch(true)
		brushBatch.SetBlock(tx, brushPos, mcblock.Gold{})
		h.Record(brushBatch)
		mainBatch := history.NewBatch(false)
		mainBatch.SetBlock(tx, mainPos, mcblock.Stone{})
		h.Record(mainBatch)

		if !h.Undo(tx, true) {
			t.Fatal("explicit brush undo returned false")
		}
		if !parse.SameBlock(tx.Block(brushPos), mcblock.Air{}) {
			t.Fatal("explicit brush undo did not undo brush batch")
		}
		if !parse.SameBlock(tx.Block(mainPos), mcblock.Stone{}) {
			t.Fatal("explicit brush undo changed command batch")
		}
		if !h.Redo(tx, true) {
			t.Fatal("explicit brush redo returned false")
		}
		if !parse.SameBlock(tx.Block(brushPos), mcblock.Gold{}) {
			t.Fatal("explicit brush redo did not redo brush batch")
		}
	})
}

func TestRecordClearsCommandAndBrushRedoStacks(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		commandPos := cube.Pos{0, 0, 0}
		brushPos := cube.Pos{1, 0, 0}
		newPos := cube.Pos{2, 0, 0}
		h := history.NewHistory(10)

		command := history.NewBatch(false)
		command.SetBlock(tx, commandPos, mcblock.Stone{})
		h.Record(command)
		brush := history.NewBatch(true)
		brush.SetBlock(tx, brushPos, mcblock.Gold{})
		h.Record(brush)

		undidLatest := h.Undo(tx, false)
		undidNext := h.Undo(tx, false)
		if !undidLatest || !undidNext {
			t.Fatal("expected command and brush undo to succeed")
		}

		next := history.NewBatch(false)
		next.SetBlock(tx, newPos, mcblock.Dirt{})
		h.Record(next)

		if h.Redo(tx, false) {
			t.Fatal("default redo succeeded after new edit")
		}
		if h.Redo(tx, true) {
			t.Fatal("brush redo succeeded after new edit")
		}
	})
}

func TestRecordReturnsChangedCountAndSkipsNoOps(t *testing.T) {
	var failure string
	withTx(t, func(tx *world.Tx) {
		h := history.NewHistory(10)
		batch := history.NewBatch(false)
		batch.SetBlock(tx, cube.Pos{0, 0, 0}, mcblock.Stone{})
		batch.SetBlock(tx, cube.Pos{1, 0, 0}, mcblock.Dirt{})
		batch.SetBlock(tx, cube.Pos{2, 0, 0}, mcblock.Air{})

		if got := h.Record(batch); got != 2 {
			failure = "Record() changed count mismatch"
			return
		}
		if got := h.Record(history.NewBatch(false)); got != 0 {
			failure = "Record() stored empty batch"
		}
	})
	if failure != "" {
		t.Fatal(failure)
	}
}
