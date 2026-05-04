package edit

import (
	"math"
	"math/rand"
	"strings"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// BlockMask filters blocks for replace, move, and similar operations.
//
// All matches every block; IncludeAir lets air count as a match. Blocks holds
// explicit block types when neither flag is set (the "only:..." form).
type BlockMask struct {
	All        bool
	IncludeAir bool
	Blocks     []world.Block

	keys      map[parse.BlockKey]struct{}
	airListed bool
}

// Match reports whether b satisfies the mask.
func (m BlockMask) Match(b world.Block) bool {
	if m.All {
		return m.IncludeAir || !parse.IsAir(b)
	}
	if len(m.Blocks) == 0 {
		return !parse.IsAir(b)
	}
	if parse.IsAir(b) && !m.IncludeAir {
		if m.keys != nil {
			return m.airListed
		}
		for _, want := range m.Blocks {
			if parse.IsAir(want) {
				return true
			}
		}
		return false
	}
	if m.keys != nil {
		_, ok := m.keys[parse.BlockKeyOf(b)]
		return ok
	}
	for _, want := range m.Blocks {
		if parse.SameBlock(b, want) {
			return true
		}
	}
	return false
}

// Prepared returns a copy of m with explicit block matches indexed for hot
// loops. Blocks remains intact for API compatibility and diagnostics.
func (m BlockMask) Prepared() BlockMask {
	if m.All || len(m.Blocks) == 0 || m.keys != nil {
		return m
	}
	m.keys = make(map[parse.BlockKey]struct{}, len(m.Blocks))
	for _, b := range m.Blocks {
		if parse.IsAir(b) {
			m.airListed = true
		}
		m.keys[parse.BlockKeyOf(b)] = struct{}{}
	}
	return m
}

// ParseMask parses a mask spec like "all" or "only:stone,dirt" into a BlockMask.
func ParseMask(spec string) (BlockMask, error) {
	spec = strings.TrimSpace(spec)
	if strings.EqualFold(spec, "all") {
		return BlockMask{All: true, IncludeAir: true}, nil
	}
	spec = strings.TrimPrefix(spec, "only:")
	blocks, err := parse.ParseBlockList(spec)
	return BlockMask{Blocks: blocks}.Prepared(), err
}

// ChooseBlock picks one block from a list using r.
func ChooseBlock(blocks []world.Block, r *rand.Rand) world.Block {
	if len(blocks) == 0 {
		return nil
	}
	if len(blocks) == 1 {
		return blocks[0]
	}
	if r == nil {
		return blocks[rand.Intn(len(blocks))]
	}
	return blocks[r.Intn(len(blocks))]
}

// FillArea sets every block in area to a random pick from blocks.
func FillArea(tx *world.Tx, area geo.Area, blocks []world.Block, batch *history.Batch) {
	if len(blocks) <= 1 {
		writeUniformArea(tx, area, ChooseBlock(blocks, nil), batch)
		return
	}
	writeDenseArea(tx, area, func(cube.Pos) world.Block { return ChooseBlock(blocks, nil) }, batch)
}

// ClearArea replaces every block in area with air and removes any liquid layer.
func ClearArea(tx *world.Tx, area geo.Area, batch *history.Batch) {
	writeUniformArea(tx, area, mcblock.Air{}, batch)
}

// Center places one block at the integer-rounded centre of area and returns the position.
func Center(tx *world.Tx, area geo.Area, blocks []world.Block, batch *history.Batch) cube.Pos {
	pos := cube.Pos{
		(area.Min[0] + area.Max[0]) / 2,
		(area.Min[1] + area.Max[1]) / 2,
		(area.Min[2] + area.Max[2]) / 2,
	}
	if batch == nil {
		setBlock(tx, pos, ChooseBlock(blocks, nil))
		return pos
	}
	batch.SetBlock(tx, pos, ChooseBlock(blocks, nil))
	return pos
}

// Walls fills only the outer shell of area's cuboid.
func Walls(tx *world.Tx, area geo.Area, blocks []world.Block, batch *history.Batch) {
	if batch != nil {
		batch.Grow(int(area.Volume()))
	}
	area.Range(func(x, y, z int) {
		if x == area.Min[0] || x == area.Max[0] || y == area.Min[1] || y == area.Max[1] || z == area.Min[2] || z == area.Max[2] {
			setBlockOrBatch(tx, batch, cube.Pos{x, y, z}, ChooseBlock(blocks, nil))
		}
	})
}

// ReplaceArea swaps blocks matching mask inside area for picks from to.
func ReplaceArea(tx *world.Tx, area geo.Area, mask BlockMask, to []world.Block, batch *history.Batch) {
	mask = mask.Prepared()
	if batch != nil {
		batch.Grow(int(area.Volume()))
	}
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		if mask.Match(tx.Block(pos)) {
			setBlockOrBatch(tx, batch, pos, ChooseBlock(to, nil))
		}
	})
}

