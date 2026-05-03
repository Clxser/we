package service

import (
	"fmt"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/parse"
)

// Line draws a line of args[0] blocks with args[1] thickness between pos1 and pos2.
func Line(tx *world.Tx, s Session, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return LineWithOptions(tx, s, args, opts)
}

func LineWithOptions(tx *world.Tx, s Session, args []string, opts EditOptions) (ChangeResult, error) {
	if len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //line <blocks> <thickness>")
	}
	blocks, err := parse.ParseBlockList(args[0])
	if err != nil {
		return ChangeResult{}, err
	}
	thickness, err := strconv.Atoi(args[1])
	if err != nil {
		return ChangeResult{}, err
	}
	pos1, pos2, ok := s.PosCorners()
	if !ok {
		return ChangeResult{}, ErrSelectionRequired
	}
	batch := historyBatch(opts)
	edit.Line(tx, pos1, pos2, thickness, blocks, batch)
	steps := max(abs(pos2[0]-pos1[0]), max(abs(pos2[1]-pos1[1]), abs(pos2[2]-pos1[2])))
	if steps == 0 {
		steps = 1
	}
	return finishEdit(s, batch, (steps+1)*max(1, thickness)*max(1, thickness)*max(1, thickness)), nil
}

// Shape applies a primitive of kind centred at anchor. The "-h" flag in args
// switches to hollow placement.
func Shape(tx *world.Tx, s Session, anchor cube.Pos, kind edit.ShapeKind, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return ShapeWithOptions(tx, s, anchor, kind, args, opts)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func ShapeWithOptions(tx *world.Tx, s Session, anchor cube.Pos, kind edit.ShapeKind, args []string, opts EditOptions) (ChangeResult, error) {
	hollow := HasFlag(args, "-h")
	args = RemoveFlags(args, "-h")
	spec, blocks, err := ParseShapeArgs(kind, args, hollow)
	if err != nil {
		return ChangeResult{}, err
	}
	bounds := spec.Bounds(anchor)
	limits := guardrailsFor(s)
	if err := limits.CheckShapeVolume(bounds.Volume()); err != nil {
		return ChangeResult{}, err
	}
	if err := limits.CheckEditSubChunks(bounds.SubChunkCount()); err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.ApplyShape(tx, anchor, spec, blocks, batch)
	return finishEdit(s, batch, int(spec.Bounds(anchor).Volume())), nil
}

// ParseShapeArgs parses a shape's argument list into a ShapeSpec and block list.
// Sphere, cylinder, and cone expect <blocks> <radius> <height>; pyramid and cube
// expect <blocks> <length> <width> <height>.
func ParseShapeArgs(kind edit.ShapeKind, args []string, hollow bool) (edit.ShapeSpec, []world.Block, error) {
	if len(args) < 3 {
		return edit.ShapeSpec{}, nil, fmt.Errorf("not enough shape arguments")
	}
	blocks, err := parse.ParseBlockList(args[0])
	if err != nil {
		return edit.ShapeSpec{}, nil, err
	}
	spec := edit.ShapeSpec{Kind: kind, Hollow: hollow}
	switch kind {
	case edit.ShapeSphere:
		r, err1 := strconv.Atoi(args[1])
		h, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			return edit.ShapeSpec{}, nil, fmt.Errorf("radius and height must be numbers")
		}
		spec.Radius, spec.Height = r, h
	case edit.ShapeCylinder, edit.ShapeCone:
		r, err1 := strconv.Atoi(args[1])
		h, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			return edit.ShapeSpec{}, nil, fmt.Errorf("radius and height must be numbers")
		}
		spec.Radius, spec.Height = r, h
	case edit.ShapePyramid, edit.ShapeCube:
		if len(args) < 4 {
			return edit.ShapeSpec{}, nil, fmt.Errorf("length, width, and height are required")
		}
		l, err1 := strconv.Atoi(args[1])
		w, err2 := strconv.Atoi(args[2])
		h, err3 := strconv.Atoi(args[3])
		if err1 != nil || err2 != nil || err3 != nil {
			return edit.ShapeSpec{}, nil, fmt.Errorf("length, width, and height must be numbers")
		}
		spec.Length, spec.Width, spec.Height = l, w, h
	}
	return spec, blocks, nil
}
