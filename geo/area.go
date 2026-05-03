package geo

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// Area contains the points with Min.X <= X <= Max.X, Min.Y <= Y <= Max.Y,
// and Min.Z <= Z <= Max.Z
// It is well-formed if Min.X <= Max.X and likewise for Y and Z. Points are
// always well-formed. An area's methods always return well-formed outputs
// for well-formed inputs.
type Area struct {
	Min, Max cube.Pos
}

// NewArea is shorthand for Area{Pos(x0, y0, z0), Pos(x1, y1, z0)}. The returned
// Area has minimum and maximum coordinates swapped if necessary so that
// it is well-formed.
func NewArea(x0, y0, z0, x1, y1, z1 int) Area {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	if z0 > z1 {
		z0, z1 = z1, z0
	}
	return Area{Min: cube.Pos{x0, y0, z0}, Max: cube.Pos{x1, y1, z1}}
}

// String returns a string representation of the Area like "(1,2,3)-(4,5,6)".
func (a Area) String() string {
	return fmt.Sprintf("%v-%v", a.Min, a.Max)
}

// Dx returns the Area's width.
func (a Area) Dx() int {
	return a.Max[0] - a.Min[0] + 1
}

// Dy returns the Area's height.
func (a Area) Dy() int {
	return a.Max[1] - a.Min[1] + 1
}

// Dz returns the Area's length.
func (a Area) Dz() int {
	return a.Max[2] - a.Min[2] + 1
}

// Volume returns the number of block positions in the Area.
func (a Area) Volume() int64 {
	return int64(a.Dx()) * int64(a.Dy()) * int64(a.Dz())
}

// SubChunkCount returns how many unique 16x16x16 sub-chunks this area touches.
// Dragonfly sends changed chunks to clients as sub-chunk cache blobs, so this is
// a better estimate of one-tick network pressure than raw block volume.
func (a Area) SubChunkCount() int64 {
	dx := int64((a.Max[0] >> 4) - (a.Min[0] >> 4) + 1)
	dy := int64((a.Max[1] >> 4) - (a.Min[1] >> 4) + 1)
	dz := int64((a.Max[2] >> 4) - (a.Min[2] >> 4) + 1)
	return dx * dy * dz
}

// UniqueSubChunks returns how many unique 16x16x16 sub-chunks are touched
// across multiple areas.
func UniqueSubChunks(areas ...Area) int64 {
	if len(areas) == 0 {
		return 0
	}
	if len(areas) == 1 {
		return areas[0].SubChunkCount()
	}
	seen := make(map[[3]int]struct{})
	for _, area := range areas {
		for x := area.Min[0] >> 4; x <= area.Max[0]>>4; x++ {
			for y := area.Min[1] >> 4; y <= area.Max[1]>>4; y++ {
				for z := area.Min[2] >> 4; z <= area.Max[2]>>4; z++ {
					seen[[3]int{x, y, z}] = struct{}{}
				}
			}
		}
	}
	return int64(len(seen))
}

// Add returns the area translated by offset.
func (a Area) Add(offset cube.Pos) Area {
	return Area{Min: a.Min.Add(offset), Max: a.Max.Add(offset)}
}

// Union returns the smallest area that contains both a and b.
func (a Area) Union(b Area) Area {
	return NewArea(
		min(a.Min[0], b.Min[0]), min(a.Min[1], b.Min[1]), min(a.Min[2], b.Min[2]),
		max(a.Max[0], b.Max[0]), max(a.Max[1], b.Max[1]), max(a.Max[2], b.Max[2]),
	)
}

// Range iterates over all points where Min.X <= X <= Max.X, Min.Y <= Y <= Max.Y,
// and Min.Z <= Z <= Max.Z and calls f for every X, Y and Z.
func (a Area) Range(f func(x, y, z int)) {
	for x := a.Min[0]; x <= a.Max[0]; x++ {
		for y := a.Min[1]; y <= a.Max[1]; y++ {
			for z := a.Min[2]; z <= a.Max[2]; z++ {
				f(x, y, z)
			}
		}
	}
}
