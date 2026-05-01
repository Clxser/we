package editbrush

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
)

// ApplyBrush dispatches a brush use to the implementation for cfg.Type.
// target is the raycast hit position. Errors propagate from block parsing or schematic IO.
func ApplyBrush(tx *world.Tx, p *player.Player, target cube.Pos, cfg BrushConfig, batch *history.Batch) error {
	blocks, err := cfg.blockList()
	if err != nil {
		return err
	}
	switch strings.ToLower(cfg.Type) {
	case "sphere", "cylinder", "pyramid", "cone", "cube":
		edit.ApplyShape(tx, target, cfg.shapeSpec(), blocks, batch)
	case "fill":
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if pos[1] <= target[1] {
				batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, nil))
			}
		})
	case "toplayer":
		brushTopLayer(tx, target, cfg, edit.BlockMask{All: true}, blocks, batch)
	case "overlay":
		brushOverlay(tx, target, cfg, blocks, batch)
	case "wrap":
		applyWrap(tx, target, cfg, blocks, batch)
	case "paint":
		applyPaint(tx, target, cfg, blocks, batch)
	case "pull", "push":
		applyPushPull(tx, p, target, cfg, strings.EqualFold(cfg.Type, "pull"), batch)
	case "terraform":
		applyTerraform(tx, target, cfg, blocks, batch)
	case "schematic":
		return applySchematicBrush(tx, target, p.Rotation().Direction(), cfg, batch)
	case "replace":
		from, err := cfg.fromList()
		if err != nil {
			return err
		}
		mask := edit.BlockMask{All: cfg.All, IncludeAir: cfg.ReplaceAir, Blocks: from}
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if mask.Match(tx.Block(pos)) {
				batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, nil))
			}
		})
	case "line":
		applyLineBrush(tx, p, cfg, blocks, batch)
	default:
		return fmt.Errorf("unknown brush type %q", cfg.Type)
	}
	return nil
}

func applyBrushShape(tx *world.Tx, target cube.Pos, cfg BrushConfig, f func(pos cube.Pos)) {
	spec := cfg.shapeSpec()
	area := spec.Bounds(target)
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		if spec.Hollow {
			if !spec.Shell(target, pos) {
				return
			}
		} else if !spec.Inside(target, pos) {
			return
		}
		f(pos)
	})
	_ = tx
}
