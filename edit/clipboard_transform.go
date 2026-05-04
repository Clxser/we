package edit

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
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
	transform := blockTransform{axis: axis, turns: turns}
	cache := make(blockTransformCache)
	for i := range cb.Entries {
		cb.Entries[i].Offset = rotateOffset(cb.Entries[i].Offset, axis, turns)
		cb.Entries[i].Block = cache.transform(cb.Entries[i].Block, transform)
		if cb.Entries[i].HasLiq {
			if l, ok := cache.transform(cb.Entries[i].Liquid, transform).(world.Liquid); ok {
				cb.Entries[i].Liquid = l
			}
		}
	}
	return nil
}

// FlipClipboard mirrors clipboard offsets around the clipboard origin.
func FlipClipboard(cb *Clipboard, axis string) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("clipboard is empty")
	}
	axis = strings.ToLower(axis)
	transform := blockTransform{axis: axis, flip: true}
	cache := make(blockTransformCache)
	for i := range cb.Entries {
		o := cb.Entries[i].Offset
		switch axis {
		case "y":
			o[1] = -o[1]
		case "z":
			o[2] = -o[2]
		default:
			o[0] = -o[0]
		}
		cb.Entries[i].Offset = o
		cb.Entries[i].Block = cache.transform(cb.Entries[i].Block, transform)
		if cb.Entries[i].HasLiq {
			if l, ok := cache.transform(cb.Entries[i].Liquid, transform).(world.Liquid); ok {
				cb.Entries[i].Liquid = l
			}
		}
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
