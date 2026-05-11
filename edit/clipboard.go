package edit

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// Clipboard is a captured selection used by copy, cut, paste, and schematic IO.
//
// OriginDir is the player's facing at copy time so paste can rotate to match
// the new facing. Entries store offsets relative to that origin.
type Clipboard struct {
	OriginDir cube.Direction
	Entries   []bufferEntry
}

// CopySelection captures area into a Clipboard. Offsets are stored relative to origin.
// If only is true, mask filters which blocks are kept; otherwise every block (including air) is copied.
func CopySelection(tx *world.Tx, area geo.Area, origin cube.Pos, dir cube.Direction, mask BlockMask, only bool) *Clipboard {
	return &Clipboard{OriginDir: dir, Entries: copyArea(tx, area, origin, mask, !only)}
}

// PasteClipboard writes cb at origin, rotating around the Y axis to match dir.
// Returns an error if the clipboard is empty. When noAir is true, air entries are skipped.
func PasteClipboard(tx *world.Tx, cb *Clipboard, origin cube.Pos, dir cube.Direction, noAir bool, batch *history.Batch) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("clipboard is empty")
	}
	traceAnnotate("PasteClipboard begin",
		"cells", len(cb.Entries),
		"origin_dir", cb.OriginDir.String(),
		"paste_dir", dir.String(),
		"no_air", noAir,
		"with_history", batch != nil,
	)
	turns := rotationTurns(cb.OriginDir, dir)
	entries := cb.Entries
	if turns != 0 {
		tRot := startTrace("PasteClipboard.rotation_copy")
		entries = make([]bufferEntry, len(cb.Entries))
		copy(entries, cb.Entries)
		transform := blockTransform{axis: "y", turns: turns}
		cache := make(blockTransformCache)
		for i := range entries {
			entries[i].Offset = rotateOffset(entries[i].Offset, "y", turns)
			entries[i].Block = cache.transform(entries[i].Block, transform)
			if entries[i].HasLiq {
				if b, ok := cache.transform(entries[i].Liquid, transform).(world.Liquid); ok {
					entries[i].Liquid = b
				}
			}
		}
		tRot.end()
	} else {
		traceAnnotate("PasteClipboard.rotation_skipped (turns=0)")
	}
	tPaste := startTrace("PasteClipboard.pasteBuffer")
	pasteBuffer(tx, origin, entries, noAir, batch)
	tPaste.end()
	return nil
}

// PasteSubChunkCount returns how many unique 16x16x16 sub-chunks a paste would
// touch. It mirrors PasteClipboard's rotation and -a air skipping so service
// guardrails can reject pathological pastes before Dragonfly queues too many
// client-cache blobs for one player.
func PasteSubChunkCount(cb *Clipboard, origin cube.Pos, dir cube.Direction, noAir bool) int64 {
	if cb == nil || len(cb.Entries) == 0 {
		return 0
	}
	turns := rotationTurns(cb.OriginDir, dir)
	seen := make(map[[3]int32]struct{})
	for _, entry := range cb.Entries {
		if noAir && parse.IsAir(entry.Block) && !entry.HasLiq {
			continue
		}
		offset := entry.Offset
		if turns != 0 {
			offset = rotateOffset(offset, "y", turns)
		}
		pos := origin.Add(offset)
		seen[[3]int32{int32(pos[0] >> 4), int32(pos[1] >> 4), int32(pos[2] >> 4)}] = struct{}{}
	}
	return int64(len(seen))
}

func rotationTurns(from, to cube.Direction) int {
	dirs := []cube.Direction{cube.North, cube.East, cube.South, cube.West}
	fi, ti := 0, 0
	for i, d := range dirs {
		if d == from {
			fi = i
		}
		if d == to {
			ti = i
		}
	}
	return (ti - fi + 4) % 4
}
