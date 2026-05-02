package edit

import (
	"math"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
)

// ShapeKind names the supported shape primitives.
type ShapeKind string

// Supported shape kinds.
const (
	ShapeSphere   ShapeKind = "sphere"
	ShapeCylinder ShapeKind = "cylinder"
	ShapePyramid  ShapeKind = "pyramid"
	ShapeCone     ShapeKind = "cone"
	ShapeCube     ShapeKind = "cube"
)

// ShapeSpec describes a shape's dimensions. Which fields are used depends on Kind:
// sphere/cylinder/cone use Radius and Height; pyramid/cube use Length, Width, Height.
// Hollow restricts placement to the shell.
type ShapeSpec struct {
	Kind          ShapeKind
	Radius        int
	Height        int
	Length, Width int
	Hollow        bool
}

// Bounds returns the inclusive cuboid that contains the shape when centred at anchor.
func (s ShapeSpec) Bounds(anchor cube.Pos) geo.Area {
	switch s.Kind {
	case ShapeSphere:
		h := max(1, s.Height)
		half := h / 2
		return geo.NewArea(anchor[0]-s.Radius, anchor[1]-half, anchor[2]-s.Radius, anchor[0]+s.Radius, anchor[1]+h-half-1, anchor[2]+s.Radius)
	case ShapeCylinder, ShapeCone:
		return geo.NewArea(anchor[0]-s.Radius, anchor[1], anchor[2]-s.Radius, anchor[0]+s.Radius, anchor[1]+max(1, s.Height)-1, anchor[2]+s.Radius)
	case ShapePyramid, ShapeCube:
		l, w, h := max(1, s.Length), max(1, s.Width), max(1, s.Height)
		return geo.NewArea(anchor[0]-l/2, anchor[1], anchor[2]-w/2, anchor[0]+(l-1)/2, anchor[1]+h-1, anchor[2]+(w-1)/2)
	default:
		return geo.NewArea(anchor[0], anchor[1], anchor[2], anchor[0], anchor[1], anchor[2])
	}
}

// Inside reports whether pos lies within the solid volume of the shape at anchor.
func (s ShapeSpec) Inside(anchor, pos cube.Pos) bool {
	inside, _ := s.insideShell(anchor, pos)
	return inside
}

// Shell reports whether pos lies on the outer one-block-thick shell of the shape.
func (s ShapeSpec) Shell(anchor, pos cube.Pos) bool {
	inside, shell := s.insideShell(anchor, pos)
	return inside && shell
}

func (s ShapeSpec) insideShell(anchor, pos cube.Pos) (bool, bool) {
	switch s.Kind {
	case ShapeSphere:
		r := float64(max(1, s.Radius))
		h := float64(max(1, s.Height)) / 2
		if h < 0.5 {
			h = 0.5
		}
		dx, dy, dz := float64(pos[0]-anchor[0]), float64(pos[1]-anchor[1]), float64(pos[2]-anchor[2])
		v := dx*dx/(r*r) + dy*dy/(h*h) + dz*dz/(r*r)
		if v > 1.0 {
			return false, false
		}
		innerR, innerH := math.Max(0.5, r-1), math.Max(0.5, h-1)
		inner := dx*dx/(innerR*innerR)+dy*dy/(innerH*innerH)+dz*dz/(innerR*innerR) <= 1.0
		return true, !inner
	case ShapeCylinder:
		r := max(1, s.Radius)
		h := max(1, s.Height)
		if pos[1] < anchor[1] || pos[1] >= anchor[1]+h {
			return false, false
		}
		dx, dz := pos[0]-anchor[0], pos[2]-anchor[2]
		d2 := dx*dx + dz*dz
		if d2 > r*r {
			return false, false
		}
		return true, pos[1] == anchor[1] || pos[1] == anchor[1]+h-1 || d2 > (r-1)*(r-1)
	case ShapeCone:
		h := max(1, s.Height)
		if pos[1] < anchor[1] || pos[1] >= anchor[1]+h {
			return false, false
		}
		layer := pos[1] - anchor[1]
		rr := float64(max(1, s.Radius)) * (1 - float64(layer)/float64(h))
		if rr < 0.5 {
			rr = 0.5
		}
		dx, dz := float64(pos[0]-anchor[0]), float64(pos[2]-anchor[2])
		if dx*dx+dz*dz > rr*rr {
			return false, false
		}
		inner := math.Max(0, rr-1)
		return true, layer == 0 || layer == h-1 || dx*dx+dz*dz > inner*inner
	case ShapePyramid:
		h := max(1, s.Height)
		if pos[1] < anchor[1] || pos[1] >= anchor[1]+h {
			return false, false
		}
		layer := pos[1] - anchor[1]
		l := math.Max(1, float64(max(1, s.Length))*(1-float64(layer)/float64(h)))
		w := math.Max(1, float64(max(1, s.Width))*(1-float64(layer)/float64(h)))
		dx, dz := math.Abs(float64(pos[0]-anchor[0])), math.Abs(float64(pos[2]-anchor[2]))
		inside := dx <= l/2 && dz <= w/2
		if !inside {
			return false, false
		}
		return true, layer == 0 || layer == h-1 || dx > l/2-1 || dz > w/2-1
	case ShapeCube:
		bounds := s.Bounds(anchor)
		if !posInInclusiveBox(pos, bounds.Min, bounds.Max) {
			return false, false
		}
		return true, pos[0] == bounds.Min[0] || pos[0] == bounds.Max[0] || pos[1] == bounds.Min[1] || pos[1] == bounds.Max[1] || pos[2] == bounds.Min[2] || pos[2] == bounds.Max[2]
	default:
		return pos == anchor, true
	}
}

// posInInclusiveBox matches cube.Pos.Within for Dragonfly versions that do not define it.
func posInInclusiveBox(p, min, max cube.Pos) bool {
	return p[0] >= min[0] && p[0] <= max[0] &&
		p[1] >= min[1] && p[1] <= max[1] &&
		p[2] >= min[2] && p[2] <= max[2]
}

// ApplyShape writes blocks at every position inside (or on the shell of) spec around anchor.
func ApplyShape(tx *world.Tx, anchor cube.Pos, spec ShapeSpec, blocks []world.Block, batch *history.Batch) {
	area := spec.Bounds(anchor)
	if spec.Kind == ShapeCube && !spec.Hollow {
		writeDenseArea(tx, area, func(cube.Pos) world.Block { return ChooseBlock(blocks, nil) }, batch)
		return
	}
	batch.Grow(int(area.Volume()))
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		if spec.Hollow {
			if !spec.Shell(anchor, pos) {
				return
			}
		} else if !spec.Inside(anchor, pos) {
			return
		}
		batch.SetBlockFast(tx, pos, ChooseBlock(blocks, nil))
	})
}

// ParseShapeKind maps user input to a shape kind.
func ParseShapeKind(s string) ShapeKind {
	switch strings.ToLower(s) {
	case "sphere", "ball":
		return ShapeSphere
	case "cylinder":
		return ShapeCylinder
	case "pyramid":
		return ShapePyramid
	case "cone":
		return ShapeCone
	case "cube", "box":
		return ShapeCube
	default:
		return ShapeKind(strings.ToLower(s))
	}
}
