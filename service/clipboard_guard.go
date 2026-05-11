package service

import (
	"fmt"

	"github.com/df-mc/we/history"
)

const maxUndoClipboardEntries = 1_000_000
const maxUndoEditPositions = 1_000_000

func ensureClipboardUndoBudget(entries int, opts EditOptions) error {
	return ensureUndoBudget(int64(entries), maxUndoClipboardEntries, "clipboard", opts)
}

func historyBatchForSize(opts EditOptions, positions int64) (*history.Batch, error) {
	if err := ensureUndoBudget(positions, maxUndoEditPositions, "edit", opts); err != nil {
		return nil, err
	}
	return historyBatch(opts), nil
}

func ensureUndoBudget(count, limit int64, label string, opts EditOptions) error {
	if opts.NoUndo || count <= limit {
		return nil
	}
	return fmt.Errorf(
		"%s has %d blocks; undo history for large edits is disabled above %d blocks, rerun with -noundo",
		label, count, limit,
	)
}
