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
	timer := startOperation()
	result, err := service.Move(tx, session.Ensure(p), edit.DirectionVector(p.Rotation().Direction().Face()), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Moved %d blocks.", result.Changed)
}

// StackCommand implements //stack <amount> [-a] — repeats the selection along the player's facing.
type StackCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c StackCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	dir, args, ok := stackDirection(edit.DirectionVector(p.Rotation().Direction().Face()), strings.Fields(string(c.Args)))
	if !ok {
		o.Error("usage: //stack <amount> [up|down|north|south|east|west] [-a]")
		return
	}
	timer := startOperation()
	result, err := service.Stack(tx, session.Ensure(p), dir, args)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Stacked with %d changes.", result.Changed)
}

func stackDirection(defaultDir cube.Pos, args []string) (cube.Pos, []string, bool) {
	if len(args) <= 1 {
		return defaultDir, args, true
	}
	dir := defaultDir
	out := []string{args[0]}
	directionSet := false
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			out = append(out, arg)
			continue
		}
		if directionSet {
			return cube.Pos{}, nil, false
		}
		parsed, ok := namedDirection(arg)
		if !ok {
			return cube.Pos{}, nil, false
		}
		dir = parsed
		directionSet = true
	}
	return dir, out, true
}

func namedDirection(s string) (cube.Pos, bool) {
	switch strings.ToLower(s) {
	case "up", "u":
		return edit.DirectionVector(cube.FaceUp), true
	case "down", "d":
		return edit.DirectionVector(cube.FaceDown), true
	case "north", "n":
		return edit.DirectionVector(cube.FaceNorth), true
	case "south", "s":
		return edit.DirectionVector(cube.FaceSouth), true
	case "east", "e":
		return edit.DirectionVector(cube.FaceEast), true
	case "west", "w":
		return edit.DirectionVector(cube.FaceWest), true
	default:
		return cube.Pos{}, false
	}
}

// RotateCommand implements //rotate <90|180|270|360> [axis] — rotates the clipboard.
type RotateCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RotateCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.Rotate(tx, session.Ensure(p), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Rotated clipboard with %d entries.", result.Changed)
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
	timer := startOperation()
	result, err := service.Flip(tx, session.Ensure(p), axis)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Flipped clipboard with %d entries.", result.Changed)
}
