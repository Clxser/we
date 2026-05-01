package editbrush

import (
	"fmt"
	"math/rand"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
)

func applySchematicBrush(tx *world.Tx, target cube.Pos, dir cube.Direction, cfg BrushConfig, store edit.SchematicStore, limits guardrail.Limits, batch *history.Batch) error {
	if store == nil {
		store = edit.DefaultSchematicStore()
	}
	if len(cfg.Schematics) == 0 {
		return fmt.Errorf("schematic brush has no schematics selected")
	}
	name := cfg.Schematics[0]
	if cfg.RandomSchematic {
		name = cfg.Schematics[rand.Intn(len(cfg.Schematics))]
	}
	cb, err := store.Load(name)
	if err != nil {
		return err
	}
	if err := limits.CheckBrushVolume(int64(len(cb.Entries))); err != nil {
		return err
	}
	if cfg.RandomRotation {
		dirs := []cube.Direction{cube.North, cube.East, cube.South, cube.West}
		dir = dirs[rand.Intn(len(dirs))]
	}
	return edit.PasteClipboard(tx, cb, target, dir, cfg.NoAir, batch)
}
