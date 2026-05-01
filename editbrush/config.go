package editbrush

import (
	"encoding/json"
	"fmt"

	mcblock "github.com/df-mc/dragonfly/server/block"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/parse"
)

var brushTypes = []string{"sphere", "cylinder", "pyramid", "cone", "cube", "fill", "toplayer", "overlay", "wrap", "paint", "pull", "push", "terraform", "schematic", "replace", "line"}
var brushShapes = []string{"sphere", "cylinder", "pyramid", "cone", "cube"}

// BrushConfig is the JSON-serialised brush state stored on an item stack.
//
// Type selects the brush behaviour (see brushTypes); the remaining fields are
// only consulted by certain types. Defaults come from DefaultBrushConfig.
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
	return BrushConfig{Type: "sphere", Shape: "sphere", Mode: "erode", Radius: 3, Height: 5, Length: 5, Width: 5, Thickness: 1, Range: 32, Strength: 1}
}

func (c BrushConfig) shapeSpec() edit.ShapeSpec {
	kind := edit.ParseShapeKind(c.Shape)
	if c.Type == "sphere" || c.Type == "cylinder" || c.Type == "pyramid" || c.Type == "cone" || c.Type == "cube" {
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

// BindBrush serialises cfg onto the item stack.
func BindBrush(i item.Stack, cfg BrushConfig) (item.Stack, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return item.Stack{}, err
	}
	name := fmt.Sprintf("WorldEdit %s brush", cfg.Type)
	return i.WithValue(keys.BrushConfigKey, string(data)).WithCustomName(name), nil
}

// ConfigFromItem reads brush JSON from an item value.
func ConfigFromItem(i item.Stack) (BrushConfig, bool) {
	v, ok := i.Value(keys.BrushConfigKey)
	if !ok {
		return BrushConfig{}, false
	}
	var raw string
	switch t := v.(type) {
	case string:
		raw = t
	case []byte:
		raw = string(t)
	default:
		return BrushConfig{}, false
	}
	var cfg BrushConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return BrushConfig{}, false
	}
	return cfg, true
}
