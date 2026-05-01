package service

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/parse"
)

func Set(tx *world.Tx, s Session, blockSpec string) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.FillArea(tx, area, blocks, batch)
	return record(s, batch), nil
}

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

func Walls(tx *world.Tx, s Session, blockSpec string) (ChangeResult, error) {
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.Walls(tx, area, blocks, batch)
	return record(s, batch), nil
}

func Drain(tx *world.Tx, s Session, center cube.Pos, radius int) (ChangeResult, error) {
	if radius < 1 {
		return ChangeResult{}, fmt.Errorf("radius must be positive")
	}
	batch := history.NewBatch(false)
	edit.Drain(tx, center, radius, batch)
	return record(s, batch), nil
}

func BiomeNames() []string {
	bs := world.Biomes()
	names := make([]string, 0, len(bs))
	for _, b := range bs {
		names = append(names, b.String())
	}
	return names
}

func SetBiome(tx *world.Tx, s Session, name string) (world.Biome, error) {
	b, ok := world.BiomeByName(name)
	if !ok {
		return nil, fmt.Errorf("unknown biome %q", name)
	}
	area, err := selectedArea(s)
	if err != nil {
		return nil, err
	}
	batch := history.NewBatch(false)
	area.Range(func(x, y, z int) { batch.SetBiome(tx, cube.Pos{x, y, z}, b) })
	s.Record(batch)
	return b, nil
}

func Replace(tx *world.Tx, s Session, args []string) (ChangeResult, error) {
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
	batch := history.NewBatch(false)
	edit.ReplaceArea(tx, area, mask, to, batch)
	return record(s, batch), nil
}

func ReplaceNear(tx *world.Tx, s Session, center cube.Pos, distance int, args []string) (ChangeResult, error) {
	if distance < 1 || len(args) < 2 {
		return ChangeResult{}, fmt.Errorf("usage: //replacenear <distance> <from> <to>")
	}
	mask, to, err := ParseMaskTo(args)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.ReplaceNear(tx, center, distance, mask, to, batch)
	return record(s, batch), nil
}

func TopLayer(tx *world.Tx, s Session, args []string) (ChangeResult, error) {
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
	batch := history.NewBatch(false)
	edit.TopLayer(tx, area, mask, to, batch)
	return record(s, batch), nil
}

func Overlay(tx *world.Tx, s Session, blockSpec string) (ChangeResult, error) {
	blocks, err := parse.ParseBlockList(blockSpec)
	if err != nil {
		return ChangeResult{}, err
	}
	area, err := selectedArea(s)
	if err != nil {
		return ChangeResult{}, err
	}
	batch := history.NewBatch(false)
	edit.Overlay(tx, area, blocks, batch)
	return record(s, batch), nil
}
