package service

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

func brushTopLayer(tx *world.Tx, target cube.Pos, cfg BrushConfig, mask edit.BlockMask, blocks []world.Block, batch *history.Batch) {
	spec := cfg.shapeSpec()
	area := spec.Bounds(target)
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			for y := area.Max[1]; y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				if !spec.Inside(target, pos) {
					continue
				}
				b := tx.Block(pos)
				if parse.IsAir(b) {
					continue
				}
				if mask.Match(b) {
					batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, nil))
				}
				break
			}
		}
	}
}

func brushOverlay(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	spec := cfg.shapeSpec()
	area := spec.Bounds(target)
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			for y := area.Max[1]; y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				if !spec.Inside(target, pos) {
					continue
				}
				if parse.IsAir(tx.Block(pos)) {
					continue
				}
				above := cube.Pos{x, y + 1, z}
				if spec.Inside(target, above) && parse.IsAir(tx.Block(above)) {
					batch.SetBlock(tx, above, edit.ChooseBlock(blocks, nil))
				}
				break
			}
		}
	}
}

func isSurface(tx *world.Tx, pos cube.Pos) bool {
	if parse.IsAir(tx.Block(pos)) {
		return false
	}
	for _, f := range cube.Faces() {
		if parse.IsAir(tx.Block(pos.Side(f))) {
			return true
		}
	}
	return false
}

func applyWrap(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	if cfg.ExtendWrap {
		base := tx.Block(target)
		seen := map[cube.Pos]bool{target: true}
		queue := []cube.Pos{target}
		limit := cfg.Radius * cfg.Radius
		for len(queue) > 0 {
			pos := queue[0]
			queue = queue[1:]
			wrapOne(tx, pos, blocks, batch)
			for _, f := range cube.Faces() {
				n := pos.Side(f)
				if seen[n] {
					continue
				}
				dx, dy, dz := n[0]-target[0], n[1]-target[1], n[2]-target[2]
				if dx*dx+dy*dy+dz*dz > limit || !parse.SameBlock(tx.Block(n), base) {
					continue
				}
				seen[n] = true
				queue = append(queue, n)
			}
		}
		return
	}
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		wrapOne(tx, pos, blocks, batch)
	})
}

func wrapOne(tx *world.Tx, pos cube.Pos, blocks []world.Block, batch *history.Batch) {
	if parse.IsAir(tx.Block(pos)) {
		return
	}
	for _, f := range cube.Faces() {
		n := pos.Side(f)
		if parse.IsAir(tx.Block(n)) {
			batch.SetBlock(tx, n, edit.ChooseBlock(blocks, nil))
		}
	}
}
