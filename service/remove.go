package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/parse"
)

const defaultRemoveHeight = 64

// RemoveAbove clears a vertical column or square prism above center.
func RemoveAbove(tx *world.Tx, s Session, center cube.Pos, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return RemoveAboveWithOptions(tx, s, center, args, opts)
}

func RemoveAboveWithOptions(tx *world.Tx, s Session, center cube.Pos, args []string, opts EditOptions) (ChangeResult, error) {
	height, radius, err := parseHeightRadius(args, "//removeabove")
	if err != nil {
		return ChangeResult{}, err
	}
	area := geo.NewArea(center[0]-radius, center[1]+1, center[2]-radius, center[0]+radius, center[1]+height, center[2]+radius)
	if err := checkArea(guardrailsFor(s), area); err != nil {
		return ChangeResult{}, err
	}
	batch, err := historyBatchForSize(opts, area.Volume())
	if err != nil {
		return ChangeResult{}, err
	}
	edit.ClearArea(tx, area, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}

// RemoveBelow clears a vertical column or square prism below center.
func RemoveBelow(tx *world.Tx, s Session, center cube.Pos, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return RemoveBelowWithOptions(tx, s, center, args, opts)
}

func RemoveBelowWithOptions(tx *world.Tx, s Session, center cube.Pos, args []string, opts EditOptions) (ChangeResult, error) {
	height, radius, err := parseHeightRadius(args, "//removebelow")
	if err != nil {
		return ChangeResult{}, err
	}
	area := geo.NewArea(center[0]-radius, center[1]-height, center[2]-radius, center[0]+radius, center[1]-1, center[2]+radius)
	if err := checkArea(guardrailsFor(s), area); err != nil {
		return ChangeResult{}, err
	}
	batch, err := historyBatchForSize(opts, area.Volume())
	if err != nil {
		return ChangeResult{}, err
	}
	edit.ClearArea(tx, area, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}

func parseHeightRadius(args []string, command string) (height, radius int, err error) {
	height, radius = defaultRemoveHeight, 0
	if len(args) > 2 {
		return 0, 0, fmt.Errorf("usage: %s [height] [radius]", command)
	}
	if len(args) >= 1 {
		height, err = strconv.Atoi(args[0])
		if err != nil || height < 1 {
			return 0, 0, fmt.Errorf("height must be positive")
		}
	}
	if len(args) == 2 {
		radius, err = strconv.Atoi(args[1])
		if err != nil || radius < 0 {
			return 0, 0, fmt.Errorf("radius must be non-negative")
		}
	}
	return height, radius, nil
}

// RemoveNear clears matching blocks in a sphere around center.
func RemoveNear(tx *world.Tx, s Session, center cube.Pos, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return RemoveNearWithOptions(tx, s, center, args, opts)
}

func RemoveNearWithOptions(tx *world.Tx, s Session, center cube.Pos, args []string, opts EditOptions) (ChangeResult, error) {
	if len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //removenear <blocks> <radius>")
	}
	radius, err := strconv.Atoi(args[len(args)-1])
	if err != nil || radius < 1 {
		return ChangeResult{}, fmt.Errorf("radius must be positive")
	}
	blocks, err := parse.ParseBlockList(strings.Join(args[:len(args)-1], " "))
	if err != nil {
		return ChangeResult{}, err
	}
	area := geo.NewArea(center[0]-radius, center[1]-radius, center[2]-radius, center[0]+radius, center[1]+radius, center[2]+radius)
	if err := checkArea(guardrailsFor(s), area); err != nil {
		return ChangeResult{}, err
	}
	batch, err := historyBatchForSize(opts, area.Volume())
	if err != nil {
		return ChangeResult{}, err
	}
	edit.RemoveNear(tx, center, radius, edit.BlockMask{Blocks: blocks}, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}

// Naturalize converts selected terrain columns into grass, dirt, then stone.
func Naturalize(tx *world.Tx, s Session) (ChangeResult, error) {
	return NaturalizeWithOptions(tx, s, EditOptions{})
}

func NaturalizeWithOptions(tx *world.Tx, s Session, opts EditOptions) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch, err := historyBatchForSize(opts, area.Volume())
	if err != nil {
		return ChangeResult{}, err
	}
	edit.Naturalize(tx, area, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}