// ReplaceNear runs ReplaceArea inside a sphere of the given radius around center.
func ReplaceNear(tx *world.Tx, center cube.Pos, radius int, mask BlockMask, to []world.Block, batch *history.Batch) {
	mask = mask.Prepared()
	r2 := radius * radius
	area := geo.NewArea(center[0]-radius, center[1]-radius, center[2]-radius, center[0]+radius, center[1]+radius, center[2]+radius)
	if batch != nil {
		batch.Grow(int(area.Volume()))
	}
	area.Range(func(x, y, z int) {
		dx, dy, dz := x-center[0], y-center[1], z-center[2]
		if dx*dx+dy*dy+dz*dz > r2 {
			return
		}
		pos := cube.Pos{x, y, z}
		if mask.Match(tx.Block(pos)) {
			setBlockOrBatch(tx, batch, pos, ChooseBlock(to, nil))
		}
	})
}

// TopLayer replaces only the topmost matching block in each (x, z) column of area.
func TopLayer(tx *world.Tx, area geo.Area, mask BlockMask, to []world.Block, batch *history.Batch) {
	mask = mask.Prepared()
	if batch != nil {
		batch.Grow(area.Dx() * area.Dz())
	}
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			for y := highestSelectionY(tx, area, x, z); y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				b := tx.Block(pos)
				if parse.IsAir(b) {
					continue
				}
				if mask.Match(b) {
					setBlockOrBatch(tx, batch, pos, ChooseBlock(to, nil))
				}
				break
			}
		}
	}
}

// Overlay places blocks on top of the highest non-air block in each column.
func Overlay(tx *world.Tx, area geo.Area, blocks []world.Block, batch *history.Batch) {
	if batch != nil {
		batch.Grow(area.Dx() * area.Dz())
	}
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			for y := highestSelectionY(tx, area, x, z); y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				if parse.IsAir(tx.Block(pos)) {
					continue
				}
				above := cube.Pos{x, y + 1, z}
				if parse.IsAir(tx.Block(above)) {
					setBlockOrBatch(tx, batch, above, ChooseBlock(blocks, nil))
				}
				break
			}
		}
	}
}

// RemoveNear clears blocks matching mask within a sphere around center.
func RemoveNear(tx *world.Tx, center cube.Pos, radius int, mask BlockMask, batch *history.Batch) {
	mask = mask.Prepared()
	r2 := radius * radius
	area := geo.NewArea(center[0]-radius, center[1]-radius, center[2]-radius, center[0]+radius, center[1]+radius, center[2]+radius)
	if batch != nil {
		batch.Grow(int(area.Volume()))
	}
	area.Range(func(x, y, z int) {
		dx, dy, dz := x-center[0], y-center[1], z-center[2]
		if dx*dx+dy*dy+dz*dz > r2 {
			return
		}
		pos := cube.Pos{x, y, z}
		if mask.Match(tx.Block(pos)) {
			setBlockOrBatch(tx, batch, pos, mcblock.Air{})
			setLiquidOrBatch(tx, batch, pos, nil)
			return
		}
		if liq, ok := tx.Liquid(pos); ok && mask.Match(liq) {
			setLiquidOrBatch(tx, batch, pos, nil)
		}
	})
}

// Naturalize converts each non-air column in area to grass, dirt, then stone.
func Naturalize(tx *world.Tx, area geo.Area, batch *history.Batch) {
	if batch != nil {
		batch.Grow(int(area.Volume()))
	}
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			depth := 0
			for y := highestSelectionY(tx, area, x, z); y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				if parse.IsAir(tx.Block(pos)) {
					continue
				}
				switch {
				case depth == 0:
					setBlockOrBatch(tx, batch, pos, mcblock.Grass{})
				case depth <= 3:
					setBlockOrBatch(tx, batch, pos, mcblock.Dirt{})
				default:
					setBlockOrBatch(tx, batch, pos, mcblock.Stone{})
				}
				depth++
			}
		}
	}
}

// Drain removes water and lava blocks (and standalone liquid layers) within a sphere.
func Drain(tx *world.Tx, center cube.Pos, radius int, batch *history.Batch) {
	r2 := radius * radius
	area := geo.NewArea(center[0]-radius, center[1]-radius, center[2]-radius, center[0]+radius, center[1]+radius, center[2]+radius)
	if batch != nil {
		batch.Grow(int(area.Volume()))
	}
	area.Range(func(x, y, z int) {
		dx, dy, dz := x-center[0], y-center[1], z-center[2]
		if dx*dx+dy*dy+dz*dz > r2 {
			return
		}
		pos := cube.Pos{x, y, z}
		if parse.IsFluidBlock(tx.Block(pos)) {
			setBlockOrBatch(tx, batch, pos, mcblock.Air{})
		}
		if liq, ok := tx.Liquid(pos); ok && parse.IsFluidBlock(liq) {
			setLiquidOrBatch(tx, batch, pos, nil)
		}
	})
}

