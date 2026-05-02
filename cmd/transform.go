package cmd

import (
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/service"
	"github.com/df-mc/we/session"
)

// MoveCommand implements //move <mask> <distance> [-a] — shifts matching blocks along the player's facing.
type MoveCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c MoveCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Move(tx, session.Ensure(p), edit.DirectionVector(p.Rotation().Direction().Face()), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Moved %d blocks.", result.Changed)
}

// StackCommand implements //stack <amount> [-a] — repeats the selection along the player's facing.
type StackCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c StackCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Stack(tx, session.Ensure(p), edit.DirectionVector(p.Rotation().Direction().Face()), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Stacked with %d changes.", result.Changed)
}

// RotateCommand implements //rotate <90|180|270|360> [axis] — rotates the clipboard.
type RotateCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RotateCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Rotate(tx, session.Ensure(p), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Rotated clipboard with %d entries.", result.Changed)
}

// FlipCommand implements //flip [axis] — mirrors the clipboard across an axis (defaults from facing).
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
	result, err := service.Flip(tx, session.Ensure(p), axis)
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Flipped clipboard with %d entries.", result.Changed)
}
