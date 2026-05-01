package visual

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/we/geo"
	"github.com/go-gl/mathgl/mgl64"
)

// Box describes a cuboid in world-space coordinates. Max is exclusive, which
// makes block selections easy to represent: selected block max + 1 is the
// visible outer edge of the selection.
type Box struct {
	Min, Max mgl64.Vec3
}

// BlockBox returns a Box that outlines all blocks in the inclusive range from
// min to max.
func BlockBox(min, max cube.Pos) Box {
	area := geo.NewArea(min[0], min[1], min[2], max[0], max[1], max[2])
	return AreaBox(area)
}

// AreaBox returns a Box that outlines all blocks in area.
func AreaBox(area geo.Area) Box {
	return Box{
		Min: area.Min.Vec3(),
		Max: area.Max.Vec3().Add(mgl64.Vec3{1, 1, 1}),
	}
}

// BoxSegments returns the 12 wireframe edges for box.
func BoxSegments(box Box) []Segment {
	minX, minY, minZ := box.Min[0], box.Min[1], box.Min[2]
	maxX, maxY, maxZ := box.Max[0], box.Max[1], box.Max[2]

	corners := [8]mgl64.Vec3{
		{minX, minY, minZ},
		{maxX, minY, minZ},
		{maxX, minY, maxZ},
		{minX, minY, maxZ},
		{minX, maxY, minZ},
		{maxX, maxY, minZ},
		{maxX, maxY, maxZ},
		{minX, maxY, maxZ},
	}

	return []Segment{
		{corners[0], corners[1]},
		{corners[1], corners[2]},
		{corners[2], corners[3]},
		{corners[3], corners[0]},
		{corners[4], corners[5]},
		{corners[5], corners[6]},
		{corners[6], corners[7]},
		{corners[7], corners[4]},
		{corners[0], corners[4]},
		{corners[1], corners[5]},
		{corners[2], corners[6]},
		{corners[3], corners[7]},
	}
}
