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
	records    int
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
func (s *fakeSession) Guardrails() guardrail.Limits { return s.guardrails }
func (s *fakeSession) Record(batch *history.Batch) int {
	s.records++
	return s.history.Record(batch)
}
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

func TestSetSubChunkGuardrailRejectsLargeSelection(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 16, 0, 0))
		s.guardrails = guardrail.Limits{MaxEditSubChunks: 1}

		if _, err := service.Set(tx, s, "stone"); err == nil || !strings.Contains(err.Error(), "edit touches 2 sub-chunks") {
			t.Fatalf("Set error = %v, want edit sub-chunk limit", err)
		}
	})
}

func TestSetWithNoUndoWritesWithoutRecording(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 1, 0, 0)
		s := newFakeSession(area)
		area.Range(func(x, y, z int) { tx.SetBlock(cube.Pos{x, y, z}, mcblock.Dirt{}, nil) })

		result, err := service.SetWithOptions(tx, s, "stone", service.EditOptions{NoUndo: true})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want selected volume 2", result.Changed)
		}
		if s.records != 0 {
			t.Fatalf("Record called %d times, want 0", s.records)
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Stone{}) {
				t.Fatalf("block %v was not set", cube.Pos{x, y, z})
			}
		})
		if err := service.Undo(tx, s, false); !errors.Is(err, service.ErrNothingToUndo) {
			t.Fatalf("Undo error = %v, want ErrNothingToUndo", err)
		}
		area.Range(func(x, y, z int) {
			if !parse.SameBlock(tx.Block(cube.Pos{x, y, z}), mcblock.Stone{}) {
				t.Fatalf("undo reverted no-undo set at %v", cube.Pos{x, y, z})
			}
		})
	})
}

func TestSetWithNoUndoPreservesOlderUndoHistory(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 0, 0, 0)
		s := newFakeSession(area)
		pos := cube.Pos{0, 0, 0}
		tx.SetBlock(pos, mcblock.Dirt{}, nil)

		if _, err := service.Set(tx, s, "stone"); err != nil {
			t.Fatal(err)
		}
		if _, err := service.SetWithOptions(tx, s, "gold_block", service.EditOptions{NoUndo: true}); err != nil {
			t.Fatal(err)
		}
		if err := service.Undo(tx, s, false); err != nil {
			t.Fatal(err)
		}
		if !parse.SameBlock(tx.Block(pos), mcblock.Dirt{}) {
			t.Fatalf("undo did not restore older batch, got %T", tx.Block(pos))
		}
	})
}

func TestSetWithNoUndoLiquidWrite(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 0, 0}
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		tx.SetBlock(pos, mcblock.Dirt{}, nil)

		if _, err := service.SetWithOptions(tx, s, "water", service.EditOptions{NoUndo: true}); err != nil {
			t.Fatal(err)
		}
		water := mcblock.Water{Depth: 8, Still: true}
		liq, ok := tx.Liquid(pos)
		if !parse.SameBlock(tx.Block(pos), water) || !parse.SameLiquid(liq, ok, water, true) {
			t.Fatalf("liquid state = block %T liquid %T/%v, want water", tx.Block(pos), liq, ok)
		}
	})
}

func TestCutWithNoUndoWritesClipboardWithoutRecording(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 1, 0, 0)
		s := newFakeSession(area)
		area.Range(func(x, y, z int) { tx.SetBlock(cube.Pos{x, y, z}, mcblock.Stone{}, nil) })

		result, err := service.CutWithOptions(tx, s, cube.Pos{}, cube.North, service.EditOptions{NoUndo: true})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want selected volume 2", result.Changed)
		}
		if s.records != 0 {
			t.Fatalf("Record called %d times, want 0", s.records)
		}
		if cb, ok := s.Clipboard(); !ok || len(cb.Entries) != 2 {
			t.Fatalf("clipboard entries = %d/%v, want 2/true", len(cb.Entries), ok)
		}
		area.Range(func(x, y, z int) {
			if !parse.IsAir(tx.Block(cube.Pos{x, y, z})) {
				t.Fatalf("cut did not clear %v", cube.Pos{x, y, z})
			}
		})
		if err := service.Undo(tx, s, false); !errors.Is(err, service.ErrNothingToUndo) {
			t.Fatalf("Undo error = %v, want ErrNothingToUndo", err)
		}
	})
}

