package cmd

import (
	"fmt"
	"strconv"
	"strings"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
	"github.com/df-mc/we/session"
)

func selectedArea(p *player.Player, o *dcf.Output) (geo.Area, bool) {
	a, ok := session.Ensure(p).SelectionArea()
	if !ok {
		o.Error("pos1 and pos2 must be set first")
		return geo.Area{}, false
	}
	return a, true
}

func record(p *player.Player, batch *history.Batch) {
	session.Ensure(p).Record(batch)
}

func optionalB(o dcf.Optional[string]) bool {
	v, ok := o.Load()
	return ok && strings.EqualFold(v, "b")
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(a, flag) {
			return true
		}
	}
	return false
}

func removeFlags(args []string, flags ...string) []string {
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

func parseMaskTo(args []string) (edit.BlockMask, []world.Block, error) {
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		return edit.BlockMask{}, nil, err
	}
	to, err := parse.ParseBlockList(strings.Join(args[1:], " "))
	return mask, to, err
}

func validAxis(axis string) bool {
	switch strings.ToLower(axis) {
	case "x", "y", "z":
		return true
	default:
		return false
	}
}

func parseShapeArgs(kind edit.ShapeKind, args []string, hollow bool) (edit.ShapeSpec, []world.Block, error) {
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
