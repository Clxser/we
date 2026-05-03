package edit

import (
	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// denseBlockEntry is one prepared block write plus the batch index that tracks
// its before/after snapshots. Keeping this tiny data carrier explicit lets the
// fast BuildStructure path preserve the same history semantics as Batch.SetBlock.
type denseBlockEntry struct {
	Pos   cube.Pos
	Index int
	Block world.Block
	Liq   world.Liquid
}

type denseBuffer struct {
	min     cube.Pos
	dims    [3]int
	ordered []bufferEntry
}

type denseBlockStructure struct {
	d       [3]int
	entries []denseBlockEntry
}

func (s denseBlockStructure) Dimensions() [3]int { return s.d }

func (s denseBlockStructure) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	entry := s.entries[(x*s.d[1]+y)*s.d[2]+z]
	return entry.Block, entry.Liq
}

type uniformBlockStructure struct {
	d     [3]int
	block world.Block
	liq   world.Liquid
}

func (s uniformBlockStructure) Dimensions() [3]int { return s.d }

func (s uniformBlockStructure) At(_, _, _ int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	return s.block, s.liq
}

func (s uniformBlockStructure) Uniform() (world.Block, world.Liquid, bool) {
	return s.block, s.liq, true
}

type blockFuncStructure struct {
	min     cube.Pos
	d       [3]int
	blockAt func(cube.Pos) world.Block
}

func (s blockFuncStructure) Dimensions() [3]int { return s.d }

func (s blockFuncStructure) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	block := s.blockAt(cube.Pos{s.min[0] + x, s.min[1] + y, s.min[2] + z})
	if block == nil {
		block = mcblock.Air{}
	}
	liq, _ := knownDenseLiquid(block, nil)
	return block, liq
}

func writeUniformArea(tx *world.Tx, area geo.Area, block world.Block, batch *history.Batch) {
	if block == nil {
		block = mcblock.Air{}
	}
	liq, hasLiq := knownDenseLiquid(block, nil)
	structure := uniformBlockStructure{d: [3]int{area.Dx(), area.Dy(), area.Dz()}, block: block, liq: liq}
	if batch == nil {
		buildStructure(tx, area.Min, structure)
		return
	}
	n := int(area.Volume())
	batch.Grow(n)
	appendHistory := batch.Empty()
	worldRange := tx.Range()
	var outOfBounds []denseBlockEntry
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		if appendHistory && !pos.OutOfBounds(worldRange) {
			batch.AppendKnownUnique(tx, pos, block, liq, hasLiq)
			return
		}
		index := batch.EnsurePos(tx, pos)
		if pos.OutOfBounds(worldRange) {
			outOfBounds = append(outOfBounds, denseBlockEntry{Pos: pos, Index: index})
			return
		}
		batch.SetAfterKnownForIndex(index, block, liq, hasLiq)
	})
	buildStructure(tx, area.Min, structure)
	for _, entry := range outOfBounds {
		batch.SetAfterForIndex(tx, entry.Index, entry.Pos)
	}
}

// writeDenseArea applies a full cuboid through Dragonfly's chunk-batched
// BuildStructure path. It snapshots every position before the structure write
// and records the known after snapshots so undo/redo behavior stays identical
// to repeated Batch.SetBlock calls without re-reading the world after writing.
func writeDenseArea(tx *world.Tx, area geo.Area, blockAt func(cube.Pos) world.Block, batch *history.Batch) {
	n := int(area.Volume())
	if batch == nil {
		buildStructure(tx, area.Min, blockFuncStructure{
			min:     area.Min,
			d:       [3]int{area.Dx(), area.Dy(), area.Dz()},
			blockAt: blockAt,
		})
		return
	}
	entries := make([]denseBlockEntry, 0, n)
	batch.Grow(n)
	appendHistory := batch.Empty()
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		block := blockAt(pos)
		var liq world.Liquid
		if block == nil {
			block = mcblock.Air{}
		}
		index := -1
		if !appendHistory {
			index = batch.EnsurePos(tx, pos)
		}
		entries = append(entries, denseBlockEntry{Pos: pos, Index: index, Block: block, Liq: liq})
	})
	worldRange := tx.Range()
	for i := range entries {
		entry := &entries[i]
		liq, hasLiq := knownDenseLiquid(entry.Block, entry.Liq)
		if appendHistory && !entry.Pos.OutOfBounds(worldRange) {
			batch.AppendKnownUnique(tx, entry.Pos, entry.Block, liq, hasLiq)
			continue
		}
		if entry.Index < 0 {
			entry.Index = batch.EnsurePos(tx, entry.Pos)
		}
		if !entry.Pos.OutOfBounds(worldRange) {
			batch.SetAfterKnownForIndex(entry.Index, entry.Block, liq, hasLiq)
		}
	}
	buildStructure(tx, area.Min, denseBlockStructure{d: [3]int{area.Dx(), area.Dy(), area.Dz()}, entries: entries})
	for _, entry := range entries {
		if entry.Pos.OutOfBounds(worldRange) {
			batch.SetAfterForIndex(tx, entry.Index, entry.Pos)
		}
	}
}

