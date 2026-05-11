package service

import "fmt"

const maxUndoClipboardEntries = 1_000_000

func ensureClipboardUndoBudget(entries int, opts EditOptions) error {
	if opts.NoUndo || entries <= maxUndoClipboardEntries {
		return nil
	}
	return fmt.Errorf(
		"clipboard has %d blocks; undo history for large pastes is disabled above %d blocks, rerun with -noundo",
		entries, maxUndoClipboardEntries,
	)
}