type bufferEntry struct {
	Offset cube.Pos
	Block  world.Block
	Liquid world.Liquid
	HasLiq bool
}

func copyArea(tx *world.Tx, area geo.Area, origin cube.Pos, mask BlockMask, includeAll bool) []bufferEntry {
	mask = mask.Prepared()
	out := make([]bufferEntry, 0, area.Dx()*area.Dy()*area.Dz())
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		b := tx.Block(pos)
		if !includeAll && !mask.Match(b) {
			return
		}
		liq, ok := tx.Liquid(pos)
		out = append(out, bufferEntry{Offset: pos.Sub(origin), Block: b, Liquid: liq, HasLiq: ok})
	})
	return out
}

func pasteBuffer(tx *world.Tx, origin cube.Pos, entries []bufferEntry, noAir bool, batch *history.Batch) {
	if !noAir && writeDenseBuffer(tx, origin, entries, batch) {
		return
	}
	if batch != nil {
		batch.Grow(len(entries))
	}
	for _, e := range entries {
		if noAir && parse.IsAir(e.Block) && !e.HasLiq {
			continue
		}
		pos := origin.Add(e.Offset)
		setBlockOrBatch(tx, batch, pos, e.Block)
		if e.HasLiq {
			setLiquidOrBatch(tx, batch, pos, e.Liquid)
		} else {
			setLiquidOrBatch(tx, batch, pos, nil)
		}
	}
}

// DirectionVector converts a face to an integer step along one axis.
func DirectionVector(face cube.Face) cube.Pos {
	switch face {
	case cube.FaceDown:
		return cube.Pos{0, -1, 0}
	case cube.FaceUp:
		return cube.Pos{0, 1, 0}
	case cube.FaceNorth:
		return cube.Pos{0, 0, -1}
	case cube.FaceSouth:
		return cube.Pos{0, 0, 1}
	case cube.FaceWest:
		return cube.Pos{-1, 0, 0}
	case cube.FaceEast:
		return cube.Pos{1, 0, 0}
	default:
		return cube.Pos{0, 0, 1}
	}
}

// Move shifts blocks matching mask by dist along dir, clearing the source.
// If noAir is true, source positions whose block is air are not written at the destination.
func Move(tx *world.Tx, area geo.Area, dir cube.Pos, dist int, mask BlockMask, noAir bool, batch *history.Batch) {
	entries := copyArea(tx, area, area.Min, mask, mask.All)
	if batch != nil {
		batch.Grow(len(entries) * 2)
	}
	if mask.All && mask.IncludeAir {
		ClearArea(tx, area, batch)
	} else {
		for _, e := range entries {
			src := area.Min.Add(e.Offset)
			setBlockOrBatch(tx, batch, src, mcblock.Air{})
			setLiquidOrBatch(tx, batch, src, nil)
		}
	}
	dest := area.Min.Add(cube.Pos{dir[0] * dist, dir[1] * dist, dir[2] * dist})
	pasteBuffer(tx, dest, entries, noAir, batch)
}

// Stack repeats area amount times along dir, advancing one full area-length per copy.
func Stack(tx *world.Tx, area geo.Area, dir cube.Pos, amount int, noAir bool, batch *history.Batch) {
	entries := copyArea(tx, area, area.Min, BlockMask{All: true, IncludeAir: true}, true)
	var layout denseBuffer
	dense := false
	if !noAir {
		layout, dense = makeDenseBuffer(entries)
	}
	if batch != nil {
		batch.Grow(len(entries) * amount)
	}
	step := cube.Pos{dir[0] * area.Dx(), dir[1] * area.Dy(), dir[2] * area.Dz()}
	if dir[0] != 0 {
		step = cube.Pos{dir[0] * area.Dx(), 0, 0}
	} else if dir[1] != 0 {
		step = cube.Pos{0, dir[1] * area.Dy(), 0}
	} else if dir[2] != 0 {
		step = cube.Pos{0, 0, dir[2] * area.Dz()}
	}
	var denseEntries []denseBlockEntry
	for i := 1; i <= amount; i++ {
		dest := area.Min.Add(cube.Pos{step[0] * i, step[1] * i, step[2] * i})
		if dense {
			denseEntries = writeDenseBufferLayoutScratch(tx, dest, layout, batch, denseEntries)
			continue
		}
		pasteBuffer(tx, dest, entries, noAir, batch)
	}
}

