package we

import (
	"testing"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestTraceBrushBlockSkipsNearStartCollision(t *testing.T) {
	withWorldTx(t, func(tx *world.Tx) {
		start := mgl64.Vec3{0.5, 1.5, 0.5}
		end := mgl64.Vec3{0.5, 1.5, 32.5}
		near := cube.Pos{0, 1, 0}
		far := cube.Pos{0, 1, 20}
		tx.SetBlock(near, mcblock.Stone{}, nil)
		tx.SetBlock(far, mcblock.Stone{}, nil)

		pos, face, ok := traceBrushBlock(start, end, tx, brushRaySelfSkipDistance)
		if !ok {
			t.Fatal("traceBrushBlock did not hit the far block")
		}
		if pos != far || face != cube.FaceNorth {
			t.Fatalf("traceBrushBlock hit %v/%v, want %v/%v", pos, face, far, cube.FaceNorth)
		}
	})
}

func TestTraceBrushBlockHitsDistantBlockPastClientReach(t *testing.T) {
	withWorldTx(t, func(tx *world.Tx) {
		start := mgl64.Vec3{0.5, 1.5, 0.5}
		end := mgl64.Vec3{0.5, 1.5, 32.5}
		far := cube.Pos{0, 1, 20}
		tx.SetBlock(far, mcblock.Stone{}, nil)

		pos, face, ok := traceBrushBlock(start, end, tx, brushRaySelfSkipDistance)
		if !ok {
			t.Fatal("traceBrushBlock did not hit the distant block")
		}
		if pos != far || face != cube.FaceNorth {
			t.Fatalf("traceBrushBlock hit %v/%v, want %v/%v", pos, face, far, cube.FaceNorth)
		}
	})
}

func withWorldTx(t *testing.T, f func(tx *world.Tx)) {
	t.Helper()
	w := world.New()
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("close world: %v", err)
		}
	}()
	<-w.Exec(f)
}