func TestReplaceWithNoUndoWritesWithoutRecording(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		area := geo.NewArea(0, 0, 0, 1, 0, 0)
		s := newFakeSession(area)
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Dirt{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Stone{}, nil)

		result, err := service.Replace(tx, s, []string{"dirt", "gold_block", "-noundo"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want selected volume 2", result.Changed)
		}
		if s.records != 0 {
			t.Fatalf("Record called %d times, want 0", s.records)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{0, 0, 0}), mcblock.Gold{}) {
			t.Fatalf("matching block was not replaced, got %T", tx.Block(cube.Pos{0, 0, 0}))
		}
		if !parse.SameBlock(tx.Block(cube.Pos{1, 0, 0}), mcblock.Stone{}) {
			t.Fatalf("non-matching block changed, got %T", tx.Block(cube.Pos{1, 0, 0}))
		}
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

func TestPasteSubChunkGuardrailRejectsSparsePaste(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		source := geo.NewArea(0, 0, 0, 16, 0, 0)
		s := newFakeSession(source)
		s.guardrails = guardrail.Limits{MaxEditSubChunks: 1}
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{16, 0, 0}, mcblock.Stone{}, nil)
		if _, err := service.Copy(tx, s, cube.Pos{}, cube.North, []string{"only", "stone"}); err != nil {
			t.Fatal(err)
		}

		if _, err := service.Paste(tx, s, cube.Pos{0, 0, 16}, cube.North, nil); err == nil || !strings.Contains(err.Error(), "edit touches 2 sub-chunks") {
			t.Fatalf("Paste error = %v, want paste sub-chunk limit", err)
		}
	})
}

func TestPasteSubChunkGuardrailHonoursNoAir(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		source := geo.NewArea(0, 0, 0, 16, 0, 0)
		s := newFakeSession(source)
		s.guardrails = guardrail.Limits{MaxEditSubChunks: 1}
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		if _, err := service.Copy(tx, s, cube.Pos{}, cube.North, nil); err != nil {
			t.Fatal(err)
		}

		if _, err := service.Paste(tx, s, cube.Pos{0, 0, 16}, cube.North, []string{"-a"}); err != nil {
			t.Fatalf("Paste with -a error = %v, want air entries ignored by guardrail", err)
		}
	})
}

func TestCopyPasteAnchorsSelectionAtCenter(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(-1, 0, 0, 1, 0, 0))
		tx.SetBlock(cube.Pos{-1, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Gold{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Dirt{}, nil)

		if _, err := service.Copy(tx, s, cube.Pos{-1, 0, 0}, cube.North, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Paste(tx, s, cube.Pos{10, 0, 0}, cube.North, []string{"-a"}); err != nil {
			t.Fatal(err)
		}

		if !parse.SameBlock(tx.Block(cube.Pos{9, 0, 0}), mcblock.Stone{}) {
			t.Fatal("left side did not paste one block before target")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{10, 0, 0}), mcblock.Gold{}) {
			t.Fatal("selection center did not paste at target")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{11, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("right side did not paste one block after target")
		}
	})
}

func TestCopyPasteEvenSelectionAnchorIsStableForNegativeCoordinates(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(-2, 0, 0, -1, 0, 0))
		tx.SetBlock(cube.Pos{-2, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{-1, 0, 0}, mcblock.Dirt{}, nil)

		if _, err := service.Copy(tx, s, cube.Pos{-2, 0, 0}, cube.North, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Paste(tx, s, cube.Pos{10, 0, 0}, cube.North, []string{"-a"}); err != nil {
			t.Fatal(err)
		}

		if !parse.SameBlock(tx.Block(cube.Pos{10, 0, 0}), mcblock.Stone{}) {
			t.Fatal("lower-middle block did not paste at target")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{11, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("upper block did not paste after target")
		}
		if !parse.IsAir(tx.Block(cube.Pos{9, 0, 0})) {
			t.Fatal("negative even-width selection pasted one block before target")
		}
	})
}

