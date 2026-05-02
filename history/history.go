package history

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
)

const defaultHistoryLimit = 40

type snapshot struct {
	Block  world.Block
	Liquid world.Liquid
	HasLiq bool
	Biome  world.Biome
}

// Change is one position update within a batch.
type Change struct {
	Pos           cube.Pos
	Before, After snapshot
}

// Batch records reversible block edits for undo/redo.
type Batch struct {
	Brush   bool
	changes []Change
	index   map[cube.Pos]int
}

var fastBlockSetOpts = &world.SetOpts{DisableBlockUpdates: true}

// NewBatch creates a batch; brush batches go on the isolated brush undo stack.
func NewBatch(brush bool) *Batch {
	return &Batch{Brush: brush, index: map[cube.Pos]int{}}
}

// Grow preallocates storage for up to n touched positions. It is a performance
// hint for large edits and does not change batch semantics.
func (b *Batch) Grow(n int) {
	if b == nil || n <= 0 {
		return
	}
	if b.index == nil {
		b.index = make(map[cube.Pos]int, n)
	} else if len(b.index) == 0 {
		b.index = make(map[cube.Pos]int, n)
	}
	b.changes = slices.Grow(b.changes, n)
}

func snapshotAt(tx *world.Tx, pos cube.Pos) snapshot {
	liq, ok := tx.Liquid(pos)
	return snapshot{Block: tx.Block(pos), Liquid: liq, HasLiq: ok, Biome: tx.Biome(pos)}
}

func sameSnapshot(a, b snapshot) bool {
	return parse.SameBlock(a.Block, b.Block) && parse.SameLiquid(a.Liquid, a.HasLiq, b.Liquid, b.HasLiq) && parse.SameBiome(a.Biome, b.Biome)
}

func (b *Batch) ensure(tx *world.Tx, pos cube.Pos) int {
	if i, ok := b.index[pos]; ok {
		return i
	}
	i := len(b.changes)
	b.index[pos] = i
	b.changes = append(b.changes, Change{Pos: pos, Before: snapshotAt(tx, pos)})
	return i
}

// SetBlock records and writes a block. Passing nil writes air, matching
// Dragonfly's SetBlock contract.
func (b *Batch) SetBlock(tx *world.Tx, pos cube.Pos, block world.Block) {
	b.setBlock(tx, pos, block, nil)
}

// SetBlockFast records and writes a block without scheduling neighbouring
// block updates. Use it for WorldEdit-style bulk writes where the caller owns
// the full edit operation and wants to avoid per-block physics/update fan-out.
func (b *Batch) SetBlockFast(tx *world.Tx, pos cube.Pos, block world.Block) {
	b.setBlock(tx, pos, block, fastBlockSetOpts)
}

func (b *Batch) setBlock(tx *world.Tx, pos cube.Pos, block world.Block, opts *world.SetOpts) {
	i := b.ensure(tx, pos)
	if liq, ok := block.(world.Liquid); ok {
		tx.SetBlock(pos, nil, opts)
		tx.SetLiquid(pos, liq)
	} else {
		tx.SetBlock(pos, block, opts)
		if _, ok := tx.Liquid(pos); ok {
			tx.SetLiquid(pos, nil)
		}
	}
	b.changes[i].After = snapshotAt(tx, pos)
}

// SetLiquid records and writes a liquid layer. Passing nil removes liquid.
func (b *Batch) SetLiquid(tx *world.Tx, pos cube.Pos, liq world.Liquid) {
	i := b.ensure(tx, pos)
	tx.SetLiquid(pos, liq)
	b.changes[i].After = snapshotAt(tx, pos)
}

// SetBiome records and applies a biome change.
func (b *Batch) SetBiome(tx *world.Tx, pos cube.Pos, biome world.Biome) {
	i := b.ensure(tx, pos)
	tx.SetBiome(pos, biome)
	b.changes[i].After = snapshotAt(tx, pos)
}

// Len returns how many positions in the batch actually changed.
func (b *Batch) Len() int {
	if b == nil {
		return 0
	}
	n := 0
	for _, c := range b.changes {
		if !sameSnapshot(c.Before, c.After) {
			n++
		}
	}
	return n
}

func (b *Batch) compact() Batch {
	out := Batch{Brush: b.Brush}
	for _, c := range b.changes {
		if !sameSnapshot(c.Before, c.After) {
			out.changes = append(out.changes, c)
		}
	}
	return out
}

