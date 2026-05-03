package service

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

// Set fills the current selection with random picks from blockSpec.
func Set(tx *world.Tx, s Session, blockSpec string) (ChangeResult, error) {
	return SetWithOptions(tx, s, blockSpec, EditOptions{})
}

// SetWithOptions fills the current selection with random picks from blockSpec.
func SetWithOptions(tx *world.Tx, s Session, blockSpec string, opts EditOptions) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return ChangeResult{}, err
	}
	if opts.NoUndo {
		edit.FillArea(tx, area, blocks, nil)
		return ChangeResult{Changed: int(area.Volume())}, nil
	}
	batch := history.NewBatch(false)
	edit.FillArea(tx, area, blocks, batch)
	return record(s, batch), nil
}

// Center places one block at the centre of the selection and returns its position.
func Center(tx *world.Tx, s Session, blockSpec string) (PositionResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return PositionResult{}, err
	}
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return PositionResult{}, err
	}
	batch := history.NewBatch(false)
	pos := edit.Center(tx, area, blocks, batch)
	result := record(s, batch)
	return PositionResult{Changed: result.Changed, Pos: pos}, nil
}

// Walls fills only the outer shell of the selection cuboid.
func Walls(tx *world.Tx, s Session, blockSpec string) (ChangeResult, error) {
	return WallsWithOptions(tx, s, blockSpec, EditOptions{})
}

func WallsWithOptions(tx *world.Tx, s Session, blockSpec string, opts EditOptions) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.Walls(tx, area, blocks, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}

// Drain removes water and lava in a sphere of the given radius around center.
func Drain(tx *world.Tx, s Session, center cube.Pos, radius int) (ChangeResult, error) {
	return DrainWithOptions(tx, s, center, radius, EditOptions{})
}

func DrainWithOptions(tx *world.Tx, s Session, center cube.Pos, radius int, opts EditOptions) (ChangeResult, error) {
	if radius < 1 {
		return ChangeResult{}, fmt.Errorf("radius must be positive")
	}
	// Drain only writes inside the sphere, but checking the bounding cube is a
	// conservative cap for client-cache pressure.
	area := geo.NewArea(center[0]-radius, center[1]-radius, center[2]-radius, center[0]+radius, center[1]+radius, center[2]+radius)
	if err := checkArea(guardrailsFor(s), area); err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.Drain(tx, center, radius, batch)
	diameter := radius*2 + 1
	return finishEdit(s, batch, diameter*diameter*diameter), nil
}

// BiomeNames returns the names of every biome registered with Dragonfly.
func BiomeNames() []string {
	bs := world.Biomes()
	names := make([]string, 0, len(bs))
	for _, b := range bs {
		names = append(names, b.String())
	}
	return names
}

// SetBiome sets the biome of every block in the selection. Returns the resolved
// biome on success, or an error if name does not match a registered biome.
func SetBiome(tx *world.Tx, s Session, name string) (world.Biome, error) {
	return SetBiomeWithOptions(tx, s, name, EditOptions{})
}

func SetBiomeWithOptions(tx *world.Tx, s Session, name string, opts EditOptions) (world.Biome, error) {
	b, ok := world.BiomeByName(name)
	if !ok {
		return nil, fmt.Errorf("unknown biome %q", name)
	}
	area, err := selectedArea(s)
	if err != nil {
		return nil, err
	}
	if opts.NoUndo {
		area.Range(func(x, y, z int) { tx.SetBiome(cube.Pos{x, y, z}, b) })
		return b, nil
	}
	batch := history.NewBatch(false)
	area.Range(func(x, y, z int) { batch.SetBiome(tx, cube.Pos{x, y, z}, b) })
	s.Record(batch)
	return b, nil
}

// Replace swaps blocks matching args[0] for picks from args[1:] inside the selection.
func Replace(tx *world.Tx, s Session, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return ReplaceWithOptions(tx, s, args, opts)
}

func ReplaceWithOptions(tx *world.Tx, s Session, args []string, opts EditOptions) (ChangeResult, error) {
	if len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //replace <all|from> <to>")
	}
	mask, to, err := ParseMaskTo(args)
	if err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.ReplaceArea(tx, area, mask, to, batch)
	return finishEdit(s, batch, int(area.Volume())), nil
}

// ReplaceNear runs Replace inside a sphere of the given distance around center,
// independent of the selection.
func ReplaceNear(tx *world.Tx, s Session, center cube.Pos, distance int, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return ReplaceNearWithOptions(tx, s, center, distance, args, opts)
}

func ReplaceNearWithOptions(tx *world.Tx, s Session, center cube.Pos, distance int, args []string, opts EditOptions) (ChangeResult, error) {
	if distance < 1 || len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //replacenear <distance> <from> <to>")
	}
	mask, to, err := ParseMaskTo(args)
	if err != nil {
		return ChangeResult{}, err
	}
	area := geo.NewArea(center[0]-distance, center[1]-distance, center[2]-distance, center[0]+distance, center[1]+distance, center[2]+distance)
	if err := checkArea(guardrailsFor(s), area); err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.ReplaceNear(tx, center, distance, mask, to, batch)
	diameter := distance*2 + 1
	return finishEdit(s, batch, diameter*diameter*diameter), nil
}

// TopLayer replaces only the topmost matching block in each (x, z) column of the selection.
func TopLayer(tx *world.Tx, s Session, args []string) (ChangeResult, error) {
	args, opts := ParseEditOptions(args)
	return TopLayerWithOptions(tx, s, args, opts)
}

func TopLayerWithOptions(tx *world.Tx, s Session, args []string, opts EditOptions) (ChangeResult, error) {
	if len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //toplayer <all|only:types> <to>")
	}
	mask, to, err := ParseMaskTo(args)
	if err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.TopLayer(tx, area, mask, to, batch)
	return finishEdit(s, batch, area.Dx()*area.Dz()), nil
}

// Overlay places blocks one layer above the highest non-air block in each column.
func Overlay(tx *world.Tx, s Session, blockSpec string) (ChangeResult, error) {
	return OverlayWithOptions(tx, s, blockSpec, EditOptions{})
}

func OverlayWithOptions(tx *world.Tx, s Session, blockSpec string, opts EditOptions) (ChangeResult, error) {
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := historyBatch(opts)
	edit.Overlay(tx, area, blocks, batch)
	return finishEdit(s, batch, area.Dx()*area.Dz()), nil
}
