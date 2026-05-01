package cmd

import (
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// SetCommand implements //set <blocks> — fills the selection with random picks.
type SetCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c SetCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.FillArea(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Set %d blocks.", batch.Len())
}

// CenterCommand implements //center <blocks> — places one block at the selection's centre.
type CenterCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c CenterCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	pos := edit.Center(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Marked center at %v.", pos)
}

// WallsCommand implements //walls <blocks> — fills only the outer shell of the selection.
type WallsCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c WallsCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.Walls(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Built walls with %d changes.", batch.Len())
}

// DrainCommand implements //drain <radius> — removes fluids in a sphere around the player.
type DrainCommand struct {
	playerCommand
	Radius int `cmd:"radius"`
}

func (c DrainCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	if c.Radius < 1 {
		o.Error("radius must be positive")
		return
	}
	batch := history.NewBatch(false)
	edit.Drain(tx, cube.PosFromVec3(p.Position()), c.Radius, batch)
	record(p, batch)
	o.Printf("Drained %d blocks.", batch.Len())
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
		bs := world.Biomes()
		names := make([]string, 0, len(bs))
		for _, b := range bs {
			names = append(names, b.String())
		}
		o.Print("Biomes: " + strings.Join(names, ", "))
		return
	}
	if !strings.EqualFold(args[0], "set") || len(args) < 2 {
		o.Error("usage: //biome list | //biome set <biome>")
		return
	}
	b, ok := world.BiomeByName(args[1])
	if !ok {
		o.Errorf("unknown biome %q", args[1])
		return
	}
	area, selected := selectedArea(p, o)
	if !selected {
		return
	}
	batch := history.NewBatch(false)
	area.Range(func(x, y, z int) { batch.SetBiome(tx, cube.Pos{x, y, z}, b) })
	record(p, batch)
	o.Printf("Set biome %s.", b.String())
}

// ReplaceCommand implements //replace <mask> <to> — swaps matching blocks in the selection.
type ReplaceCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c ReplaceCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //replace <all|from> <to>")
		return
	}
	mask, to, err := parseMaskTo(args)
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.ReplaceArea(tx, area, mask, to, batch)
	record(p, batch)
	o.Printf("Replaced %d blocks.", batch.Len())
}

// ReplaceNearCommand implements //replacenear <distance> <mask> <to> — replace inside a sphere around the player.
type ReplaceNearCommand struct {
	playerCommand
	Distance int         `cmd:"distance"`
	Args     dcf.Varargs `cmd:"args"`
}

func (c ReplaceNearCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if c.Distance < 1 || len(args) < 2 {
		o.Error("usage: //replacenear <distance> <from> <to>")
		return
	}
	mask, to, err := parseMaskTo(args)
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.ReplaceNear(tx, cube.PosFromVec3(p.Position()), c.Distance, mask, to, batch)
	record(p, batch)
	o.Printf("Replaced %d nearby blocks.", batch.Len())
}

// TopLayerCommand implements //toplayer <mask> <to> — replaces only the topmost matching block per column.
type TopLayerCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c TopLayerCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //toplayer <all|only:types> <to>")
		return
	}
	mask, to, err := parseMaskTo(args)
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.TopLayer(tx, area, mask, to, batch)
	record(p, batch)
	o.Printf("Replaced %d top-layer blocks.", batch.Len())
}

// OverlayCommand implements //overlay <blocks> — places blocks above the highest solid blocks per column.
type OverlayCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c OverlayCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.Overlay(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Overlay changed %d blocks.", batch.Len())
}