func TestClearClipboardRemovesStoredClipboard(t *testing.T) {
	var hasClipboard bool
	var err error
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		_, err = service.Copy(tx, s, cube.Pos{}, cube.North, nil)
		if err != nil {
			return
		}
		service.ClearClipboard(s)
		_, hasClipboard = s.Clipboard()
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasClipboard {
		t.Fatal("clipboard was not cleared")
	}
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

func TestReplaceOnlyMatchingBlocks(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 1, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Dirt{}, nil)

		result, err := service.Replace(tx, s, []string{"stone", "gold_block"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 1 {
			t.Fatalf("changed = %d, want 1", result.Changed)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{0, 0, 0}), mcblock.Gold{}) {
			t.Fatal("matching block was not replaced")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{1, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("non-matching block was replaced")
		}
	})
}

func TestOverlayCanPlaceAboveSelectionMaxY(t *testing.T) {
	var changed int
	var placed bool
	var err error
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 80, 0, 0, 80, 0))
		tx.SetBlock(cube.Pos{0, 80, 0}, mcblock.Dirt{}, nil)
		result, overlayErr := service.Overlay(tx, s, "stone")
		changed, err = result.Changed, overlayErr
		placed = parse.SameBlock(tx.Block(cube.Pos{0, 81, 0}), mcblock.Stone{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	if !placed {
		t.Fatal("overlay did not place above selected max Y")
	}
}

func TestOverlayStillUsesSelectedColumnWhenHigherBlocksExist(t *testing.T) {
	var changed int
	var placedInSelection bool
	var untouchedAbove bool
	var err error
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Dirt{}, nil)
		tx.SetBlock(cube.Pos{0, 10, 0}, mcblock.Gold{}, nil)
		result, overlayErr := service.Overlay(tx, s, "stone")
		changed, err = result.Changed, overlayErr
		placedInSelection = parse.SameBlock(tx.Block(cube.Pos{0, 1, 0}), mcblock.Stone{})
		untouchedAbove = parse.SameBlock(tx.Block(cube.Pos{0, 10, 0}), mcblock.Gold{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 || !placedInSelection || !untouchedAbove {
		t.Fatalf("overlay changed=%d placed=%v untouchedAbove=%v, want selected placement only", changed, placedInSelection, untouchedAbove)
	}
}

func TestRemoveAboveBelowAndNear(t *testing.T) {
	var err error
	var aboveAir, belowAir, nearAir, farKept bool
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		center := cube.Pos{0, 10, 0}
		tx.SetBlock(cube.Pos{0, 11, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{0, 9, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 10, 0}, mcblock.Gold{}, nil)
		tx.SetBlock(cube.Pos{4, 10, 0}, mcblock.Gold{}, nil)
		if _, err = service.RemoveAbove(tx, s, center, []string{"1"}); err != nil {
			return
		}
		if _, err = service.RemoveBelow(tx, s, center, []string{"1"}); err != nil {
			return
		}
		if _, err = service.RemoveNear(tx, s, center, []string{"gold_block", "2"}); err != nil {
			return
		}
		aboveAir = parse.IsAir(tx.Block(cube.Pos{0, 11, 0}))
		belowAir = parse.IsAir(tx.Block(cube.Pos{0, 9, 0}))
		nearAir = parse.IsAir(tx.Block(cube.Pos{1, 10, 0}))
		farKept = parse.SameBlock(tx.Block(cube.Pos{4, 10, 0}), mcblock.Gold{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !aboveAir || !belowAir || !nearAir || !farKept {
		t.Fatalf("remove results above=%v below=%v near=%v farKept=%v", aboveAir, belowAir, nearAir, farKept)
	}
}

func TestMoveShiftsSelectionAndClearsSource(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)

		result, err := service.Move(tx, s, cube.Pos{1, 0, 0}, []string{"all", "1"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want 2", result.Changed)
		}
		if !parse.IsAir(tx.Block(cube.Pos{0, 0, 0})) {
			t.Fatal("source block was not cleared")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{1, 0, 0}), mcblock.Stone{}) {
			t.Fatal("destination block was not moved")
		}
	})
}

func TestMoveSubChunkGuardrailCountsOnlySourceAndDestination(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		s.guardrails = guardrail.Limits{MaxEditSubChunks: 2}
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)

		result, err := service.Move(tx, s, cube.Pos{1, 0, 0}, []string{"all", "256"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want 2", result.Changed)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{256, 0, 0}), mcblock.Stone{}) {
			t.Fatal("destination block was not moved")
		}
	})
}

