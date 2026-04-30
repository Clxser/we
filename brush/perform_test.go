package brush_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/brush"
	"github.com/df-mc/we/geo"
)

//go:linkname finaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func finaliseBlockRegistry()

type setBlockAction struct {
	b world.Block
}

func (a setBlockAction) At(_ int, _ int, _ int, _ *rand.Rand, _ *world.Tx, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	return a.b, nil
}

func (a setBlockAction) Form(brush.Shape) form.Form {
	return nil
}

func TestPerformRevertRestoresChangedBlocks(t *testing.T) {
	finaliseBlockRegistry()
	w := world.Config{RandomTickSpeed: -1, SaveInterval: -1}.New()
	defer w.Close()

	var failures []string
	<-w.Exec(func(tx *world.Tx) {
		pos := cube.Pos{1, 2, 3}
		before := block.Dirt{}
		after := block.Stone{}

		tx.SetBlock(pos, before, nil)
		revert := brush.Perform(pos, geo.Cube{R: 0}, setBlockAction{b: after}, tx)
		if got := tx.Block(pos); !sameBlock(got, after) {
			failures = append(failures, fmt.Sprintf("after Perform block = %v, want %v", blockState(got), blockState(after)))
		}

		revert(tx)
		if got := tx.Block(pos); !sameBlock(got, before) {
			failures = append(failures, fmt.Sprintf("after revert block = %v, want %v", blockState(got), blockState(before)))
		}
	})
	if len(failures) != 0 {
		t.Fatal(strings.Join(failures, "\n"))
	}
}

func sameBlock(a, b world.Block) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	an, ap := a.EncodeBlock()
	bn, bp := b.EncodeBlock()
	return an == bn && reflect.DeepEqual(ap, bp)
}

func blockState(b world.Block) string {
	if b == nil {
		return "<nil>"
	}
	name, props := b.EncodeBlock()
	if len(props) == 0 {
		return name
	}
	return fmt.Sprintf("%s%v", name, props)
}
