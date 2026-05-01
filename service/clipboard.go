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

func Undo(tx *world.Tx, s Session, brush bool) error {
	if !s.Undo(tx, brush) {
		return ErrNothingToUndo
	}
	return nil
}

func Redo(tx *world.Tx, s Session, brush bool) error {
	if !s.Redo(tx, brush) {
		return ErrNothingToRedo
	}
	return nil
}