// BlockSnapshot is exported world state at a position (for brush displacement).
type BlockSnapshot struct {
	Block  world.Block
	Liquid world.Liquid
	HasLiq bool
	Biome  world.Biome
}

func snapToInternal(s BlockSnapshot) snapshot {
	return snapshot(s)
}

// SnapshotAtBlock captures current state for undo-aware brush operations.
func SnapshotAtBlock(tx *world.Tx, pos cube.Pos) BlockSnapshot {
	s := snapshotAt(tx, pos)
	return BlockSnapshot(s)
}

// ApplyBlockSnapshot writes a captured snapshot to the world.
func ApplyBlockSnapshot(tx *world.Tx, pos cube.Pos, s BlockSnapshot) {
	applySnapshot(tx, pos, snapToInternal(s))
}

// EnsurePos records before-state for pos if needed and returns the change index.
func (b *Batch) EnsurePos(tx *world.Tx, pos cube.Pos) int {
	return b.ensure(tx, pos)
}

// SetAfterForIndex refreshes the "after" snapshot for a change index (after world writes).
func (b *Batch) SetAfterForIndex(tx *world.Tx, i int, pos cube.Pos) {
	b.changes[i].After = snapshotAt(tx, pos)
}

func applySnapshot(tx *world.Tx, pos cube.Pos, s snapshot) {
	tx.SetBlock(pos, s.Block, fastBlockSetOpts)
	if s.HasLiq {
		tx.SetLiquid(pos, s.Liquid)
	} else if _, ok := tx.Liquid(pos); ok {
		tx.SetLiquid(pos, nil)
	}
	if s.Biome != nil {
		tx.SetBiome(pos, s.Biome)
	}
}

// Undo applies before snapshots.
func (b Batch) Undo(tx *world.Tx) {
	for i := len(b.changes) - 1; i >= 0; i-- {
		c := b.changes[i]
		applySnapshot(tx, c.Pos, c.Before)
	}
}

// Redo applies after snapshots.
func (b Batch) Redo(tx *world.Tx) {
	for _, c := range b.changes {
		applySnapshot(tx, c.Pos, c.After)
	}
}

// History holds undo/redo stacks for commands and brushes separately.
type History struct {
	limit int

	undo, redo           []Batch
	brushUndo, brushRedo []Batch
}

// NewHistory creates an undo manager with the given cap per stack.
func NewHistory(limit int) *History {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	return &History{limit: limit}
}

// Record stores a compacted batch; returns new stack depth for feedback.
func (h *History) Record(batch *Batch) int {
	if batch == nil || batch.Len() == 0 {
		return 0
	}
	b := batch.compact()
	if b.Brush {
		h.brushUndo = appendLimited(h.brushUndo, b, h.limit)
		h.brushRedo = nil
		return len(h.brushUndo)
	}
	h.undo = appendLimited(h.undo, b, h.limit)
	h.redo = nil
	return len(h.undo)
}

func appendLimited(stack []Batch, b Batch, limit int) []Batch {
	if len(stack) == limit {
		copy(stack, stack[1:])
		stack[len(stack)-1] = b
		return stack
	}
	return append(stack, b)
}

// Undo pops one batch; brush selects the brush-only stack.
func (h *History) Undo(tx *world.Tx, brush bool) bool {
	if brush {
		if len(h.brushUndo) == 0 {
			return false
		}
		i := len(h.brushUndo) - 1
		b := h.brushUndo[i]
		h.brushUndo = h.brushUndo[:i]
		b.Undo(tx)
		h.brushRedo = appendLimited(h.brushRedo, b, h.limit)
		return true
	}
	if len(h.undo) == 0 {
		return false
	}
	i := len(h.undo) - 1
	b := h.undo[i]
	h.undo = h.undo[:i]
	b.Undo(tx)
	h.redo = appendLimited(h.redo, b, h.limit)
	return true
}

// Redo restores one undone batch.
func (h *History) Redo(tx *world.Tx, brush bool) bool {
	if brush {
		if len(h.brushRedo) == 0 {
			return false
		}
		i := len(h.brushRedo) - 1
		b := h.brushRedo[i]
		h.brushRedo = h.brushRedo[:i]
		b.Redo(tx)
		h.brushUndo = appendLimited(h.brushUndo, b, h.limit)
		return true
	}
	if len(h.redo) == 0 {
		return false
	}
	i := len(h.redo) - 1
	b := h.redo[i]
	h.redo = h.redo[:i]
	b.Redo(tx)
	h.undo = appendLimited(h.undo, b, h.limit)
	return true
}
