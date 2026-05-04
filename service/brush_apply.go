package service

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
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
	if bounds, ok := BrushVolumeBounds(target, cfg); ok {
		if err := limits.CheckBrushVolume(bounds.Volume()); err != nil {
			return err
		}
		if err := limits.CheckEditSubChunks(bounds.SubChunkCount()); err != nil {
			return err
		}
	}
	switch brushType {
	case BrushSphere, BrushCylinder, BrushPyramid, BrushCone, BrushCube:
		edit.ApplyShape(tx, target, cfg.shapeSpec(), blocks, batch)
	case BrushFill:
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if pos[1] <= target[1] {
				batch.SetBlockFast(tx, pos, edit.ChooseBlock(blocks, nil))
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
		mask := edit.BlockMask{All: cfg.All, IncludeAir: cfg.ReplaceAir, Blocks: from}.Prepared()
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if mask.Match(tx.Block(pos)) {
				batch.SetBlockFast(tx, pos, edit.ChooseBlock(blocks, nil))
			}
		})
	case BrushLine:
		applyLineBrush(tx, actor, cfg, blocks, batch)
	default:
		return fmt.Errorf("unknown brush type %q", cfg.Type)
	}
	return nil
}

// ApplyBrushAndRecord applies a brush and records the resulting brush history
// batch on s. It keeps brush undo/redo bookkeeping out of Dragonfly adapters.
func ApplyBrushAndRecord(tx *world.Tx, s Session, actor BrushActor, target cube.Pos, cfg BrushConfig, store edit.SchematicStore, limits guardrail.Limits) error {
	batch := history.NewBatch(true)
	if err := ApplyBrush(tx, actor, target, cfg, store, limits, batch); err != nil {
		return err
	}
	s.Record(batch)
	return nil
}

// BrushVolumeBounds returns the inclusive area used for brush volume checks and
// visual previews. Brushes without a shape volume return ok=false.
func BrushVolumeBounds(target cube.Pos, cfg BrushConfig) (area geo.Area, ok bool) {
	if !brushTypeUsesShapeVolume(strings.ToLower(cfg.Type)) {
		return geo.Area{}, false
	}
	return cfg.shapeSpec().Bounds(target), true
}

// BrushAnchorFromSurface returns the brush anchor for an aimed surface. Shape
// volume brushes are moved so their nearest edge starts at surface and the
// generated volume extends out through face instead of being centred inside the
// clicked block/player. Non-volume brushes use surface directly.
func BrushAnchorFromSurface(surface cube.Pos, face cube.Face, cfg BrushConfig) cube.Pos {
	if brushAnchorsOnHitBlock(strings.ToLower(cfg.Type)) {
		return surface.Side(face.Opposite())
	}
	if !brushTypeUsesShapeVolume(strings.ToLower(cfg.Type)) {
		return surface
	}
	spec := cfg.shapeSpec()
	bounds := spec.Bounds(cube.Pos{})
	anchor := surface
	switch face {
	case cube.FaceUp:
		anchor[1] -= bounds.Min[1]
	case cube.FaceDown:
		anchor[1] -= bounds.Max[1]
	case cube.FaceEast:
		anchor[0] -= bounds.Min[0]
	case cube.FaceWest:
		anchor[0] -= bounds.Max[0]
	case cube.FaceSouth:
		anchor[2] -= bounds.Min[2]
	case cube.FaceNorth:
		anchor[2] -= bounds.Max[2]
	}
	return anchor
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
