package service

import (
	"strings"
	"testing"
)

func TestEnsureClipboardUndoBudgetRejectsHugeUndo(t *testing.T) {
	err := ensureClipboardUndoBudget(maxUndoClipboardEntries+1, EditOptions{})
	if err == nil {
		t.Fatal("expected huge undoable paste to be rejected")
	}
	if !strings.Contains(err.Error(), "-noundo") {
		t.Fatalf("error %q should tell operators to use -noundo", err)
	}
}

func TestEnsureClipboardUndoBudgetAllowsNoUndo(t *testing.T) {
	if err := ensureClipboardUndoBudget(maxUndoClipboardEntries+1, EditOptions{NoUndo: true}); err != nil {
		t.Fatalf("NoUndo paste rejected: %v", err)
	}
}

func TestEnsureClipboardUndoBudgetAllowsNormalUndo(t *testing.T) {
	if err := ensureClipboardUndoBudget(maxUndoClipboardEntries, EditOptions{}); err != nil {
		t.Fatalf("normal undoable paste rejected: %v", err)
	}
}

func TestHistoryBatchForSizeRejectsHugeUndoableEdit(t *testing.T) {
	batch, err := historyBatchForSize(EditOptions{}, maxUndoEditPositions+1)
	if err == nil {
		t.Fatal("expected huge undoable edit to be rejected")
	}
	if batch != nil {
		t.Fatal("rejected edit returned a history batch")
	}
	if !strings.Contains(err.Error(), "-noundo") {
		t.Fatalf("error %q should tell operators to use -noundo", err)
	}
}

func TestHistoryBatchForSizeAllowsHugeNoUndoEdit(t *testing.T) {
	batch, err := historyBatchForSize(EditOptions{NoUndo: true}, maxUndoEditPositions+1)
	if err != nil {
		t.Fatalf("NoUndo edit rejected: %v", err)
	}
	if batch != nil {
		t.Fatal("NoUndo edit returned a history batch")
	}
}
