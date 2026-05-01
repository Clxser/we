package palette

import (
	"encoding/json"
	"io"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
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
	var states []parse.BlockState
	if err := json.NewDecoder(r).Decode(&states); err != nil {
		return Blocks{}, err
	}
	blocks := make([]world.Block, 0, len(states))
	for _, state := range states {
		b, err := parse.BlockFromState(state)
		if err != nil {
			return Blocks{}, err
		}
		blocks = append(blocks, b)
	}
	return Blocks{b: blocks}, nil
}

// Write writes a Blocks palette to an io.Writer.
func (b Blocks) Write(w io.Writer) error {
	states := make([]parse.BlockState, 0, len(b.b))
	for _, bl := range b.b {
		states = append(states, parse.StateOfBlock(bl))
	}
	return json.NewEncoder(w).Encode(states)
}

// Blocks returns all world.Block passed to the NewBlocks function upon creation of the palette.
func (b Blocks) Blocks(_ *world.Tx) []world.Block {
	return b.b
}
