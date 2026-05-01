package service

import (
	"errors"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

var (
	ErrSelectionRequired = errors.New("pos1 and pos2 must be set first")
	ErrClipboardEmpty    = errors.New("clipboard is empty")
	ErrNothingToUndo     = errors.New("nothing to undo")
	ErrNothingToRedo     = errors.New("nothing to redo")
)

type Session interface {
	SelectionArea() (geo.Area, bool)
	PosCorners() (pos1, pos2 cube.Pos, ok bool)
	SetClipboard(*edit.Clipboard)
	Clipboard() (*edit.Clipboard, bool)
	Record(*history.Batch) int
	Undo(*world.Tx, bool) bool
	Redo(*world.Tx, bool) bool
}

type ChangeResult struct {
	Changed int
}

type PositionResult struct {
	Changed int
	Pos     cube.Pos
}

type CopyResult struct {
	Copied int
}

type SchematicResult struct {
	Name    string
	Names   []string
	Changed int
}

func selectedArea(s Session) (geo.Area, error) {
	area, ok := s.SelectionArea()
	if !ok {
		return geo.Area{}, ErrSelectionRequired
	}
	return area, nil
}

func record(s Session, batch *history.Batch) ChangeResult {
	s.Record(batch)
	return ChangeResult{Changed: batch.Len()}
}

func HasFlag(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(a, flag) {
			return true
		}
	}
	return false
}

func RemoveFlags(args []string, flags ...string) []string {
	var out []string
	for _, a := range args {
		remove := false
		for _, f := range flags {
			if strings.EqualFold(a, f) {
				remove = true
				break
			}
		}
		if !remove {
			out = append(out, a)
		}
	}
	return out
}

func ParseMaskTo(args []string) (edit.BlockMask, []world.Block, error) {
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		return edit.BlockMask{}, nil, err
	}
	to, err := parse.ParseBlockList(strings.Join(args[1:], " "))
	return mask, to, err
}

func ValidAxis(axis string) bool {
	switch strings.ToLower(axis) {
	case "x", "y", "z":
		return true
	default:
		return false
	}
}
