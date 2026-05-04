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
	timer := startOperation()
	result, err := service.Copy(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction(), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Copied %d blocks.", result.Copied)
}

// PasteCommand implements //paste [-a] — pastes the clipboard at the player's position.
type PasteCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c PasteCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	result, err := service.Paste(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction(), strings.Fields(string(c.Args)))
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Pasted %d blocks.", result.Changed)
}

// ClearClipboardCommand implements //clearclipboard — clears the player's clipboard.
type ClearClipboardCommand struct{ playerCommand }

func (ClearClipboardCommand) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	service.ClearClipboard(session.Ensure(p))
	o.Print("Clipboard cleared.")
}

// CutCommand implements //cut [-noundo] — copies the selection to the clipboard, then clears it.
type CutCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c CutCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args, opts := service.ParseEditOptions(strings.Fields(string(c.Args)))
	if len(args) != 0 {
		o.Error("usage: //cut [-noundo]")
		return
	}
	timer := startOperation()
	result, err := service.CutWithOptions(tx, session.Ensure(p), cube.PosFromVec3(p.Position()), p.Rotation().Direction(), opts)
	if err != nil {
		o.Error(err)
		return
	}
	timer.Printf(o, "Cut %d blocks.", result.Changed)
}

// SchematicCommand implements //schematic <create|paste|delete|list> — disk-backed selection storage.
type SchematicCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c SchematicCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	s := session.Ensure(p)
	timer := startOperation()
	result, err := service.Schematic(tx, s, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), s.SchematicStore(), args)
	if err != nil {
		o.Error(err)
		return
	}
	switch strings.ToLower(args[0]) {
	case "create":
		timer.Printf(o, "Saved schematic %q.", result.Name)
	case "paste":
		timer.Printf(o, "Pasted schematic %q.", result.Name)
	case "delete":
		timer.Printf(o, "Deleted schematic %q.", result.Name)
	case "list":
		o.Print("Schematics: " + strings.Join(result.Names, ", "))
	}
}

// UndoCommand implements //undo [b] — reverts the last edit; "b" targets only the brush stack.
type UndoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c UndoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	if err := service.Undo(tx, session.Ensure(p), optionalB(c.Target)); err != nil {
		o.Error(err)
		return
	}
	timer.Print(o, "Undo successful.")
}

// RedoCommand implements //redo [b] — restores the last undone edit; "b" targets only the brush stack.
type RedoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c RedoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	timer := startOperation()
	if err := service.Redo(tx, session.Ensure(p), optionalB(c.Target)); err != nil {
		o.Error(err)
		return
	}
	timer.Print(o, "Redo successful.")
}
