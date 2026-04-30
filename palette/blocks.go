package palette

import (
	"encoding/json"
	"fmt"

	"github.com/df-mc/dragonfly/server/world"
	"io"
)

// Blocks is a Palette that exists out of a slice of world.Block. It is a static palette in the sense that the
// blocks returned in the Blocks method do not change.
type Blocks struct {
	b []world.Block
}

// NewBlocks creates a Blocks palette that returns the blocks passed in the Blocks method.
func NewBlocks(b []world.Block) Blocks {
	return Blocks{b: b}
}

// Read reads a Blocks palette from an io.Reader.
func Read(r io.Reader) (Blocks, error) {
	var states []blockState
	if err := json.NewDecoder(r).Decode(&states); err != nil {
		return Blocks{}, err
	}
	blocks := make([]world.Block, 0, len(states))
	for _, state := range states {
		b, err := blockFromState(state)
		if err != nil {
			return Blocks{}, err
		}
		blocks = append(blocks, b)
	}
	return Blocks{b: blocks}, nil
}

// Write writes a Blocks palette to an io.Writer.
func (b Blocks) Write(w io.Writer) error {
	states := make([]blockState, 0, len(b.b))
	for _, bl := range b.b {
		states = append(states, stateOfBlock(bl))
	}
	return json.NewEncoder(w).Encode(states)
}

// Blocks returns all world.Block passed to the NewBlocks function upon creation of the palette.
func (b Blocks) Blocks(_ *world.Tx) []world.Block {
	return b.b
}

type blockState struct {
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties,omitempty"`
}

func stateOfBlock(b world.Block) blockState {
	if b == nil {
		if air, ok := world.BlockByName("minecraft:air", nil); ok {
			b = air
		}
	}
	name, props := b.EncodeBlock()
	if len(props) == 0 {
		props = nil
	}
	return blockState{Name: name, Properties: props}
}

func blockFromState(s blockState) (world.Block, error) {
	props := normaliseProps(s.Properties)
	if b, ok := world.BlockByName(s.Name, props); ok {
		return b, nil
	}
	return nil, fmt.Errorf("unknown block state %s", s.Name)
}

func normaliseProps(props map[string]any) map[string]any {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]any, len(props))
	for k, v := range props {
		if n, ok := v.(float64); ok && n == float64(int32(n)) {
			out[k] = int32(n)
			continue
		}
		out[k] = v
	}
	return out
}
