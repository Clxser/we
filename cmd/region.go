package cmd

import (
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
	result, err := service.Set(tx, session.Ensure(p), string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Set %d blocks.", result.Changed)
}

// CenterCommand implements //center <blocks> — places one block at the selection's centre.
type CenterCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c CenterCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Center(tx, session.Ensure(p), string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Marked center at %v.", result.Pos)
}

// WallsCommand implements //walls <blocks> — fills only the outer shell of the selection.
type WallsCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c WallsCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Walls(tx, session.Ensure(p), string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Built walls with %d changes.", result.Changed)
}

// DrainCommand implements //drain <radius> — removes fluids in a sphere around the player.
type DrainCommand struct {
	playerCommand
	Radius int `cmd:"radius"`
}

func (c DrainCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Drain(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), c.Radius)
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Drained %d blocks.", result.Changed)
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
	b, err := service.SetBiome(tx, session.Ensure(p), args[1])
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Set biome %s.", b.String())
}

// ReplaceCommand implements //replace <mask> <to> — swaps matching blocks in the selection.
type ReplaceCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c ReplaceCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Replace(tx, session.Ensure(p), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Replaced %d blocks.", result.Changed)
}

// ReplaceNearCommand implements //replacenear <distance> <mask> <to> — replace inside a sphere around the player.
type ReplaceNearCommand struct {
	playerCommand
	Distance int         `cmd:"distance"`
	Args     dcf.Varargs `cmd:"args"`
}

func (c ReplaceNearCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.ReplaceNear(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), c.Distance, strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Replaced %d nearby blocks.", result.Changed)
}

// TopLayerCommand implements //toplayer <mask> <to> — replaces only the topmost matching block per column.
type TopLayerCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c TopLayerCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.TopLayer(tx, session.Ensure(p), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Replaced %d top-layer blocks.", result.Changed)
}

// OverlayCommand implements //overlay <blocks> — places blocks above the highest solid blocks per column.
type OverlayCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c OverlayCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Overlay(tx, session.Ensure(p), string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Overlay changed %d blocks.", result.Changed)
}
