package editbrush_test

import (
	"strings"
	"testing"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/editbrush"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
)

//go:linkname finaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func finaliseBlockRegistry()

func withTx(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	finaliseBlockRegistry()
	w := world.New()
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("close world: %v", err)
		}
	}()
	<-w.Exec(f)
}

func TestApplyBrushWithSettingsRejectsLargeBrush(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		batch := history.NewBatch(true)
		err := editbrush.ApplyBrushWithSettings(tx, nil, cube.Pos{0, 0, 0}, editbrush.BrushConfig{
			Type:   "cube",
			Length: 2,
			Width:  1,
			Height: 1,
		}, edit.DefaultSchematicStore(), guardrail.Limits{MaxBrushVolume: 1}, batch)
		if err == nil || !strings.Contains(err.Error(), "brush volume 2 exceeds limit 1") {
			t.Fatalf("ApplyBrushWithSettings error = %v, want brush limit error", err)
		}
		if batch.Len() != 0 {
			t.Fatalf("batch Len = %d, want 0", batch.Len())
		}
	})
}