func TestStackCopiesSelectionByAreaSize(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 1, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Dirt{}, nil)

		result, err := service.Stack(tx, s, cube.Pos{1, 0, 0}, []string{"1"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 2 {
			t.Fatalf("changed = %d, want 2", result.Changed)
		}
		if !parse.SameBlock(tx.Block(cube.Pos{2, 0, 0}), mcblock.Stone{}) {
			t.Fatal("first stacked block mismatch")
		}
		if !parse.SameBlock(tx.Block(cube.Pos{3, 0, 0}), mcblock.Dirt{}) {
			t.Fatal("second stacked block mismatch")
		}
	})
}

func TestRotateTurnsClipboardWithoutChangingWorld(t *testing.T) {
	var changed int
	var err error
	var worldChanged bool
	var pasted bool
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 1))
		tx.SetBlock(cube.Pos{0, 0, 1}, mcblock.Stone{}, nil)
		_, err = service.Copy(tx, s, cube.Pos{0, 0, 0}, cube.North, nil)
		if err != nil {
			return
		}
		result, rotateErr := service.Rotate(tx, s, []string{"90", "y"})
		changed, err = result.Changed, rotateErr
		worldChanged = !parse.SameBlock(tx.Block(cube.Pos{0, 0, 1}), mcblock.Stone{})
		if err != nil {
			return
		}
		_, err = service.Paste(tx, s, cube.Pos{10, 0, 0}, cube.North, nil)
		pasted = parse.SameBlock(tx.Block(cube.Pos{-1, 0, 0}.Add(cube.Pos{10, 0, 0})), mcblock.Stone{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed != 2 {
		t.Fatalf("changed = %d, want 2 clipboard entries", changed)
	}
	if worldChanged {
		t.Fatal("rotate edited the world instead of only the clipboard")
	}
	if !pasted {
		t.Fatal("rotated clipboard did not paste at expected offset")
	}
}

func TestFlipMirrorsClipboardWithoutChangingWorld(t *testing.T) {
	var changed int
	var err error
	var worldChanged bool
	var pasted bool
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 1, 0, 0))
		tx.SetBlock(cube.Pos{1, 0, 0}, mcblock.Stone{}, nil)
		_, err = service.Copy(tx, s, cube.Pos{0, 0, 0}, cube.North, nil)
		if err != nil {
			return
		}
		result, flipErr := service.Flip(tx, s, "x")
		changed, err = result.Changed, flipErr
		worldChanged = !parse.SameBlock(tx.Block(cube.Pos{1, 0, 0}), mcblock.Stone{})
		if err != nil {
			return
		}
		_, err = service.Paste(tx, s, cube.Pos{10, 0, 0}, cube.North, nil)
		pasted = parse.SameBlock(tx.Block(cube.Pos{9, 0, 0}), mcblock.Stone{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed != 2 {
		t.Fatalf("changed = %d, want 2 clipboard entries", changed)
	}
	if worldChanged {
		t.Fatal("flip edited the world instead of only the clipboard")
	}
	if !pasted {
		t.Fatal("flipped clipboard did not paste at expected offset")
	}
}

func TestLineDrawsBetweenSelectionCorners(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 2, 0, 0))

		result, err := service.Line(tx, s, []string{"stone", "1"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed != 3 {
			t.Fatalf("changed = %d, want 3", result.Changed)
		}
		for x := 0; x <= 2; x++ {
			if !parse.SameBlock(tx.Block(cube.Pos{x, 0, 0}), mcblock.Stone{}) {
				t.Fatalf("line missed x=%d", x)
			}
		}
	})
}

