package editbrush

import (
	"math"
	"math/rand"
	"sort"
	"strings"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

func applyPaint(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	strength := math.Max(0, math.Min(1, cfg.Strength))
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		if isSurface(tx, pos) && rand.Float64() <= strength {
			batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, nil))
		}
	})
}

func applyPushPull(tx *world.Tx, p *player.Player, target cube.Pos, cfg BrushConfig, pull bool, batch *history.Batch) {
	dir := dominantDir(target, cube.PosFromVec3(p.Position()))
	if !pull {
		dir = cube.Pos{-dir[0], -dir[1], -dir[2]}
	}
	var positions []cube.Pos
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		if !parse.IsAir(tx.Block(pos)) {
			positions = append(positions, pos)
		}
	})
	sortPositionsForMove(positions, dir)
	snap := make(map[cube.Pos]history.BlockSnapshot, len(positions))
	for _, pos := range positions {
		snap[pos] = history.SnapshotAtBlock(tx, pos)
	}
	for _, pos := range positions {
		batch.SetBlock(tx, pos, mcblock.Air{})
		batch.SetLiquid(tx, pos, nil)
	}
	for _, pos := range positions {
		dst := pos.Add(dir)
		i := batch.EnsurePos(tx, dst)
		history.ApplyBlockSnapshot(tx, dst, snap[pos])
		batch.SetAfterForIndex(tx, i, dst)
	}
}

func sortPositionsForMove(positions []cube.Pos, dir cube.Pos) {
	sort.Slice(positions, func(i, j int) bool {
		a, b := positions[i], positions[j]
		return a[0]*dir[0]+a[1]*dir[1]+a[2]*dir[2] > b[0]*dir[0]+b[1]*dir[1]+b[2]*dir[2]
	})
}

func dominantDir(from, to cube.Pos) cube.Pos {
	dx, dy, dz := to[0]-from[0], to[1]-from[1], to[2]-from[2]
	if absInt(dx) >= absInt(dy) && absInt(dx) >= absInt(dz) {
		if dx >= 0 {
			return cube.Pos{1, 0, 0}
		}
		return cube.Pos{-1, 0, 0}
	}
	if absInt(dy) >= absInt(dx) && absInt(dy) >= absInt(dz) {
		if dy >= 0 {
			return cube.Pos{0, 1, 0}
		}
		return cube.Pos{0, -1, 0}
	}
	if dz >= 0 {
		return cube.Pos{0, 0, 1}
	}
	return cube.Pos{0, 0, -1}
}

func applyTerraform(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	if strings.EqualFold(cfg.Mode, "expand") {
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if !parse.IsAir(tx.Block(pos)) {
				return
			}
			for _, f := range cube.Faces() {
				if !parse.IsAir(tx.Block(pos.Side(f))) {
					batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, nil))
					return
				}
			}
		})
		return
	}
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		if isSurface(tx, pos) {
			batch.SetBlock(tx, pos, mcblock.Air{})
		}
	})
}
