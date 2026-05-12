package parse

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Velvet-MC/s2d/translate"
	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func init() {
	world.DefaultBlockRegistry.Finalize()
}

func TestParseBlockExactBedrockState(t *testing.T) {
	got, err := ParseBlock("minecraft:blue_glazed_terracotta[facing_direction=5]")
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	want := mcblock.GlazedTerracotta{Colour: item.ColourBlue(), Facing: cube.East}
	if !SameBlock(got, want) {
		name, props := got.EncodeBlock()
		t.Fatalf("block = %s %#v, want east-facing blue glazed terracotta", name, props)
	}
}

func TestParseBlockListKeepsCommasInsideStates(t *testing.T) {
	got, err := ParseBlockList("stone,minecraft:blue_glazed_terracotta[facing_direction=5]")
	if err != nil {
		t.Fatalf("ParseBlockList: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !SameBlock(got[0], mcblock.Stone{}) {
		t.Fatalf("first block = %#v, want stone", got[0])
	}
	if !SameBlock(got[1], mcblock.GlazedTerracotta{Colour: item.ColourBlue(), Facing: cube.East}) {
		name, props := got[1].EncodeBlock()
		t.Fatalf("second block = %s %#v, want east-facing blue glazed terracotta", name, props)
	}
}

func TestParseBlockReturnsStateBlockForKnownUnimplementedState(t *testing.T) {
	name, props, ok := firstUnimplementedState()
	if !ok {
		t.Skip("no unimplemented Bedrock states in this Dragonfly build")
	}
	got, err := ParseBlock(formatState(name, props))
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	if _, ok := got.(translate.StateBlock); !ok {
		t.Fatalf("block type = %T, want inert StateBlock for unimplemented state", got)
	}
	gotName, gotProps := got.EncodeBlock()
	if gotName != name || !reflect.DeepEqual(gotProps, props) {
		t.Fatalf("state = %s %#v, want %s %#v", gotName, gotProps, name, props)
	}
}

func firstUnimplementedState() (string, map[string]any, bool) {
	for _, b := range world.Blocks() {
		name, props := b.EncodeBlock()
		if name == "minecraft:air" || world.BlockImplemented(name, props) {
			continue
		}
		return name, cloneProps(props), true
	}
	return "", nil, false
}

func formatState(name string, props map[string]any) string {
	if len(props) == 0 {
		return name
	}
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, props[key]))
	}
	return name + "[" + strings.Join(parts, ",") + "]"
}
