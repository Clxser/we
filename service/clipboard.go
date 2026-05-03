package service

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/parse"
)

// Copy stores the current selection on s's clipboard. Optional args of "only <blocks>"
// restrict the copy to those block types. The clipboard is anchored at the
// selection centre so shapes paste around the target instead of from one corner.
func Copy(tx *world.Tx, s Session, _ cube.Pos, dir cube.Direction, args []string) (CopyResult, error) {
	area, err := selectedReadArea(s)
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
	cb := edit.CopySelection(tx, area, areaCenter(area), dir, mask, only)
	s.SetClipboard(cb)
	return CopyResult{Copied: len(cb.Entries)}, nil
}

// Paste writes s's clipboard at origin, rotated to match dir. The "-a" flag in
// args skips writing air. Returns ErrClipboardEmpty if no clipboard is set.
func Paste(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return PasteWithOptions(tx, s, origin, dir, args, opts)
}

func PasteWithOptions(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction, args []string, opts EditOptions) (ChangeResult, error) {
	cb, ok := s.Clipboard()
	if !ok {
		return ChangeResult{}, ErrClipboardEmpty
	}
	noAir := HasFlag(args, "-a")
	if err := guardrailsFor(s).CheckEditSubChunks(edit.PasteSubChunkCount(cb, origin, dir, noAir)); err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	if err := edit.PasteClipboard(tx, cb, origin, dir, noAir, batch); err != nil {
		return ChangeResult{}, err
	}
	return finishEdit(s, batch, len(cb.Entries)), nil
}

// ClearClipboard removes the stored clipboard from s.
func ClearClipboard(s Session) {
	s.SetClipboard(nil)
}

// Cut copies the selection to s's clipboard (including air) and clears it to air.
func Cut(tx *world.Tx, s Session, _ cube.Pos, dir cube.Direction) (ChangeResult, error) {
	return CutWithOptions(tx, s, cube.Pos{}, dir, EditOptions{})
}

func CutWithOptions(tx *world.Tx, s Session, _ cube.Pos, dir cube.Direction, opts EditOptions) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	cb := edit.CopySelection(tx, area, areaCenter(area), dir, edit.BlockMask{All: true, IncludeAir: true}, false)
	s.SetClipboard(cb)
	batch := historyBatch(opts)
	edit.ClearArea(tx, area, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}

// Schematic dispatches the //schematic subcommands: create, paste, delete, list.
// args[0] selects the subcommand; args[1] is the schematic name when required.
func Schematic(tx *world.Tx, s Session, origin cube.Pos, dir cube.Direction, store edit.SchematicStore, args []string) (SchematicResult, error) {
	if store == nil {
		store = edit.DefaultSchematicStore()
	}
	if len(args) == 0 {
		return SchematicResult{}, fmt.Errorf("usage: //schematic <create|paste|delete|list> [name] [-a]")
	}
	switch strings.ToLower(args[0]) {
	case "create":
		if len(args) < 2 {
			return SchematicResult{}, fmt.Errorf("schematic create requires a name")
		}
		area, err := selectedReadArea(s)
		if err != nil {
			return SchematicResult{}, err
		}
		cb := edit.CopySelection(tx, area, areaCenter(area), dir, edit.BlockMask{All: true, IncludeAir: true}, false)
		if err := store.Save(args[1], cb); err != nil {
			return SchematicResult{}, err
		}
		return SchematicResult{Name: args[1]}, nil
	case "paste":
		if len(args) < 2 {
			return SchematicResult{}, fmt.Errorf("schematic paste requires a name")
		}
		pasteArgs, opts := ParseEditOptions(args[2:])
		cb, err := store.Load(args[1])
		if err != nil {
			return SchematicResult{}, err
		}
		noAir := HasFlag(pasteArgs, "-a")
		if err := guardrailsFor(s).CheckEditSubChunks(edit.PasteSubChunkCount(cb, origin, dir, noAir)); err != nil {
			return SchematicResult{}, err
		}
		batch := historyBatch(opts)
		if err := edit.PasteClipboard(tx, cb, origin, dir, noAir, batch); err != nil {
			return SchematicResult{}, err
		}
		result := finishEdit(s, batch, len(cb.Entries))
		return SchematicResult{Name: args[1], Changed: result.Changed}, nil
	case "delete":
		if len(args) < 2 {
			return SchematicResult{}, fmt.Errorf("schematic delete requires a name")
		}
		if err := store.Delete(args[1]); err != nil {
			return SchematicResult{}, err
		}
		return SchematicResult{Name: args[1]}, nil
	case "list":
		names, err := store.List()
		if err != nil {
			return SchematicResult{}, err
		}
		return SchematicResult{Names: names}, nil
	default:
		return SchematicResult{}, fmt.Errorf("unknown schematic subcommand")
	}
}

func areaCenter(area geo.Area) cube.Pos {
	return cube.Pos{
		midpoint(area.Min[0], area.Max[0]),
		midpoint(area.Min[1], area.Max[1]),
		midpoint(area.Min[2], area.Max[2]),
	}
}

func midpoint(minimum, maximum int) int {
	return minimum + (maximum-minimum)/2
}

// Undo reverts the most recent batch. If brush is true, only the brush stack is
// used. Otherwise the newest command or brush batch is undone.
func Undo(tx *world.Tx, s Session, brush bool) error {
	if !s.Undo(tx, brush) {
		return ErrNothingToUndo
	}
	return nil
}

// Redo restores the most recently undone batch. If brush is true, only the
// brush stack is used. Otherwise the latest undone command or brush batch is
// redone.
func Redo(tx *world.Tx, s Session, brush bool) error {
	if !s.Redo(tx, brush) {
		return ErrNothingToRedo
	}
	return nil
}
