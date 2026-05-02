package service

import (
	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/parse"
)

const (
	BrushSphere    = "sphere"
	BrushCylinder  = "cylinder"
	BrushPyramid   = "pyramid"
	BrushCone      = "cone"
	BrushCube      = "cube"
	BrushFill      = "fill"
	BrushTopLayer  = "toplayer"
	BrushOverlay   = "overlay"
	BrushWrap      = "wrap"
	BrushPaint     = "paint"
	BrushPull      = "pull"
	BrushPush      = "push"
	BrushTerraform = "terraform"
	BrushSchematic = "schematic"
	BrushReplace   = "replace"
	BrushLine      = "line"
)

// BrushDefinition describes a supported brush type and the shared metadata used
// by service execution and form adapters.
type BrushDefinition struct {
	Type        string
	ShapeBrush  bool
	ShapeVolume bool
}

var brushDefinitions = []BrushDefinition{
	{Type: BrushSphere, ShapeBrush: true, ShapeVolume: true},
	{Type: BrushCylinder, ShapeBrush: true, ShapeVolume: true},
	{Type: BrushPyramid, ShapeBrush: true, ShapeVolume: true},
	{Type: BrushCone, ShapeBrush: true, ShapeVolume: true},
	{Type: BrushCube, ShapeBrush: true, ShapeVolume: true},
	{Type: BrushFill, ShapeVolume: true},
	{Type: BrushTopLayer, ShapeVolume: true},
	{Type: BrushOverlay, ShapeVolume: true},
	{Type: BrushWrap, ShapeVolume: true},
	{Type: BrushPaint, ShapeVolume: true},
	{Type: BrushPull, ShapeVolume: true},
	{Type: BrushPush, ShapeVolume: true},
	{Type: BrushTerraform, ShapeVolume: true},
	{Type: BrushSchematic},
	{Type: BrushReplace, ShapeVolume: true},
	{Type: BrushLine},
}

// BrushTypes returns supported brush type names in form display order.
func BrushTypes() []string {
	out := make([]string, 0, len(brushDefinitions))
	for _, def := range brushDefinitions {
		out = append(out, def.Type)
	}
	return out
}

// BrushShapes returns brush footprint shape names in form display order.
func BrushShapes() []string {
	out := make([]string, 0, 5)
	for _, def := range brushDefinitions {
		if def.ShapeBrush {
			out = append(out, def.Type)
		}
	}
	return out
}

func brushTypeUsesShapeVolume(brushType string) bool {
	for _, def := range brushDefinitions {
		if def.Type == brushType {
			return def.ShapeVolume
		}
	}
	return false
}

func isShapeBrush(brushType string) bool {
	for _, def := range brushDefinitions {
		if def.Type == brushType {
			return def.ShapeBrush
		}
	}
	return false
}

// BrushConfig is the JSON-serialised brush state stored by brush adapters.
//
// Type selects the brush behaviour; the remaining fields are only consulted by
// certain types. Defaults come from DefaultBrushConfig.
type BrushConfig struct {
	Type   string `json:"type"`
	Shape  string `json:"shape"`
	Mode   string `json:"mode"`
	Radius int    `json:"radius"`
	Height int    `json:"height"`
	Length int    `json:"length"`
	Width  int    `json:"width"`

	Blocks []parse.BlockState `json:"blocks,omitempty"`
	From   []parse.BlockState `json:"from,omitempty"`

	Thickness int     `json:"thickness"`
	Range     int     `json:"range"`
	Strength  float64 `json:"strength"`

	Hollow          bool `json:"hollow"`
	All             bool `json:"all"`
	ReplaceAir      bool `json:"replace_air"`
	NoAir           bool `json:"no_air"`
	ExtendWrap      bool `json:"extend_wrap"`
	PassThrough     bool `json:"pass_through"`
	RandomSchematic bool `json:"random_schematic"`
	RandomRotation  bool `json:"random_rotation"`

	Schematics []string `json:"schematics,omitempty"`
}

// DefaultBrushConfig returns factory defaults for quick //brush binding.
func DefaultBrushConfig() BrushConfig {
	return BrushConfig{Type: BrushSphere, Shape: BrushSphere, Mode: "erode", Radius: 3, Height: 5, Length: 5, Width: 5, Thickness: 1, Range: 32, Strength: 1}
}

func (c BrushConfig) shapeSpec() edit.ShapeSpec {
	kind := edit.ParseShapeKind(c.Shape)
	if isShapeBrush(c.Type) {
		kind = edit.ParseShapeKind(c.Type)
	}
	r := c.Radius
	if r <= 0 {
		r = 1
	}
	h := c.Height
	if h <= 0 {
		h = r*2 + 1
	}
	l, w := c.Length, c.Width
	if l <= 0 {
		l = r*2 + 1
	}
	if w <= 0 {
		w = r*2 + 1
	}
	return edit.ShapeSpec{Kind: kind, Radius: r, Height: h, Length: l, Width: w, Hollow: c.Hollow}
}

func (c BrushConfig) blockList() ([]world.Block, error) {
	if len(c.Blocks) == 0 {
		return []world.Block{mcblock.Stone{}}, nil
	}
	return statesToBlocks(c.Blocks)
}

func (c BrushConfig) fromList() ([]world.Block, error) {
	return statesToBlocks(c.From)
}

func statesToBlocks(states []parse.BlockState) ([]world.Block, error) {
	blocks := make([]world.Block, 0, len(states))
	for _, s := range states {
		b, err := parse.BlockFromState(s)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// StatesFromBlocks encodes blocks for JSON-backed brush config.
func StatesFromBlocks(blocks []world.Block) []parse.BlockState {
	states := make([]parse.BlockState, 0, len(blocks))
	for _, b := range blocks {
		states = append(states, parse.StateOfBlock(b))
	}
	return states
}
