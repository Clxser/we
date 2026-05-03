package service

import (
	"fmt"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
)

// Move shifts blocks matching args[0] by args[1] along dir. The "-a" flag skips
// writing air at the destination.
func Move(tx *world.Tx, s Session, dir cube.Pos, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return MoveWithOptions(tx, s, dir, args, opts)
}

func MoveWithOptions(tx *world.Tx, s Session, dir cube.Pos, args []string, opts EditOptions) (ChangeResult, error) {
	if len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //move <all|only:types> <distance> [-a]")
	}
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		return ChangeResult{}, err
	}
	dist, err := strconv.Atoi(args[1])
	if err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedReadArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	dest := area.Add(cube.Pos{dir[0] * dist, dir[1] * dist, dir[2] * dist})
	if err := guardrailsFor(s).CheckEditSubChunks(geo.UniqueSubChunks(area, dest)); err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.Move(tx, area, dir, dist, mask, HasFlag(args[2:], "-a"), batch)
	return finishEdit(s, batch, int(area.Volume())*2), nil
}

// Stack repeats the selection args[0] times along dir.
func Stack(tx *world.Tx, s Session, dir cube.Pos, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return StackWithOptions(tx, s, dir, args, opts)
}

func StackWithOptions(tx *world.Tx, s Session, dir cube.Pos, args []string, opts EditOptions) (ChangeResult, error) {
	if len(args) < 1 {
		return ChangeResult{}, fmt.Errorf("usage: //stack <amount> [-a]")
	}
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		return ChangeResult{}, err
	}
	if err := guardrailsFor(s).CheckStackCopies(amount); err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedReadArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	if err := guardrailsFor(s).CheckEditSubChunks(stackEditBounds(area, dir, amount).SubChunkCount()); err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.Stack(tx, area, dir, amount, HasFlag(args[1:], "-a"), batch)
	return finishEdit(s, batch, int(area.Volume())*amount), nil
}

func stackEditBounds(area geo.Area, dir cube.Pos, amount int) geo.Area {
	if amount <= 0 {
		return area
	}
	step := cube.Pos{dir[0] * area.Dx(), dir[1] * area.Dy(), dir[2] * area.Dz()}
	last := cube.Pos{step[0] * amount, step[1] * amount, step[2] * amount}
	return area.Add(step).Union(area.Add(last))
}

// Rotate rotates the clipboard by args[0] degrees (90, 180, 270, or 360)
// around the optional args[1] axis (default y).
func Rotate(tx *world.Tx, s Session, args []string) (ChangeResult, error) {
	if len(args) < 1 {
		return ChangeResult{}, fmt.Errorf("usage: //rotate <90|180|270|360> [x|y|z]")
	}
	deg, err := strconv.Atoi(args[0])
	if err != nil || (deg != 90 && deg != 180 && deg != 270 && deg != 360) {
		return ChangeResult{}, fmt.Errorf("rotation must be one of 90, 180, 270, or 360")
	}
	axis := "y"
	if len(args) > 1 {
		axis = args[1]
	}
	if !ValidAxis(axis) {
		return ChangeResult{}, fmt.Errorf("axis must be x, y, or z")
	}
	cb, ok := s.Clipboard()
	if !ok {
		return ChangeResult{}, ErrClipboardEmpty
	}
	if err := edit.RotateClipboard(cb, deg, axis); err != nil {
		return ChangeResult{}, err
	}
	s.SetClipboard(cb)
	return ChangeResult{Changed: len(cb.Entries)}, nil
}

// Flip mirrors the clipboard across axis (x, y, or z).
func Flip(tx *world.Tx, s Session, axis string) (ChangeResult, error) {
	if !ValidAxis(axis) {
		return ChangeResult{}, fmt.Errorf("axis must be x, y, or z")
	}
	cb, ok := s.Clipboard()
	if !ok {
		return ChangeResult{}, ErrClipboardEmpty
	}
	if err := edit.FlipClipboard(cb, axis); err != nil {
		return ChangeResult{}, err
	}
	s.SetClipboard(cb)
	return ChangeResult{Changed: len(cb.Entries)}, nil
}
