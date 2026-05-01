package service

import (
	"fmt"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
)

func Move(tx *world.Tx, s Session, dir cube.Pos, args []string) (ChangeResult, error) {
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
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.Move(tx, area, dir, dist, mask, HasFlag(args[2:], "-a"), batch)
	return record(s, batch), nil
}

func Stack(tx *world.Tx, s Session, dir cube.Pos, args []string) (ChangeResult, error) {
	if len(args) < 1 {
		return ChangeResult{}, fmt.Errorf("usage: //stack <amount> [-a]")
	}
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.Stack(tx, area, dir, amount, HasFlag(args[1:], "-a"), batch)
	return record(s, batch), nil
}

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
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.RotateCopy(tx, area, deg, axis, batch)
	return record(s, batch), nil
}

func Flip(tx *world.Tx, s Session, axis string) (ChangeResult, error) {
	if !ValidAxis(axis) {
		return ChangeResult{}, fmt.Errorf("axis must be x, y, or z")
	}
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.FlipCopy(tx, area, axis, batch)
	return record(s, batch), nil
}
