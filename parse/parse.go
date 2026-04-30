package parse

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// BlockState is a serialisable block identity for JSON (brush config, schematics).
type BlockState struct {
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties,omitempty"`
}

// StateOfBlock encodes a block for storage.
func StateOfBlock(b world.Block) BlockState {
	if b == nil {
		b = mcblock.Air{}
	}
	name, props := b.EncodeBlock()
	return BlockState{Name: name, Properties: cloneProps(props)}
}

// BlockFromState decodes a stored block state.
func BlockFromState(s BlockState) (world.Block, error) {
	props := NormaliseProps(s.Properties)
	if b, ok := world.BlockByName(s.Name, props); ok {
		return b, nil
	}
	return nil, fmt.Errorf("unknown block state %s", s.Name)
}

func cloneProps(props map[string]any) map[string]any {
	if len(props) == 0 {
		return nil
	}
	cp := make(map[string]any, len(props))
	for k, v := range props {
		cp[k] = v
	}
	return cp
}

// NormaliseProps coerces JSON-decoded numbers to int32 where appropriate.
func NormaliseProps(props map[string]any) map[string]any {
	if len(props) == 0 {
		return nil
	}
	cp := make(map[string]any, len(props))
	for k, v := range props {
		switch n := v.(type) {
		case float64:
			if n == float64(int32(n)) {
				cp[k] = int32(n)
			} else {
				cp[k] = n
			}
		default:
			cp[k] = v
		}
	}
	return cp
}

// ParseBlockList parses a comma or whitespace separated block list.
func ParseBlockList(input string) ([]world.Block, error) {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
	blocks := make([]world.Block, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		b, err := ParseBlock(part)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no block types specified")
	}
	return blocks, nil
}

// ParseBlock parses a single block name.
func ParseBlock(name string) (world.Block, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return nil, fmt.Errorf("empty block name")
	}
	switch name {
	case "air", "minecraft:air":
		return mcblock.Air{}, nil
	case "water", "minecraft:water":
		return mcblock.Water{Still: true, Depth: 8}, nil
	case "lava", "minecraft:lava":
		return mcblock.Lava{Still: true, Depth: 8}, nil
	}
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	if b, ok := world.BlockByName(name, nil); ok {
		return b, nil
	}
	for _, b := range world.Blocks() {
		n, _ := b.EncodeBlock()
		if n == name {
			return b, nil
		}
	}
	return nil, fmt.Errorf("unknown block type %q", name)
}

// SameBlock compares block identities (name + properties).
func SameBlock(a, b world.Block) bool {
	if a == nil {
		a = mcblock.Air{}
	}
	if b == nil {
		b = mcblock.Air{}
	}
	as, bs := StateOfBlock(a), StateOfBlock(b)
	return as.Name == bs.Name && reflect.DeepEqual(as.Properties, bs.Properties)
}

// SameLiquid compares liquid layers using block identity.
func SameLiquid(a world.Liquid, aOK bool, b world.Liquid, bOK bool) bool {
	if aOK != bOK {
		return false
	}
	if !aOK {
		return true
	}
	return SameBlock(a, b)
}

// SameBiome reports whether two biomes are equivalent.
func SameBiome(a, b world.Biome) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.EncodeBiome() == b.EncodeBiome() && a.String() == b.String()
}

// IsAir reports whether b is air or nil.
func IsAir(b world.Block) bool {
	_, ok := b.(mcblock.Air)
	return ok || b == nil
}

// IsFluidBlock reports whether b is water or lava block types.
func IsFluidBlock(b world.Block) bool {
	switch b.(type) {
	case mcblock.Water, mcblock.Lava:
		return true
	default:
		return false
	}
}

// SerialBlock is JSON for schematic disk format.
type SerialBlock struct {
	Set   bool       `json:"set"`
	State BlockState `json:"state"`
}

// MarshalBlock encodes a block for schematic JSON.
func MarshalBlock(b world.Block, set bool) SerialBlock {
	if !set {
		return SerialBlock{}
	}
	return SerialBlock{Set: true, State: StateOfBlock(b)}
}

// UnmarshalBlock decodes schematic JSON into a block.
func UnmarshalBlock(sb SerialBlock) (world.Block, bool, error) {
	if !sb.Set {
		return nil, false, nil
	}
	b, err := BlockFromState(sb.State)
	return b, true, err
}

// JSONRoundTripProps normalises property maps through JSON.
func JSONRoundTripProps(props map[string]any) map[string]any {
	if len(props) == 0 {
		return nil
	}
	b, _ := json.Marshal(props)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return NormaliseProps(out)
}
