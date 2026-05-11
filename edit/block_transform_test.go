package edit

import (
	"fmt"
	"testing"

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
