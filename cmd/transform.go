package cmd

import (
	"strconv"
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
)

// MoveCommand implements //move <mask> <distance> [-a] — shifts matching blocks along the player's facing.
type MoveCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c MoveCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //move <all|only:types> <distance> [-a]")
		return
	}
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		o.Error(err)
		return
	}
	dist, err := strconv.Atoi(args[1])
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.Move(tx, area, edit.DirectionVector(p.Rotation().Direction().Face()), dist, mask, hasFlag(args[2:], "-a"), batch)
	record(p, batch)
	o.Printf("Moved %d blocks.", batch.Len())
}

// StackCommand implements //stack <amount> [-a] — repeats the selection along the player's facing.
type StackCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c StackCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 1 {
		o.Error("usage: //stack <amount> [-a]")
		return
	}
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.Stack(tx, area, edit.DirectionVector(p.Rotation().Direction().Face()), amount, hasFlag(args[1:], "-a"), batch)
	record(p, batch)
	o.Printf("Stacked with %d changes.", batch.Len())
}

// RotateCommand implements //rotate <90|180|270|360> [axis] — rotates blocks inside the selection in place.
type RotateCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RotateCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 1 {
		o.Error("usage: //rotate <90|180|270|360> [x|y|z]")
		return
	}
	deg, err := strconv.Atoi(args[0])
	if err != nil || (deg != 90 && deg != 180 && deg != 270 && deg != 360) {
		o.Error("rotation must be one of 90, 180, 270, or 360")
		return
	}
	axis := "y"
	if len(args) > 1 {
		axis = args[1]
	}
	if !validAxis(axis) {
		o.Error("axis must be x, y, or z")
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.RotateCopy(tx, area, deg, axis, batch)
	record(p, batch)
	o.Printf("Rotated copy with %d changes.", batch.Len())
}

// FlipCommand implements //flip [axis] — mirrors the selection across an axis (defaults from facing).
type FlipCommand struct {
	playerCommand
	Axis dcf.Optional[string] `cmd:"axis"`
}

func (c FlipCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	var axis string
	if v, ok := c.Axis.Load(); ok {
		axis = v
	} else {
		switch p.Rotation().Direction() {
		case cube.North, cube.South:
			axis = "z"
		default:
			axis = "x"
		}
	}
	if !validAxis(axis) {
		o.Error("axis must be x, y, or z")
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.FlipCopy(tx, area, axis, batch)
	record(p, batch)
	o.Printf("Flipped copy with %d changes.", batch.Len())
}