func TestNaturalizeBuildsGrassDirtStoneLayers(t *testing.T) {
	var err error
	var topGrass, dirtLayer, stoneLayer bool
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 5, 0))
		for y := 0; y <= 4; y++ {
			tx.SetBlock(cube.Pos{0, y, 0}, mcblock.Gold{}, nil)
		}
		_, err = service.Naturalize(tx, s)
		topGrass = parse.SameBlock(tx.Block(cube.Pos{0, 4, 0}), mcblock.Grass{})
		dirtLayer = parse.SameBlock(tx.Block(cube.Pos{0, 3, 0}), mcblock.Dirt{})
		stoneLayer = parse.SameBlock(tx.Block(cube.Pos{0, 0, 0}), mcblock.Stone{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !topGrass || !dirtLayer || !stoneLayer {
		t.Fatalf("naturalize layers top=%v dirt=%v stone=%v", topGrass, dirtLayer, stoneLayer)
	}
}

func TestSearchItemsFindsRegisteredNames(t *testing.T) {
	matches := service.SearchItems("stone", 5)
	if len(matches) == 0 {
		t.Fatal("expected stone search matches")
	}
	for _, match := range matches {
		if !strings.Contains(match, "stone") {
			t.Fatalf("match %q does not contain query", match)
		}
	}
}

func TestParseShapeArgsForCubeAndErrors(t *testing.T) {
	spec, blocks, err := service.ParseShapeArgs(edit.ShapeCube, []string{"stone", "3", "2", "1"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != edit.ShapeCube || !spec.Hollow || spec.Length != 3 || spec.Width != 2 || spec.Height != 1 {
		t.Fatalf("spec = %+v, want hollow 3x2x1 cube", spec)
	}
	if len(blocks) != 1 || !parse.SameBlock(blocks[0], mcblock.Stone{}) {
		t.Fatalf("blocks = %v, want stone", blocks)
	}
	if _, _, err := service.ParseShapeArgs(edit.ShapeCube, []string{"stone", "3", "2"}, false); err == nil {
		t.Fatal("ParseShapeArgs accepted missing cube height")
	}
	if _, _, err := service.ParseShapeArgs(edit.ShapeSphere, []string{"stone", "x", "2"}, false); err == nil {
		t.Fatal("ParseShapeArgs accepted non-numeric sphere radius")
	}
}

func TestSetBiomeRecordsUndoableBiomeChange(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		pos := cube.Pos{0, 0, 0}
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		before := tx.Biome(pos)
		var target world.Biome
		for _, b := range world.Biomes() {
			if !parse.SameBiome(before, b) {
				target = b
				break
			}
		}
		if target == nil {
			t.Fatal("expected at least two registered biomes")
		}
		got, err := service.SetBiome(tx, s, target.String())
		if err != nil {
			t.Fatal(err)
		}
		if !parse.SameBiome(got, target) || !parse.SameBiome(tx.Biome(pos), target) {
			t.Fatal("biome was not set to target")
		}
		if err := service.Undo(tx, s, false); err != nil {
			t.Fatal(err)
		}
		if !parse.SameBiome(tx.Biome(pos), before) {
			t.Fatal("undo did not restore biome")
		}
	})
}

func TestSchematicListAndDelete(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		store := edit.NewFileSchematicStore(filepath.Join(t.TempDir(), "schematics"))
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		tx.SetBlock(cube.Pos{0, 0, 0}, mcblock.Stone{}, nil)

		for _, name := range []string{"two", "one"} {
			if _, err := service.Schematic(tx, s, cube.Pos{0, 0, 0}, cube.North, store, []string{"create", name}); err != nil {
				t.Fatalf("create %s: %v", name, err)
			}
		}
		listed, err := service.Schematic(tx, s, cube.Pos{}, cube.North, store, []string{"list"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(listed.Names, ",") != "one,two" {
			t.Fatalf("listed names = %v, want [one two]", listed.Names)
		}
		deleted, err := service.Schematic(tx, s, cube.Pos{}, cube.North, store, []string{"delete", "one"})
		if err != nil {
			t.Fatal(err)
		}
		if deleted.Name != "one" {
			t.Fatalf("deleted name = %q, want one", deleted.Name)
		}
		listed, err = service.Schematic(tx, s, cube.Pos{}, cube.North, store, []string{"list"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(listed.Names, ",") != "two" {
			t.Fatalf("listed names after delete = %v, want [two]", listed.Names)
		}
	})
}

func TestUndoRedoEmptyHistoryErrors(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		s := newFakeSession(geo.NewArea(0, 0, 0, 0, 0, 0))
		if err := service.Undo(tx, s, false); !errors.Is(err, service.ErrNothingToUndo) {
			t.Fatalf("Undo error = %v, want ErrNothingToUndo", err)
		}
		if err := service.Redo(tx, s, false); !errors.Is(err, service.ErrNothingToRedo) {
			t.Fatalf("Redo error = %v, want ErrNothingToRedo", err)
		}
	})
}
