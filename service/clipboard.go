package service

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// Copy stores the current selection on s's clipboard. Optional args of "only <blocks>"
// restrict the copy to those block types.
func Copy(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction, args []string) (CopyResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return CopyResult{}, err
	}
	only := len(args) > 0 && strings.EqualFold(args[0], "only")
	mask := edit.BlockMask{All: true, IncludeAir: true}
	if only {
		if len(args) < 2 {
			return CopyResult{}, fmt.Errorf("copy only requires block types")
		}
		blocks, err := parse.ParseBlockList(strings.Join(args[1:], " "))
		if err != nil {
			return CopyResult{}, err
		}
		mask = edit.BlockMask{Blocks: blocks}
	}
	cb := edit.CopySelection(tx, area, origin, dir, mask, only)
	s.SetClipboard(cb)
	return CopyResult{Copied: len(cb.Entries)}, nil
}

// Paste writes s's clipboard at origin, rotated to match dir. The "-a" flag in
// args skips writing air. Returns ErrClipboardEmpty if no clipboard is set.
func Paste(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction, args []string) (ChangeResult, error) {
	cb, ok := s.Clipboard()
	if !ok {
		return ChangeResult{}, ErrClipboardEmpty
	}
	batch := history.NewBatch(false)
	if err := edit.PasteClipboard(tx, cb, origin, dir, HasFlag(args, "-a"), batch); err != nil {
		return ChangeResult{}, err
	}
	return record(s, batch), nil
}

// Cut copies the selection to s's clipboard (including air) and clears it to air.
func Cut(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	cb := edit.CopySelection(tx, area, origin, dir, edit.BlockMask{All: true, IncludeAir: true}, false)
	s.SetClipboard(cb)
	batch := history.NewBatch(false)
	edit.ClearArea(tx, area, batch)
	return record(s, batch), nil
}

// Schematic dispatches the //schematic subcommands: create, paste, delete, list.
// args[0] selects the subcommand; args[1] is the schematic name when required.
func Schematic(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction, args []string) (SchematicResult, error) {
	if len(args) == 0 {
		return SchematicResult{}, fmt.Errorf("usage: //schematic <create|paste|delete|list> [name] [-a]")
	}
	switch strings.ToLower(args[0]) {
	case "create":
		if len(args) < 2 {
			return SchematicResult{}, fmt.Errorf("schematic create requires a name")
		}
		area, err := selectedArea(s)
		if err != nil {
			return SchematicResult{}, err
		}
		cb := edit.CopySelection(tx, area, origin, dir, edit.BlockMask{All: true, IncludeAir: true}, false)
		if err := edit.SaveSchematic(args[1], cb); err != nil {
			return SchematicResult{}, err
		}
		return SchematicResult{Name: args[1]}, nil
	case "paste":
		if len(args) < 2 {
			return SchematicResult{}, fmt.Errorf("schematic paste requires a name")
		}
		cb, err := edit.LoadSchematic(args[1])
		if err != nil {
			return SchematicResult{}, err
		}
		batch := history.NewBatch(false)
		if err := edit.PasteClipboard(tx, cb, origin, dir, HasFlag(args[2:], "-a"), batch); err != nil {
			return SchematicResult{}, err
		}
		result := record(s, batch)
		return SchematicResult{Name: args[1], Changed: result.Changed}, nil
	case "delete":
		if len(args) < 2 {
			return SchematicResult{}, fmt.Errorf("schematic delete requires a name")
		}
		if err := edit.DeleteSchematic(args[1]); err != nil {
			return SchematicResult{}, err
		}
		return SchematicResult{Name: args[1]}, nil
	case "list":
		names, err := edit.ListSchematics()
		if err != nil {
			return SchematicResult{}, err
		}
		return SchematicResult{Names: names}, nil
	default:
		return SchematicResult{}, fmt.Errorf("unknown schematic subcommand")
	}
}

// Undo reverts the most recent batch on s's stack. If brush is true the brush
// stack is used. Returns ErrNothingToUndo when the stack is empty.
func Undo(tx *world.Tx, s Session, brush bool) error {
	if !s.Undo(tx, brush) {
		return ErrNothingToUndo
	}
	return nil
}

// Redo restores the most recently undone batch. If brush is true the brush stack
// is used. Returns ErrNothingToRedo when the stack is empty.
func Redo(tx *world.Tx, s Session, brush bool) error {
	if !s.Redo(tx, brush) {
		return ErrNothingToRedo
	}
	return nil
}
