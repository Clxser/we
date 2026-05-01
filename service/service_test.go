package service_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	_ "unsafe"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
	"github.com/df-mc/we/service"
)

//go:linkname finaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func finaliseBlockRegistry()

type fakeSession struct {
	area       geo.Area
	hasArea    bool
	pos1, pos2 cube.Pos
	hasCorners bool
	clipboard  *edit.Clipboard
	guardrails guardrail.Limits
	history    *history.History
}

func newFakeSession(area geo.Area) *fakeSession {
	return &fakeSession{area: area, hasArea: true, pos1: area.Min, pos2: area.Max, hasCorners: true, history: history.NewHistory(10)}
}

func (s *fakeSession) SelectionArea() (geo.Area, bool) { return s.area, s.hasArea }
func (s *fakeSession) PosCorners() (cube.Pos, cube.Pos, bool) {
	return s.pos1, s.pos2, s.hasCorners
}
func (s *fakeSession) SetClipboard(c *edit.Clipboard) { s.clipboard = c }
func (s *fakeSession) Clipboard() (*edit.Clipboard, bool) {
	return s.clipboard, s.clipboard != nil
}
func (s *fakeSession) Guardrails() guardrail.Limits       { return s.guardrails }
func (s *fakeSession) Record(batch *history.Batch) int    { return s.history.Record(batch) }
func (s *fakeSession) Undo(tx *world.Tx, brush bool) bool { return s.history.Undo(tx, brush) }
func (s *fakeSession) Redo(tx *world.Tx, brush bool) bool { return s.history.Redo(tx, brush) }

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

func TestSetRequiresSelection(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		_, err := service.Set(tx, &fakeSession{history: history.NewHistory(10)}, "stone")
		if !errors.Is(err, service.ErrSelectionRequired) {
			t.Fatalf("Set error = %v, want ErrSelectionRequired", err)
		}
	})
}

func TestSetRecordsUndoableChanges(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 1, 0, 0)
		s := newFakeSession(area)
		area.Range(func(x, y, z int) { tx.SetBlock(cube.Pos{x, y, z}, mcblock.Dirt{}, nil) })

		result, err := service.Set(tx, s, "stone")
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want 2", result.Changed)
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Stone{}) {
				t.Fatalf("block %v was not set", cube.Pos{x, y, z})
			}
		})
		if err := service.Undo(tx, s, false); err != nil {
			t.Fatal(err)
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Dirt{}) {
				t.Fatalf("block %v was not restored", cube.Pos{x, y, z})
			}
		})
	})
}

func TestCopyPasteNoAirKeepsExistingBlocks(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 1, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Air{}, nil)
		tx.SetBlock(cube.Pos{11, 0, 0}, mcblock.Dirt{}, nil)

		copyResult, err := service.Copy(tx, s, cube.Pos{0, 0, 0}, cube.North, nil)
		if err != nil {
			t.Fatal(err)
		}
		if copyResult.Copied != 2 {
			t.Fatalf("copied = %d, want 2", copyResult.Copied)
		}
		pasteResult, err := service.Paste(tx, s, cube.Pos{10, 0, 0}, cube.North, []string{"-a"})
		if err != nil {
			t.Fatal(err)
		}
		if pasteResult.Changed != 1 {
			t.Fatalf("changed = %d, want 1", pasteResult.Changed)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{10, 0, 0}), mcblock.Stone{}) {
			t.Fatal("solid clipboard block was not pasted")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{11, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("air clipboard block overwrote destination despite -a")
		}
	})
}

func TestSelectionGuardrailRejectsLargeSelection(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 1, 0, 0))
		s.guardrails = guardrail.Limits{MaxSelectionVolume: 1}
		_, err := service.Set(tx, s, "stone")
		if err == nil || !strings.Contains(err.Error(), "selection volume 2 exceeds limit 1") {
			t.Fatalf("Set error = %v, want selection limit error", err)
		}
	})
}

func TestShapeGuardrailRejectsLargeShape(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		s.guardrails = guardrail.Limits{MaxShapeVolume: 1}
		_, err := service.Shape(tx, s, cube.Pos{0, 0, 0}, edit.ShapeCube, []string{"stone", "2", "1", "1"})
		if err == nil || !strings.Contains(err.Error(), "shape volume 2 exceeds limit 1") {
			t.Fatalf("Shape error = %v, want shape limit error", err)
		}
	})
}

func TestStackGuardrailRejectsTooManyCopies(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		s.guardrails = guardrail.Limits{MaxStackCopies: 2}
		_, err := service.Stack(tx, s, cube.Pos{1, 0, 0}, []string{"3"})
		if err == nil || !strings.Contains(err.Error(), "stack copies 3 exceeds limit 2") {
			t.Fatalf("Stack error = %v, want stack copy limit error", err)
		}
	})
}

func TestSchematicRoundTrip(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		store := edit.NewFileSchematicStore(filepath.Join(t.TempDir(), "schematics"))

		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		created, err := service.Schematic(tx, s, cube.Pos{0, 0, 0}, cube.North, store, []string{"create", "one"})
		if err != nil {
			t.Fatal(err)
		}
		if created.Name != "one" {
			t.Fatalf("created name = %q, want one", created.Name)
		}
		pasted, err := service.Schematic(tx, s, cube.Pos{5, 0, 0}, cube.North, store, []string{"paste", "one"})
		if err != nil {
			t.Fatal(err)
		}
		if pasted.Changed != 1 {
			t.Fatalf("pasted changed = %d, want 1", pasted.Changed)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{5, 0, 0}), mcblock.Stone{}) {
			t.Fatal("schematic paste did not restore saved block")
		}
	})
}