// writeDenseBuffer applies a complete rectangular clipboard/buffer snapshot
// through BuildStructure. It returns false when entries are sparse or have
// duplicate offsets, in which case callers should fall back to SetBlock writes.
func writeDenseBuffer(tx *world.Tx, origin cube.Pos, entries []bufferEntry, batch *history.Batch) bool {
	layout, ok := makeDenseBuffer(entries)
	if !ok {
		return false
	}
	writeDenseBufferLayout(tx, origin, layout, batch)
	return true
}

func writeDenseBufferLayout(tx *world.Tx, origin cube.Pos, layout denseBuffer, batch *history.Batch) {
	writeDenseBufferLayoutScratch(tx, origin, layout, batch, nil)
}

func writeDenseBufferLayoutScratch(tx *world.Tx, origin cube.Pos, layout denseBuffer, batch *history.Batch, denseEntries []denseBlockEntry) []denseBlockEntry {
	n := len(layout.ordered)
	if batch == nil {
		if cap(denseEntries) < n {
			denseEntries = make([]denseBlockEntry, n)
		} else {
			denseEntries = denseEntries[:n]
		}
		for i, entry := range layout.ordered {
			block, liq := structureLayers(entry)
			denseEntries[i] = denseBlockEntry{Pos: origin.Add(entry.Offset), Index: -1, Block: block, Liq: liq}
		}
		buildStructure(tx, origin.Add(layout.min), denseBlockStructure{d: layout.dims, entries: denseEntries})
		return denseEntries
	}
	batch.Grow(n)
	if cap(denseEntries) < n {
		denseEntries = make([]denseBlockEntry, n)
	} else {
		denseEntries = denseEntries[:n]
	}
	appendHistory := batch.Empty()
	for i, entry := range layout.ordered {
		pos := origin.Add(entry.Offset)
		block, liq := structureLayers(entry)
		index := -1
		if !appendHistory {
			index = batch.EnsurePos(tx, pos)
		}
		denseEntries[i] = denseBlockEntry{Pos: pos, Index: index, Block: block, Liq: liq}
	}
	worldRange := tx.Range()
	for i := range denseEntries {
		entry := &denseEntries[i]
		liq, hasLiq := knownDenseLiquid(entry.Block, entry.Liq)
		if appendHistory && !entry.Pos.OutOfBounds(worldRange) {
			batch.AppendKnownUnique(tx, entry.Pos, entry.Block, liq, hasLiq)
			continue
		}
		if entry.Index < 0 {
			entry.Index = batch.EnsurePos(tx, entry.Pos)
		}
		if !entry.Pos.OutOfBounds(worldRange) {
			batch.SetAfterKnownForIndex(entry.Index, entry.Block, liq, hasLiq)
		}
	}
	tx.BuildStructure(origin.Add(layout.min), denseBlockStructure{d: layout.dims, entries: denseEntries})
	for _, entry := range denseEntries {
		if entry.Pos.OutOfBounds(worldRange) {
			batch.SetAfterForIndex(tx, entry.Index, entry.Pos)
		}
	}
	return denseEntries
}

func buildStructure(tx *world.Tx, pos cube.Pos, structure world.Structure) {
	tx.BuildStructure(pos, structure)
}

func makeDenseBuffer(entries []bufferEntry) (denseBuffer, bool) {
	if len(entries) == 0 {
		return denseBuffer{}, false
	}
	lo, hi := entries[0].Offset, entries[0].Offset
	for _, entry := range entries[1:] {
		lo = cube.Pos{min(lo[0], entry.Offset[0]), min(lo[1], entry.Offset[1]), min(lo[2], entry.Offset[2])}
		hi = cube.Pos{max(hi[0], entry.Offset[0]), max(hi[1], entry.Offset[1]), max(hi[2], entry.Offset[2])}
	}
	dims := [3]int{hi[0] - lo[0] + 1, hi[1] - lo[1] + 1, hi[2] - lo[2] + 1}
	volume := dims[0] * dims[1] * dims[2]
	if volume != len(entries) {
		return denseBuffer{}, false
	}
	inOrder := true
	for i, entry := range entries {
		if denseIndex(entry.Offset, lo, dims) != i {
			inOrder = false
			break
		}
	}
	if inOrder {
		return denseBuffer{min: lo, dims: dims, ordered: entries}, true
	}
	ordered := make([]bufferEntry, volume)
	seen := make([]bool, volume)
	for _, entry := range entries {
		i := denseIndex(entry.Offset, lo, dims)
		if seen[i] {
			return denseBuffer{}, false
		}
		seen[i] = true
		ordered[i] = entry
	}
	return denseBuffer{min: lo, dims: dims, ordered: ordered}, true
}

func denseIndex(pos, min cube.Pos, dims [3]int) int {
	x, y, z := pos[0]-min[0], pos[1]-min[1], pos[2]-min[2]
	return (x*dims[1]+y)*dims[2] + z
}

func structureLayers(entry bufferEntry) (world.Block, world.Liquid) {
	block := entry.Block
	if block == nil {
		block = mcblock.Air{}
	}
	if !entry.HasLiq {
		return block, nil
	}
	if _, ok := block.(world.Liquid); ok {
		return block, nil
	}
	if parse.IsAir(block) {
		return entry.Liquid, nil
	}
	return block, entry.Liquid
}

func knownDenseLiquid(block world.Block, liq world.Liquid) (world.Liquid, bool) {
	if liq != nil {
		return liq, true
	}
	if l, ok := block.(world.Liquid); ok {
		return l, true
	}
	return nil, false
}
