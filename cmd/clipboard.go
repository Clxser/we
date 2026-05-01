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
	"github.com/df-mc/we/session"
)

// CopyCommand implements //copy [only <blocks>] — copies the selection to the player's clipboard.
type CopyCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c CopyCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	args := strings.Fields(string(c.Args))
	only := len(args) > 0 && strings.EqualFold(args[0], "only")
	mask := edit.BlockMask{All: true, IncludeAir: true}
	if only {
		if len(args) < 2 {
			o.Error("copy only requires block types")
			return
		}
		blocks, err := parse.ParseBlockList(strings.Join(args[1:], " "))
		if err != nil {
			o.Error(err)
			return
		}
		mask = edit.BlockMask{Blocks: blocks}
	}
	cb := edit.CopySelection(tx, area, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), mask, only)
	session.Ensure(p).SetClipboard(cb)
	o.Printf("Copied %d blocks.", len(cb.Entries))
}

// PasteCommand implements //paste [-a] — pastes the clipboard at the player's position.
type PasteCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c PasteCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	cb, ok := session.Ensure(p).Clipboard()
	if !ok {
		o.Error("clipboard is empty")
		return
	}
	batch := history.NewBatch(false)
	if err := edit.PasteClipboard(tx, cb, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), hasFlag(strings.Fields(string(c.Args)), "-a"), batch); err != nil {
		o.Error(err)
		return
	}
	record(p, batch)
	o.Printf("Pasted %d blocks.", batch.Len())
}

// CutCommand implements //cut — copies the selection to the clipboard, then clears it.
type CutCommand struct{ playerCommand }

func (CutCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	cb := edit.CopySelection(tx, area, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), edit.BlockMask{All: true, IncludeAir: true}, false)
	session.Ensure(p).SetClipboard(cb)
	batch := history.NewBatch(false)
	edit.ClearArea(tx, area, batch)
	record(p, batch)
	o.Printf("Cut %d blocks.", batch.Len())
}

// SchematicCommand implements //schematic <create|paste|delete|list> — disk-backed selection storage.
type SchematicCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c SchematicCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) == 0 {
		o.Error("usage: //schematic <create|paste|delete|list> [name] [-a]")
		return
	}
	switch strings.ToLower(args[0]) {
	case "create":
		if len(args) < 2 {
			o.Error("schematic create requires a name")
			return
		}
		area, ok := selectedArea(p, o)
		if !ok {
			return
		}
		cb := edit.CopySelection(tx, area, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), edit.BlockMask{All: true, IncludeAir: true}, false)
		if err := edit.SaveSchematic(args[1], cb); err != nil {
			o.Error(err)
			return
		}
		o.Printf("Saved schematic %q.", args[1])
	case "paste":
		if len(args) < 2 {
			o.Error("schematic paste requires a name")
			return
		}
		cb, err := edit.LoadSchematic(args[1])
		if err != nil {
			o.Error(err)
			return
		}
		batch := history.NewBatch(false)
		if err := edit.PasteClipboard(tx, cb, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), hasFlag(args[2:], "-a"), batch); err != nil {
			o.Error(err)
			return
		}
		record(p, batch)
		o.Printf("Pasted schematic %q.", args[1])
	case "delete":
		if len(args) < 2 {
			o.Error("schematic delete requires a name")
			return
		}
		if err := edit.DeleteSchematic(args[1]); err != nil {
			o.Error(err)
			return
		}
		o.Printf("Deleted schematic %q.", args[1])
	case "list":
		names, err := edit.ListSchematics()
		if err != nil {
			o.Error(err)
			return
		}
		o.Print("Schematics: " + strings.Join(names, ", "))
	default:
		o.Error("unknown schematic subcommand")
	}
}

// UndoCommand implements //undo [b] — reverts the last edit; "b" targets the brush stack.
type UndoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c UndoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	brush := optionalB(c.Target)
	if !session.Ensure(p).Undo(tx, brush) {
		o.Error("nothing to undo")
		return
	}
	o.Print("Undo successful.")
}

// RedoCommand implements //redo [b] — restores the last undone edit; "b" targets the brush stack.
type RedoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c RedoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	brush := optionalB(c.Target)
	if !session.Ensure(p).Redo(tx, brush) {
		o.Error("nothing to redo")
		return
	}
	o.Print("Redo successful.")
}
