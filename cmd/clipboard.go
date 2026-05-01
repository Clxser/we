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

// CopyCommand implements //copy [only <blocks>] — copies the selection to the player's clipboard.
type CopyCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c CopyCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Copy(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction(), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Copied %d blocks.", result.Copied)
}

// PasteCommand implements //paste [-a] — pastes the clipboard at the player's position.
type PasteCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c PasteCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Paste(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction(), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Pasted %d blocks.", result.Changed)
}

// CutCommand implements //cut — copies the selection to the clipboard, then clears it.
type CutCommand struct{ playerCommand }

func (CutCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	result, err := service.Cut(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction())
	if err != nil {
		o.Error(err)
		return
	}
	o.Printf("Cut %d blocks.", result.Changed)
}

// SchematicCommand implements //schematic <create|paste|delete|list> — disk-backed selection storage.
type SchematicCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c SchematicCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	result, err := service.Schematic(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction(), args)
	if err != nil {
		o.Error(err)
		return
	}
	switch strings.ToLower(args[0]) {
	case "create":
		o.Printf("Saved schematic %q.", result.Name)
	case "paste":
		o.Printf("Pasted schematic %q.", result.Name)
	case "delete":
		o.Printf("Deleted schematic %q.", result.Name)
	case "list":
		o.Print("Schematics: " + strings.Join(result.Names, ", "))
	}
}

// UndoCommand implements //undo [b] — reverts the last edit; "b" targets the brush stack.
type UndoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c UndoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	if err := service.Undo(tx, session.Ensure(p), optionalB(c.Target)); err != nil {
		o.Error(err)
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
	if err := service.Redo(tx, session.Ensure(p), optionalB(c.Target)); err != nil {
		o.Error(err)
		return
	}
	o.Print("Redo successful.")
}
