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

type denseBlockStructure struct {
	d       [3]int
	entries []denseBlockEntry
}

func (s denseBlockStructure) Dimensions() [3]int { return s.d }

func (s denseBlockStructure) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	entry := s.entries[(x*s.d[1]+y)*s.d[2]+z]
	return entry.Block, entry.Liq
}

// writeDenseArea applies a full cuboid through Dragonfly's chunk-batched
// BuildStructure path. It snapshots every position before the structure write
// and refreshes after snapshots afterwards so undo/redo behavior stays identical
// to repeated Batch.SetBlock calls.
func writeDenseArea(tx *world.Tx, area geo.Area, blockAt func(cube.Pos) world.Block, batch *history.Batch) {
	n := int(area.Volume())
	batch.Grow(n)
	entries := make([]denseBlockEntry, 0, n)
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		block := blockAt(pos)
		var liq world.Liquid
		if block == nil {
			block = mcblock.Air{}
		}
		entries = append(entries, denseBlockEntry{Pos: pos, Index: batch.EnsurePos(tx, pos), Block: block, Liq: liq})
	})
	tx.BuildStructure(area.Min, denseBlockStructure{d: [3]int{area.Dx(), area.Dy(), area.Dz()}, entries: entries})
	for _, entry := range entries {
		batch.SetAfterForIndex(tx, entry.Index, entry.Pos)
	}
}

// writeDenseBuffer applies a complete rectangular clipboard/buffer snapshot
// through BuildStructure. It returns false when entries are sparse or have
// duplicate offsets, in which case callers should fall back to SetBlock writes.
func writeDenseBuffer(tx *world.Tx, origin cube.Pos, entries []bufferEntry, batch *history.Batch) bool {
	min, dims, ordered, ok := denseBufferLayout(entries)
	if !ok {
		return false
	}
	n := len(ordered)
	batch.Grow(n)
	denseEntries := make([]denseBlockEntry, n)
	for i, entry := range ordered {
		pos := origin.Add(entry.Offset)
		block, liq := structureLayers(entry)
		denseEntries[i] = denseBlockEntry{Pos: pos, Index: batch.EnsurePos(tx, pos), Block: block, Liq: liq}
	}
	tx.BuildStructure(origin.Add(min), denseBlockStructure{d: dims, entries: denseEntries})
	for _, entry := range denseEntries {
		batch.SetAfterForIndex(tx, entry.Index, entry.Pos)
	}
	return true
}

func denseBufferLayout(entries []bufferEntry) (cube.Pos, [3]int, []bufferEntry, bool) {
	if len(entries) == 0 {
		return cube.Pos{}, [3]int{}, nil, false
	}
	lo, hi := entries[0].Offset, entries[0].Offset
	for _, entry := range entries[1:] {
		lo = cube.Pos{min(lo[0], entry.Offset[0]), min(lo[1], entry.Offset[1]), min(lo[2], entry.Offset[2])}
		hi = cube.Pos{max(hi[0], entry.Offset[0]), max(hi[1], entry.Offset[1]), max(hi[2], entry.Offset[2])}
	}
	dims := [3]int{hi[0] - lo[0] + 1, hi[1] - lo[1] + 1, hi[2] - lo[2] + 1}
	volume := dims[0] * dims[1] * dims[2]
	if volume != len(entries) {
		return cube.Pos{}, [3]int{}, nil, false
	}
	ordered := make([]bufferEntry, volume)
	seen := make([]bool, volume)
	for _, entry := range entries {
		i := denseIndex(entry.Offset, lo, dims)
		if seen[i] {
			return cube.Pos{}, [3]int{}, nil, false
		}
		seen[i] = true
		ordered[i] = entry
	}
	return lo, dims, ordered, true
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
