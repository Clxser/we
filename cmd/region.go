package cmd

import (
	"strconv"
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/service"
	"github.com/df-mc/we/session"
)

// SetCommand implements //set <blocks> — fills the selection with random picks.
type SetCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c SetCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	blockSpec, opts := parseSetArgs(string(c.Blocks))
	timer := startOperation()
	result, err := service.SetWithOptions(tx, session.Ensure(p), blockSpec, opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Set %d blocks.", result.Changed)
}

func parseSetArgs(raw string) (string, service.EditOptions) {
	args := strings.Fields(raw)
	args, opts := service.ParseEditOptions(args)
	return strings.Join(args, " "), opts
}

// CenterCommand implements //center <blocks> — places one block at the selection's centre.
type CenterCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c CenterCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.Center(tx, session.Ensure(p), string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Marked center at %v.", result.Pos)
}

// WallsCommand implements //walls <blocks> — fills only the outer shell of the selection.
type WallsCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c WallsCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	blockSpec, opts := parseSetArgs(string(c.Blocks))
	timer := startOperation()
	result, err := service.WallsWithOptions(tx, session.Ensure(p), blockSpec, opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Built walls with %d changes.", result.Changed)
}

// DrainCommand implements //drain <radius> — removes fluids in a sphere around the player.
type DrainCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c DrainCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args, opts := service.ParseEditOptions(strings.Fields(string(c.Args)))
	if len(args) != 1 {
		o.Error("usage: //drain <radius> [-noundo]")
		return
	}
	radius, err := strconv.Atoi(args[0])
	if err != nil {
		o.Error("radius must be positive")
		return
	}
	timer := startOperation()
	result, err := service.DrainWithOptions(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), radius, opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Drained %d blocks.", result.Changed)
}

// BiomeCommand implements //biome list and //biome set <name> — biome inspection and assignment.
type BiomeCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c BiomeCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) == 0 || strings.EqualFold(args[0], "list") {
		o.Print("Biomes: " + strings.Join(service.BiomeNames(), ", "))
		return
	}
	if !strings.EqualFold(args[0], "set") || len(args) < 2 {
		o.Error("usage: //biome list | //biome set <biome>")
		return
	}
	setArgs, opts := service.ParseEditOptions(args[1:])
	if len(setArgs) != 1 {
		o.Error("usage: //biome set <biome> [-noundo]")
		return
	}
	timer := startOperation()
	b, err := service.SetBiomeWithOptions(tx, session.Ensure(p), setArgs[0], opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Set biome %s.", b.String())
}

// ReplaceCommand implements //replace <mask> <to> — swaps matching blocks in the selection.
type ReplaceCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c ReplaceCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.Replace(tx, session.Ensure(p), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Replaced %d blocks.", result.Changed)
}

// ReplaceNearCommand implements //replacenear <distance> <mask> <to> — replace inside a sphere around the player.
type ReplaceNearCommand struct {
	playerCommand
	Distance int         `cmd:"distance"`
	Args     dcf.Varargs `cmd:"args"`
}

func (c ReplaceNearCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.ReplaceNear(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), c.Distance, strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Replaced %d nearby blocks.", result.Changed)
}

// TopLayerCommand implements //toplayer <mask> <to> — replaces only the topmost matching block per column.
type TopLayerCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c TopLayerCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.TopLayer(tx, session.Ensure(p), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Replaced %d top-layer blocks.", result.Changed)
}

// OverlayCommand implements //overlay <blocks> — places blocks above the highest solid blocks per column.
type OverlayCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c OverlayCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	blockSpec, opts := parseSetArgs(string(c.Blocks))
	timer := startOperation()
	result, err := service.OverlayWithOptions(tx, session.Ensure(p), blockSpec, opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Overlay changed %d blocks.", result.Changed)
}

// RemoveAboveCommand implements //removeabove [height] [radius] — clears blocks above the player.
type RemoveAboveCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RemoveAboveCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.RemoveAbove(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Removed %d blocks above.", result.Changed)
}

// RemoveBelowCommand implements //removebelow [height] [radius] — clears blocks below the player.
type RemoveBelowCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RemoveBelowCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.RemoveBelow(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Removed %d blocks below.", result.Changed)
}

// RemoveNearCommand implements //removenear <blocks> <radius> — clears matching nearby blocks.
type RemoveNearCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RemoveNearCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.RemoveNear(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Removed %d nearby blocks.", result.Changed)
}

// NaturalizeCommand implements //naturalize — turns selected terrain into grass, dirt, and stone layers.
type NaturalizeCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c NaturalizeCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args, opts := service.ParseEditOptions(strings.Fields(string(c.Args)))
	if len(args) != 0 {
		o.Error("usage: //naturalize [-noundo]")
		return
	}
	timer := startOperation()
	result, err := service.NaturalizeWithOptions(tx, session.Ensure(p), opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Naturalized %d blocks.", result.Changed)
}
