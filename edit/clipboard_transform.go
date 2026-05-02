package edit

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// RotateClipboard rotates clipboard offsets around the clipboard origin.
func RotateClipboard(cb *Clipboard, degrees int, axis string) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("clipboard is empty")
	}
	turns := ((degrees/90)%4 + 4) % 4
	if turns == 0 {
		return nil
	}
	axis = strings.ToLower(axis)
	for i := range cb.Entries {
		cb.Entries[i].Offset = rotateOffset(cb.Entries[i].Offset, axis, turns)
	}
	return nil
}

// FlipClipboard mirrors clipboard offsets around the clipboard origin.
func FlipClipboard(cb *Clipboard, axis string) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("clipboard is empty")
	}
	for i := range cb.Entries {
		o := cb.Entries[i].Offset
		switch strings.ToLower(axis) {
		case "y":
			o[1] = -o[1]
		case "z":
			o[2] = -o[2]
		default:
			o[0] = -o[0]
		}
		cb.Entries[i].Offset = o
	}
	return nil
}

func rotateOffset(pos cube.Pos, axis string, turns int) cube.Pos {
	for i := 0; i < turns; i++ {
		switch axis {
		case "x":
			pos = cube.Pos{pos[0], -pos[2], pos[1]}
		case "z":
			pos = cube.Pos{-pos[1], pos[0], pos[2]}
		default:
			pos = cube.Pos{-pos[2], pos[1], pos[0]}
		}
	}
	return pos
}
