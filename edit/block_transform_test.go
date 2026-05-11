package edit

import (
	"fmt"
	"testing"

	"github.com/Velvet-MC/s2d/translate"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestRotateBlockDoesNotPanicForRegisteredBlocks(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	for _, b := range world.Blocks() {
		t.Run(fmt.Sprintf("%T", b), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("RotateBlock panicked: %v", r)
				}
			}()
			_ = RotateBlock(b, "y", 3)
		})
	}
}

func TestRotateBlockKeepsInertStateBlocksInert(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	got := translate.Lookup("minecraft:smoker[facing=north,lit=false]")
	if !got.Recognized {
		t.Fatal("smoker was not recognized")
	}
	rotated := RotateBlock(got.Block, "y", 1)
	if _, ticking := rotated.(interface {
		Tick(int64, cube.Pos, *world.Tx)
	}); ticking {
		t.Fatalf("rotated schematic block %T must remain inert", rotated)
	}
}
