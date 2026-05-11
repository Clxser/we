package edit

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/parse"
)

type compactBlockState struct {
	Block  world.Block
	Liquid world.Liquid
	HasLiq bool
}

type compactBlockKey struct {
	block  parse.BlockKey
	liquid parse.BlockKey
	hasLiq bool
}

// CompactSchematic stores a dense schematic as palette indexes instead of one
// full block entry per cell. It is intended for large, no-undo Java schematic
// pastes where the caller does not need a reusable clipboard.
type CompactSchematic struct {
	dims    [3]int
	palette []compactBlockState
	lookup  map[compactBlockKey]uint32
	small   []uint16
	wide    []uint32
}

// NewCompactSchematic creates a dense compact schematic with the given
// dimensions.
func NewCompactSchematic(width, height, length int) (*CompactSchematic, error) {
	if width <= 0 || height <= 0 || length <= 0 {
		return nil, fmt.Errorf("compact schematic dimensions must be positive")
	}
	volume := width * height * length
	return &CompactSchematic{
		dims:   [3]int{width, height, length},
		lookup: make(map[compactBlockKey]uint32),
		small:  make([]uint16, volume),
	}, nil
}

// AddBlock stores block/liquid at pos within the compact schematic.
func (s *CompactSchematic) AddBlock(pos cube.Pos, block world.Block, liquid world.Liquid) error {
	if s == nil {
		return fmt.Errorf("compact schematic is nil")
	}
	if pos[0] < 0 || pos[0] >= s.dims[0] || pos[1] < 0 || pos[1] >= s.dims[1] || pos[2] < 0 || pos[2] >= s.dims[2] {
		return fmt.Errorf("compact schematic position %v out of bounds", pos)
	}
	state := compactBlockState{Block: block}
	if liquid != nil {
		state.Liquid = liquid
		state.HasLiq = true
	}
	paletteIndex := s.paletteIndex(state)
	i := denseIndex(pos, cube.Pos{}, s.dims)
	s.setIndex(i, paletteIndex)
	return nil
}

// AppendPaletteState appends block/liquid as a palette entry without
// deduplicating it. This is used when the source format already has a palette
// and callers can map source palette indexes once.
func (s *CompactSchematic) AppendPaletteState(block world.Block, liquid world.Liquid) uint32 {
	state := compactBlockState{Block: block}
	if liquid != nil {
		state.Liquid = liquid
		state.HasLiq = true
	}
	i := uint32(len(s.palette))
	s.palette = append(s.palette, state)
	if i > uint32(^uint16(0)) && s.wide == nil {
		s.wide = make([]uint32, len(s.small))
		for n, v := range s.small {
			s.wide[n] = uint32(v)
		}
		s.small = nil
	}
	return i
}

// SetPaletteIndex stores a previously appended palette index at pos.
func (s *CompactSchematic) SetPaletteIndex(pos cube.Pos, paletteIndex uint32) error {
	if s == nil {
		return fmt.Errorf("compact schematic is nil")
	}
	if pos[0] < 0 || pos[0] >= s.dims[0] || pos[1] < 0 || pos[1] >= s.dims[1] || pos[2] < 0 || pos[2] >= s.dims[2] {
		return fmt.Errorf("compact schematic position %v out of bounds", pos)
	}
	if int(paletteIndex) >= len(s.palette) {
		return fmt.Errorf("compact schematic palette index %d out of range", paletteIndex)
	}
	s.setPaletteIndexXYZ(pos[0], pos[1], pos[2], paletteIndex)
	return nil
}

func (s *CompactSchematic) setPaletteIndexXYZ(x, y, z int, paletteIndex uint32) {
	s.setIndex((x*s.dims[1]+y)*s.dims[2]+z, paletteIndex)
}

// Volume returns the number of cells in the compact schematic.
func (s *CompactSchematic) Volume() int {
	if s == nil {
		return 0
	}
	return s.dims[0] * s.dims[1] * s.dims[2]
}

// PaletteLen returns the number of unique block/liquid states in the compact
// schematic.
func (s *CompactSchematic) PaletteLen() int {
	if s == nil {
		return 0
	}
	return len(s.palette)
}

func (s *CompactSchematic) paletteIndex(state compactBlockState) uint32 {
	key := compactKey(state)
	if i, ok := s.lookup[key]; ok {
		return i
	}
	i := uint32(len(s.palette))
	s.lookup[key] = i
	s.palette = append(s.palette, state)
	if i > uint32(^uint16(0)) && s.wide == nil {
		s.wide = make([]uint32, len(s.small))
		for n, v := range s.small {
			s.wide[n] = uint32(v)
		}
		s.small = nil
	}
	return i
}