// RotateCopy rotates blocks inside area in place by degrees around axis (x, y, or z).
func RotateCopy(tx *world.Tx, area geo.Area, degrees int, axis string, batch *history.Batch) {
	entries := copyArea(tx, area, area.Min, BlockMask{All: true, IncludeAir: true}, true)
	axis = strings.ToLower(axis)
	turns := ((degrees/90)%4 + 4) % 4
	center := cube.Pos{(area.Dx() - 1) / 2, (area.Dy() - 1) / 2, (area.Dz() - 1) / 2}
	transform := blockTransform{axis: axis, turns: turns}
	cache := make(blockTransformCache)
	for i := range entries {
		o := entries[i].Offset.Sub(center)
		for t := 0; t < turns; t++ {
			switch axis {
			case "x":
				o = cube.Pos{o[0], -o[2], o[1]}
			case "z":
				o = cube.Pos{-o[1], o[0], o[2]}
			default:
				o = cube.Pos{-o[2], o[1], o[0]}
			}
		}
		entries[i].Offset = o.Add(center)
		entries[i].Block = cache.transform(entries[i].Block, transform)
		if entries[i].HasLiq {
			if l, ok := cache.transform(entries[i].Liquid, transform).(world.Liquid); ok {
				entries[i].Liquid = l
			}
		}
	}
	pasteBuffer(tx, area.Min, entries, false, batch)
}

// FlipCopy mirrors blocks inside area across axis (x, y, or z).
func FlipCopy(tx *world.Tx, area geo.Area, axis string, batch *history.Batch) {
	entries := copyArea(tx, area, area.Min, BlockMask{All: true, IncludeAir: true}, true)
	axis = strings.ToLower(axis)
	transform := blockTransform{axis: axis, flip: true}
	cache := make(blockTransformCache)
	for i := range entries {
		o := entries[i].Offset
		switch axis {
		case "y":
			o[1] = area.Dy() - 1 - o[1]
		case "z":
			o[2] = area.Dz() - 1 - o[2]
		default:
			o[0] = area.Dx() - 1 - o[0]
		}
		entries[i].Offset = o
		entries[i].Block = cache.transform(entries[i].Block, transform)
		if entries[i].HasLiq {
			if l, ok := cache.transform(entries[i].Liquid, transform).(world.Liquid); ok {
				entries[i].Liquid = l
			}
		}
	}
	pasteBuffer(tx, area.Min, entries, false, batch)
}

// Line draws a 3D Bresenham-style line from start to end with the given block thickness.
func Line(tx *world.Tx, start, end cube.Pos, thickness int, blocks []world.Block, batch *history.Batch) {
	if thickness < 1 {
		thickness = 1
	}
	dx, dy, dz := end[0]-start[0], end[1]-start[1], end[2]-start[2]
	steps := max(abs(dx), max(abs(dy), abs(dz)))
	if steps == 0 {
		steps = 1
	}
	minOffset := -(thickness / 2)
	maxOffset := minOffset + thickness - 1
	if batch != nil {
		batch.Grow((steps + 1) * thickness * thickness * thickness)
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(float64(start[0]) + float64(dx)*t))
		y := int(math.Round(float64(start[1]) + float64(dy)*t))
		z := int(math.Round(float64(start[2]) + float64(dz)*t))
		for ox := minOffset; ox <= maxOffset; ox++ {
			for oy := minOffset; oy <= maxOffset; oy++ {
				for oz := minOffset; oz <= maxOffset; oz++ {
					setBlockOrBatch(tx, batch, cube.Pos{x + ox, y + oy, z + oz}, ChooseBlock(blocks, nil))
				}
			}
		}
	}
}

func setBlockOrBatch(tx *world.Tx, batch *history.Batch, pos cube.Pos, block world.Block) {
	if batch == nil {
		setBlock(tx, pos, block)
		return
	}
	batch.SetBlockFast(tx, pos, block)
}

func setLiquidOrBatch(tx *world.Tx, batch *history.Batch, pos cube.Pos, liq world.Liquid) {
	if batch == nil {
		tx.SetLiquid(pos, liq)
		return
	}
	batch.SetLiquid(tx, pos, liq)
}

func setBlock(tx *world.Tx, pos cube.Pos, block world.Block) {
	if block == nil {
		block = mcblock.Air{}
	}
	if liq, ok := block.(world.Liquid); ok {
		tx.SetBlock(pos, nil, nil)
		tx.SetLiquid(pos, liq)
		return
	}
	tx.SetBlock(pos, block, nil)
	if _, ok := tx.Liquid(pos); ok {
		tx.SetLiquid(pos, nil)
	}
}

func highestSelectionY(tx *world.Tx, area geo.Area, x, z int) int {
	return min(tx.HighestBlock(x, z), area.Max[1])
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
