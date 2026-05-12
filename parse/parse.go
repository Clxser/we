package parse

import (
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/Velvet-MC/s2d/translate"
	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// BlockState is a serialisable block identity for JSON (brush config, schematics).
type BlockState struct {
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties,omitempty"`
}

// BlockKey is a comparable block identity for hot-path equality checks.
type BlockKey struct {
	Base, State uint64
}

// StateOfBlock encodes a block for storage.
func StateOfBlock(b world.Block) BlockState {
	if b == nil {
		b = mcblock.Air{}
	}
	name, props := b.EncodeBlock()
	return BlockState{Name: name, Properties: cloneProps(props)}
}

// BlockKeyOf returns a comparable identity for b without cloning properties.
func BlockKeyOf(b world.Block) BlockKey {
	if b == nil {
		b = mcblock.Air{}
	}
	base, state := b.Hash()
	return BlockKey{Base: base, State: state}
}

// BlockFromState decodes a stored block state.
func BlockFromState(s BlockState) (world.Block, error) {
	props := NormaliseProps(s.Properties)
	if b, ok := blockByState(s.Name, props); ok {
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
	parts, err := splitBlockList(input)
	if err != nil {
		return nil, err
	}
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
	blockName, props, exactState, err := parseBlockState(name)
	if err != nil {
		return nil, err
	}
	if exactState {
		block, ok := blockByState(blockName, props)
		if !ok {
			return nil, fmt.Errorf("unknown block state %q", name)
		}
		return block, nil
	}
	name = blockName
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
	if b, ok := blockByState(name, nil); ok {
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

func splitBlockList(input string) ([]string, error) {
	var parts []string
	start, depth := 0, 0
	for i, r := range input {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unexpected ] in block list")
			}
		case ',', ';', ' ', '\t', '\n':
			if depth == 0 {
				if part := strings.TrimSpace(input[start:i]); part != "" {
					parts = append(parts, part)
				}
				start = i + len(string(r))
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unterminated block state in block list")
	}
	if part := strings.TrimSpace(input[start:]); part != "" {
		parts = append(parts, part)
	}
	return parts, nil
}

func parseBlockState(input string) (name string, props map[string]any, exact bool, err error) {
	open := strings.IndexByte(input, '[')
	if open < 0 {
		return normaliseBlockName(input), nil, false, nil
	}
	if !strings.HasSuffix(input, "]") {
		return "", nil, false, fmt.Errorf("malformed block state %q", input)
	}
	name = normaliseBlockName(strings.TrimSpace(input[:open]))
	body := strings.TrimSpace(input[open+1 : len(input)-1])
	props = map[string]any{}
	if body == "" {
		return name, props, true, nil
	}
	for _, part := range strings.Split(body, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return "", nil, false, fmt.Errorf("malformed block state property %q", part)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return "", nil, false, fmt.Errorf("malformed block state property %q", part)
		}
		props[key] = parseBlockStateValue(value)
	}
	return name, props, true, nil
}

func normaliseBlockName(name string) string {
	name = strings.TrimSpace(name)
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	return name
}

func parseBlockStateValue(value string) any {
	value = strings.Trim(value, `"'`)
	switch value {
	case "true":
		return true
	case "false":
		return false
	}
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		return int32(n)
	}
	return value
}

func blockByState(name string, props map[string]any) (world.Block, bool) {
	props = NormaliseProps(props)
	if b, ok := world.BlockByName(name, props); ok {
		return concreteOrStateBlock(name, props, b), true
	}
	for _, b := range world.Blocks() {
		candidateName, candidateProps := b.EncodeBlock()
		if candidateName != name || !propertiesEqual(candidateProps, props) {
			continue
		}
		return concreteOrStateBlock(candidateName, candidateProps, b), true
	}
	return nil, false
}

func concreteOrStateBlock(name string, props map[string]any, b world.Block) world.Block {
	if world.BlockImplemented(name, props) {
		return b
	}
	return translate.NewStateBlock(translate.BedrockState{Name: name, Properties: maps.Clone(props)})
}

func propertiesEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		bv, ok := b[key]
		if !ok || !propertyValueEqual(av, bv) {
			return false
		}
	}
	return true
}

func propertyValueEqual(a, b any) bool {
	switch av := a.(type) {
	case uint8:
		switch bv := b.(type) {
		case uint8:
			return av == bv
		case int32:
			return int32(av) == bv
		case int:
			return int(av) == bv
		}
	case int32:
		switch bv := b.(type) {
		case uint8:
			return av == int32(bv)
		case int32:
			return av == bv
		case int:
			return int(av) == bv
		}
	case int:
		switch bv := b.(type) {
		case uint8:
			return av == int(bv)
		case int32:
			return av == int(bv)
		case int:
			return av == bv
		}
	}
	return a == b
}

// SameBlock compares block identities (name + properties).
func SameBlock(a, b world.Block) bool {
	return BlockKeyOf(a) == BlockKeyOf(b)
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