func compactKey(state compactBlockState) compactBlockKey {
	key := compactBlockKey{block: parse.BlockKeyOf(state.Block), hasLiq: state.HasLiq}
	if state.HasLiq {
		key.liquid = parse.BlockKeyOf(state.Liquid)
	}
	return key
}

func (s *CompactSchematic) setIndex(i int, paletteIndex uint32) {
	if s.wide != nil {
		s.wide[i] = paletteIndex
		return
	}
	s.small[i] = uint16(paletteIndex)
}

func (s *CompactSchematic) indexAt(i int) uint32 {
	if s.wide != nil {
		return s.wide[i]
	}
	return uint32(s.small[i])
}

// PasteCompactSchematicNoUndo writes s at origin using Dragonfly's dense
// BuildStructure path. The schematic is treated as copied facing north and is
// rotated to dir around the Y axis.
func PasteCompactSchematicNoUndo(tx *world.Tx, s *CompactSchematic, origin cube.Pos, dir cube.Direction) error {
	if tx == nil {
		return fmt.Errorf("world transaction is nil")
	}
	if s == nil || s.Volume() == 0 || len(s.palette) == 0 {
		return fmt.Errorf("compact schematic is empty")
	}
	turns := rotationTurns(cube.North, dir)
	palette := transformedCompactPalette(s.palette, turns)
	min := cube.Pos{}
	dims := s.dims
	if turns != 0 {
		max := cube.Pos{s.dims[0] - 1, s.dims[1] - 1, s.dims[2] - 1}
		rotMin, rotMax := rotatedBounds(cube.Pos{}, max, turns)
		min = rotMin
		dims = [3]int{rotMax[0] - rotMin[0] + 1, rotMax[1] - rotMin[1] + 1, rotMax[2] - rotMin[2] + 1}
	}
	buildStructure(tx, origin.Add(min), compactSchematicStructure{
		schematic: s,
		palette:   palette,
		min:       min,
		dims:      dims,
		turns:     turns,
	})
	return nil
}

// CompactPasteSubChunkCount returns how many sub-chunks a dense compact paste
// touches after rotation.
func CompactPasteSubChunkCount(s *CompactSchematic, origin cube.Pos, dir cube.Direction) int64 {
	if s == nil || s.Volume() == 0 {
		return 0
	}
	turns := rotationTurns(cube.North, dir)
	min := cube.Pos{}
	dims := s.dims
	if turns != 0 {
		max := cube.Pos{s.dims[0] - 1, s.dims[1] - 1, s.dims[2] - 1}
		rotMin, rotMax := rotatedBounds(cube.Pos{}, max, turns)
		min = rotMin
		dims = [3]int{rotMax[0] - rotMin[0] + 1, rotMax[1] - rotMin[1] + 1, rotMax[2] - rotMin[2] + 1}
	}
	pasteMin := origin.Add(min)
	pasteMax := pasteMin.Add(cube.Pos{dims[0] - 1, dims[1] - 1, dims[2] - 1})
	return geo.Area{Min: pasteMin, Max: pasteMax}.SubChunkCount()
}

func transformedCompactPalette(palette []compactBlockState, turns int) []compactBlockState {
	if turns == 0 {
		return palette
	}
	transform := blockTransform{axis: "y", turns: turns}
	cache := make(blockTransformCache)
	out := make([]compactBlockState, len(palette))
	for i, state := range palette {
		out[i] = compactBlockState{
			Block:  cache.transform(state.Block, transform),
			Liquid: state.Liquid,
			HasLiq: state.HasLiq,
		}
		if state.HasLiq {
			if b, ok := cache.transform(state.Liquid, transform).(world.Liquid); ok {
				out[i].Liquid = b
			}
		}
	}
	return out
}

type compactSchematicStructure struct {
	schematic *CompactSchematic
	palette   []compactBlockState
	min       cube.Pos
	dims      [3]int
	turns     int
}

func (s compactSchematicStructure) Dimensions() [3]int { return s.dims }

func (s compactSchematicStructure) At(x, y, z int, _ func(x, y, z int) world.Block) (world.Block, world.Liquid) {
	pos := cube.Pos{x, y, z}
	if s.turns != 0 {
		pos = rotateOffset(pos.Add(s.min), "y", (4-s.turns)%4)
	}
	index := denseIndex(pos, cube.Pos{}, s.schematic.dims)
	state := s.palette[s.schematic.indexAt(index)]
	return compactStateLayers(state)
}

func compactStateLayers(state compactBlockState) (world.Block, world.Liquid) {
	return structureLayers(bufferEntry{
		Block:  state.Block,
		Liquid: state.Liquid,
		HasLiq: state.HasLiq,
	})
}
