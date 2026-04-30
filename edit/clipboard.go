package edit

import (
	"fmt"
	"math/rand"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
)

type Clipboard struct {
	OriginDir cube.Direction
	Entries   []bufferEntry
}

func CopySelection(tx *world.Tx, area geo.Area, origin cube.Pos, dir cube.Direction, mask BlockMask, only bool) *Clipboard {
	return &Clipboard{OriginDir: dir, Entries: copyArea(tx, area, origin, mask, !only)}
}

func PasteClipboard(tx *world.Tx, cb *Clipboard, origin cube.Pos, dir cube.Direction, noAir bool, batch *history.Batch) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("clipboard is empty")
	}
	entries := make([]bufferEntry, len(cb.Entries))
	copy(entries, cb.Entries)
	turns := rotationTurns(cb.OriginDir, dir)
	for i := range entries {
		entries[i].Offset = rotateY(entries[i].Offset, turns)
	}
	pasteBuffer(tx, origin, entries, noAir, batch)
	return nil
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

func rotateY(pos cube.Pos, turns int) cube.Pos {
	for i := 0; i < turns; i++ {
		pos = cube.Pos{-pos[2], pos[1], pos[0]}
	}
	return pos
}

func randomBlock(blocks []world.Block) world.Block {
	return ChooseBlock(blocks, rand.New(rand.NewSource(1)))
}
