package cmd

import (
	"strconv"
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/editbrush"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
	"github.com/df-mc/we/session"
)

// LineCommand implements //line <blocks> <thickness> — draws a line between pos1 and pos2.
type LineCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c LineCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //line <blocks> <thickness>")
		return
	}
	blocks, err := parse.ParseBlockList(args[0])
	if err != nil {
		o.Error(err)
		return
	}
	thickness, err := strconv.Atoi(args[1])
	if err != nil {
		o.Error(err)
		return
	}
	pos1, pos2, ok := session.Ensure(p).PosCorners()
	if !ok {
		o.Error("pos1 and pos2 must be set first")
		return
	}
	batch := history.NewBatch(false)
	edit.Line(tx, pos1, pos2, thickness, blocks, batch)
	record(p, batch)
	o.Printf("Drew line with %d changes.", batch.Len())
}

// ShapeCommand backs //sphere, //cylinder, //pyramid, //cone, and //cube.
// Kind selects the primitive; Args holds dimensions and the optional -h hollow flag.
type ShapeCommand struct {
	playerCommand
	Kind edit.ShapeKind `cmd:"-"`
	Args dcf.Varargs    `cmd:"args"`
}

func (c ShapeCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	hollow := hasFlag(args, "-h")
	args = removeFlags(args, "-h")
	spec, blocks, err := parseShapeArgs(c.Kind, args, hollow)
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.ApplyShape(tx, cube.PosFromVec3(p.Position()), spec, blocks, batch)
	record(p, batch)
	o.Printf("Created %s with %d changes.", c.Kind, batch.Len())
}

// BrushCommand implements //brush — opens the brush form with no args, or quick-binds with <type> [blocks] [radius].
type BrushCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c BrushCommand) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	held, off := p.HeldItems()
	if held.Empty() {
		o.Error("hold an item before running //brush")
		return
	}
	args := strings.Fields(string(c.Args))
	if len(args) == 0 {
		editbrush.SendBrushForm(p)
		o.Print("Opened brush menu.")
		return
	}
	cfg := editbrush.DefaultBrushConfig()
	cfg.Type = strings.ToLower(args[0])
	cfg.Shape = cfg.Type
	if len(args) > 1 {
		blocks, err := parse.ParseBlockList(args[1])
		if err != nil {
			o.Error(err)
			return
		}
		cfg.Blocks = editbrush.StatesFromBlocks(blocks)
	}
	if len(args) > 2 {
		if r, err := strconv.Atoi(args[2]); err == nil {
			cfg.Radius = r
			cfg.Height = r*2 + 1
			cfg.Length = r*2 + 1
			cfg.Width = r*2 + 1
		}
	}
	bound, err := editbrush.BindBrush(held, cfg)
	if err != nil {
		o.Error(err)
		return
	}
	p.SetHeldItems(bound, off)
	o.Printf("Bound %s brush.", cfg.Type)
}
