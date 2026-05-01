package service

import (
	"errors"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// Sentinel errors returned by service functions. Adapters check these to format
// user-facing messages without parsing strings.
var (
	ErrSelectionRequired = errors.New("pos1 and pos2 must be set first")
	ErrClipboardEmpty    = errors.New("clipboard is empty")
	ErrNothingToUndo     = errors.New("nothing to undo")
	ErrNothingToRedo     = errors.New("nothing to redo")
)

// Session is the subset of per-player state that service functions need.
//
// The session package satisfies it; callers may also pass test doubles. Methods
// are expected to be safe under their own locking; service functions do not
// hold a Session lock across edit operations.
type Session interface {
	SelectionArea() (geo.Area, bool)
	PosCorners() (pos1, pos2 cube.Pos, ok bool)
	SetClipboard(*edit.Clipboard)
	Clipboard() (*edit.Clipboard, bool)
	Record(*history.Batch) int
	Undo(*world.Tx, bool) bool
	Redo(*world.Tx, bool) bool
}

// ChangeResult reports how many positions an edit modified.
type ChangeResult struct {
	Changed int
}

// PositionResult reports how many positions changed and the anchor block written.
type PositionResult struct {
	Changed int
	Pos     cube.Pos
}

// CopyResult reports how many entries were copied to the clipboard.
type CopyResult struct {
	Copied int
}

// SchematicResult reports the outcome of a //schematic subcommand.
//
// Name is set for create, paste, and delete; Names is set for list; Changed is
// set for paste.
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
	if err := guardrailsFor(s).CheckSelectionVolume(area.Volume()); err != nil {
		return geo.Area{}, err
	}
	return area, nil
}

type guardedSession interface {
	Guardrails() guardrail.Limits
}

func guardrailsFor(s Session) guardrail.Limits {
	if guarded, ok := s.(guardedSession); ok {
		return guarded.Guardrails()
	}
	return guardrail.Limits{}
}

func record(s Session, batch *history.Batch) ChangeResult {
	s.Record(batch)
	return ChangeResult{Changed: batch.Len()}
}

// HasFlag reports whether args contains flag (case-insensitive).
func HasFlag(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(a, flag) {
			return true
		}
	}
	return false
}

// RemoveFlags returns args with any case-insensitive matches of flags filtered out.
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

// ParseMaskTo parses args[0] as a mask and the rest as a destination block list.
func ParseMaskTo(args []string) (edit.BlockMask, []world.Block, error) {
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		return edit.BlockMask{}, nil, err
	}
	to, err := parse.ParseBlockList(strings.Join(args[1:], " "))
	return mask, to, err
}

// ValidAxis reports whether axis is one of "x", "y", or "z" (case-insensitive).
func ValidAxis(axis string) bool {
	switch strings.ToLower(axis) {
	case "x", "y", "z":
		return true
	default:
		return false
	}
}
