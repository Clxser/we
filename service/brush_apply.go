package service

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
	"github.com/go-gl/mathgl/mgl64"
)

// BrushActor describes the player state needed by brush service operations.
type BrushActor struct {
	Position mgl64.Vec3
	Rotation cube.Rotation
}

// ApplyBrush dispatches a brush use with a schematic store and optional safety
// limits. Zero-valued limits are unlimited.
func ApplyBrush(tx *world.Tx, actor BrushActor, target cube.Pos, cfg BrushConfig, store edit.SchematicStore, limits guardrail.Limits, batch *history.Batch) error {
	blocks, err := cfg.blockList()
	if err != nil {
		return err
	}
	brushType := strings.ToLower(cfg.Type)
	if brushTypeUsesShapeVolume(brushType) {
		if err := limits.CheckBrushVolume(cfg.shapeSpec().Bounds(target).Volume()); err != nil {
			return err
		}
	}
	switch brushType {
	case BrushSphere, BrushCylinder, BrushPyramid, BrushCone, BrushCube:
		edit.ApplyShape(tx, target, cfg.shapeSpec(), blocks, batch)
	case BrushFill:
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if pos[1] <= target[1] {
				batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, nil))
			}
		})
	case BrushTopLayer:
		brushTopLayer(tx, target, cfg, edit.BlockMask{All: true}, blocks, batch)
	case BrushOverlay:
		brushOverlay(tx, target, cfg, blocks, batch)
	case BrushWrap:
		applyWrap(tx, target, cfg, blocks, batch)
	case BrushPaint:
		applyPaint(tx, target, cfg, blocks, batch)
	case BrushPull, BrushPush:
		applyPushPull(tx, actor, target, cfg, strings.EqualFold(cfg.Type, "pull"), batch)
	case BrushTerraform:
		applyTerraform(tx, target, cfg, blocks, batch)
	case BrushSchematic:
		return applySchematicBrush(tx, target, actor.Rotation.Direction(), cfg, store, limits, batch)
	case BrushReplace:
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
	case BrushLine:
		applyLineBrush(tx, actor, cfg, blocks, batch)
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
