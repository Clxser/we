package service_test

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/service"
)

func TestApplyBrushRejectsLargeBrush(t *testing.T) {
	withTx(t, func(tx *world.Tx) {
		batch := history.NewBatch(true)
		err := service.ApplyBrush(tx, service.BrushActor{}, cube.Pos{0, 0, 0}, service.BrushConfig{
			Type:   "cube",
			Length: 2,
			Width:  1,
			Height: 1,
		}, edit.DefaultSchematicStore(), guardrail.Limits{MaxBrushVolume: 1}, batch)
		if err == nil || !strings.Contains(err.Error(), "brush volume 2 exceeds limit 1") {
			t.Fatalf("ApplyBrush error = %v, want brush limit error", err)
		}
		if batch.Len() != 0 {
			t.Fatalf("batch Len = %d, want 0", batch.Len())
		}
	})
}
