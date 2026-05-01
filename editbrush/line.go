package editbrush

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

func applyLineBrush(tx *world.Tx, p *player.Player, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	start := cube.PosFromVec3(p.Position().Add(p.Rotation().Vec3()))
	step := p.Rotation().Vec3()
	last := start
	for i := 0; i < max(1, cfg.Range); i++ {
		pos := cube.PosFromVec3(p.Position().Add(step.Mul(float64(i + 1))))
		if !cfg.PassThrough && !parse.IsAir(tx.Block(pos)) && i > 0 {
			break
		}
		last = pos
	}
	edit.Line(tx, start, last, max(1, cfg.Thickness), blocks, batch)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
