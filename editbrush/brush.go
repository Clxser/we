package editbrush

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	mcblock "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/parse"
)

var brushTypes = []string{"sphere", "cylinder", "pyramid", "cone", "cube", "fill", "toplayer", "overlay", "wrap", "paint", "pull", "push", "terraform", "schematic", "replace", "line"}
var brushShapes = []string{"sphere", "cylinder", "pyramid", "cone", "cube"}

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

// SendBrushForm opens the configuration UI for //brush.
func SendBrushForm(p *player.Player) {
	p.SendForm(form.New(brushConfigForm{
		Type:            form.NewDropdown("Brush type", brushTypes, 0),
		Shape:           form.NewDropdown("Footprint shape", brushShapes, 0),
		Mode:            form.NewDropdown("Mode", []string{"erode", "expand"}, 0),
		Blocks:          form.NewInput("Blocks", "stone", "stone,dirt"),
		From:            form.NewInput("Replace/from blocks", "", "all or stone,dirt"),
		Schematics:      form.NewInput("Schematics", "", "name or name1,name2"),
		Radius:          form.NewSlider("Radius", 1, 32, 1, 3),
		Height:          form.NewSlider("Height/range Y", 1, 64, 1, 5),
		Length:          form.NewSlider("Length", 1, 64, 1, 5),
		Width:           form.NewSlider("Width", 1, 64, 1, 5),
		Thickness:       form.NewSlider("Line thickness", 1, 16, 1, 1),
		Range:           form.NewSlider("Line range", 1, 128, 1, 32),
		Strength:        form.NewSlider("Paint strength percent", 1, 100, 1, 100),
		Hollow:          form.NewToggle("Hollow shape", false),
		All:             form.NewToggle("Replace all block types", false),
		ReplaceAir:      form.NewToggle("Replace air", false),
		NoAir:           form.NewToggle("Do not paste air", true),
		ExtendWrap:      form.NewToggle("Extend wrap across same type", false),
		PassThrough:     form.NewToggle("Line passes through blocks", false),
		RandomSchematic: form.NewToggle("Random schematic", false),
		RandomRotation:  form.NewToggle("Random schematic rotation", false),
	}, "WorldEdit Brush"))
}

type brushConfigForm struct {
	Type       form.Dropdown
	Shape      form.Dropdown
	Mode       form.Dropdown
	Blocks     form.Input
	From       form.Input
	Schematics form.Input

	Radius    form.Slider
	Height    form.Slider
	Length    form.Slider
	Width     form.Slider
	Thickness form.Slider
	Range     form.Slider
	Strength  form.Slider

	Hollow          form.Toggle
	All             form.Toggle
	ReplaceAir      form.Toggle
	NoAir           form.Toggle
	ExtendWrap      form.Toggle
	PassThrough     form.Toggle
	RandomSchematic form.Toggle
	RandomRotation  form.Toggle
}

func (f brushConfigForm) Submit(submitter form.Submitter, _ *world.Tx) {
	p := submitter.(*player.Player)
	blocks, err := parse.ParseBlockList(f.Blocks.Value())
	if err != nil {
		p.Message(err.Error())
		return
	}
	var from []world.Block
	if strings.TrimSpace(f.From.Value()) != "" && !strings.EqualFold(strings.TrimSpace(f.From.Value()), "all") {
		from, err = parse.ParseBlockList(f.From.Value())
		if err != nil {
			p.Message(err.Error())
			return
		}
	}
	cfg := BrushConfig{
		Type:            brushTypes[f.Type.Value()],
		Shape:           brushShapes[f.Shape.Value()],
		Mode:            []string{"erode", "expand"}[f.Mode.Value()],
		Radius:          int(f.Radius.Value()),
		Height:          int(f.Height.Value()),
		Length:          int(f.Length.Value()),
		Width:           int(f.Width.Value()),
		Blocks:          StatesFromBlocks(blocks),
		From:            StatesFromBlocks(from),
		Thickness:       int(f.Thickness.Value()),
		Range:           int(f.Range.Value()),
		Strength:        f.Strength.Value() / 100,
		Hollow:          f.Hollow.Value(),
		All:             f.All.Value() || strings.EqualFold(strings.TrimSpace(f.From.Value()), "all"),
		ReplaceAir:      f.ReplaceAir.Value(),
		NoAir:           f.NoAir.Value(),
		ExtendWrap:      f.ExtendWrap.Value(),
		PassThrough:     f.PassThrough.Value(),
		RandomSchematic: f.RandomSchematic.Value(),
		RandomRotation:  f.RandomRotation.Value(),
		Schematics:      splitNames(f.Schematics.Value()),
	}
	held, off := p.HeldItems()
	bound, err := BindBrush(held, cfg)
	if err != nil {
		p.Message(err.Error())
		return
	}
	p.SetHeldItems(bound, off)
	p.Message("Brush bound to held item.")
}

func splitNames(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == ';' || r == '\n' || r == '\t' })
	out := fields[:0]
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func ApplyBrush(tx *world.Tx, p *player.Player, target cube.Pos, cfg BrushConfig, batch *history.Batch) error {
	blocks, err := cfg.blockList()
	if err != nil {
		return err
	}
	switch strings.ToLower(cfg.Type) {
	case "sphere", "cylinder", "pyramid", "cone", "cube":
		edit.ApplyShape(tx, target, cfg.shapeSpec(), blocks, batch)
	case "fill":
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if pos[1] <= target[1] {
				batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, rand.New(rand.NewSource(int64(pos[0]*31+pos[1]*17+pos[2])))))
			}
		})
	case "toplayer":
		brushTopLayer(tx, target, cfg, edit.BlockMask{All: true}, blocks, batch)
	case "overlay":
		brushOverlay(tx, target, cfg, blocks, batch)
	case "wrap":
		applyWrap(tx, target, cfg, blocks, batch)
	case "paint":
		applyPaint(tx, target, cfg, blocks, batch)
	case "pull", "push":
		applyPushPull(tx, p, target, cfg, strings.EqualFold(cfg.Type, "pull"), batch)
	case "terraform":
		applyTerraform(tx, target, cfg, blocks, batch)
	case "schematic":
		return applySchematicBrush(tx, target, p.Rotation().Direction(), cfg, batch)
	case "replace":
		from, err := cfg.fromList()
		if err != nil {
			return err
		}
		mask := edit.BlockMask{All: cfg.All, IncludeAir: cfg.ReplaceAir, Blocks: from}
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if mask.Match(tx.Block(pos)) {
				batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, rand.New(rand.NewSource(int64(pos[0]*31+pos[1]*17+pos[2])))))
			}
		})
	case "line":
		applyLineBrush(tx, p, cfg, blocks, batch)
	default:
		return fmt.Errorf("unknown brush type %q", cfg.Type)
	}
	return nil
}

func applyBrushShape(tx *world.Tx, target cube.Pos, cfg BrushConfig, f func(pos cube.Pos)) {
	spec := cfg.shapeSpec()
	area := spec.Bounds(target)
	area.Range(func(x, y, z int) {
		pos := cube.Pos{x, y, z}
		if spec.Hollow {
			if !spec.Shell(target, pos) {
				return
			}
		} else if !spec.Inside(target, pos) {
			return
		}
		f(pos)
	})
	_ = tx
}

func brushTopLayer(tx *world.Tx, target cube.Pos, cfg BrushConfig, mask edit.BlockMask, blocks []world.Block, batch *history.Batch) {
	spec := cfg.shapeSpec()
	area := spec.Bounds(target)
	r := rand.New(rand.NewSource(1))
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			for y := area.Max[1]; y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				if !spec.Inside(target, pos) {
					continue
				}
				b := tx.Block(pos)
				if parse.IsAir(b) {
					continue
				}
				if mask.Match(b) {
					batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, r))
				}
				break
			}
		}
	}
}

func brushOverlay(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	spec := cfg.shapeSpec()
	area := spec.Bounds(target)
	r := rand.New(rand.NewSource(1))
	for x := area.Min[0]; x <= area.Max[0]; x++ {
		for z := area.Min[2]; z <= area.Max[2]; z++ {
			for y := area.Max[1]; y >= area.Min[1]; y-- {
				pos := cube.Pos{x, y, z}
				if !spec.Inside(target, pos) {
					continue
				}
				if parse.IsAir(tx.Block(pos)) {
					continue
				}
				above := cube.Pos{x, y + 1, z}
				if spec.Inside(target, above) && parse.IsAir(tx.Block(above)) {
					batch.SetBlock(tx, above, edit.ChooseBlock(blocks, r))
				}
				break
			}
		}
	}
}

func isSurface(tx *world.Tx, pos cube.Pos) bool {
	if parse.IsAir(tx.Block(pos)) {
		return false
	}
	for _, f := range cube.Faces() {
		if parse.IsAir(tx.Block(pos.Side(f))) {
			return true
		}
	}
	return false
}

func applyWrap(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	r := rand.New(rand.NewSource(1))
	if cfg.ExtendWrap {
		base := tx.Block(target)
		seen := map[cube.Pos]bool{target: true}
		queue := []cube.Pos{target}
		limit := cfg.Radius * cfg.Radius
		for len(queue) > 0 {
			pos := queue[0]
			queue = queue[1:]
			wrapOne(tx, pos, blocks, r, batch)
			for _, f := range cube.Faces() {
				n := pos.Side(f)
				if seen[n] {
					continue
				}
				dx, dy, dz := n[0]-target[0], n[1]-target[1], n[2]-target[2]
				if dx*dx+dy*dy+dz*dz > limit || !parse.SameBlock(tx.Block(n), base) {
					continue
				}
				seen[n] = true
				queue = append(queue, n)
			}
		}
		return
	}
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		wrapOne(tx, pos, blocks, r, batch)
	})
}

func wrapOne(tx *world.Tx, pos cube.Pos, blocks []world.Block, r *rand.Rand, batch *history.Batch) {
	if parse.IsAir(tx.Block(pos)) {
		return
	}
	for _, f := range cube.Faces() {
		n := pos.Side(f)
		if parse.IsAir(tx.Block(n)) {
			batch.SetBlock(tx, n, edit.ChooseBlock(blocks, r))
		}
	}
}

func applyPaint(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	r := rand.New(rand.NewSource(1))
	strength := math.Max(0, math.Min(1, cfg.Strength))
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		if isSurface(tx, pos) && r.Float64() <= strength {
			batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, r))
		}
	})
}

func applyPushPull(tx *world.Tx, p *player.Player, target cube.Pos, cfg BrushConfig, pull bool, batch *history.Batch) {
	dir := dominantDir(target, cube.PosFromVec3(p.Position()))
	if !pull {
		dir = cube.Pos{-dir[0], -dir[1], -dir[2]}
	}
	var positions []cube.Pos
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		if !parse.IsAir(tx.Block(pos)) {
			positions = append(positions, pos)
		}
	})
	sortPositionsForMove(positions, dir)
	snap := make(map[cube.Pos]history.BlockSnapshot, len(positions))
	for _, pos := range positions {
		snap[pos] = history.SnapshotAtBlock(tx, pos)
	}
	for _, pos := range positions {
		batch.SetBlock(tx, pos, mcblock.Air{})
		batch.SetLiquid(tx, pos, nil)
	}
	for _, pos := range positions {
		dst := pos.Add(dir)
		i := batch.EnsurePos(tx, dst)
		history.ApplyBlockSnapshot(tx, dst, snap[pos])
		batch.SetAfterForIndex(tx, i, dst)
	}
}

func sortPositionsForMove(positions []cube.Pos, dir cube.Pos) {
	sort.Slice(positions, func(i, j int) bool {
		a, b := positions[i], positions[j]
		return a[0]*dir[0]+a[1]*dir[1]+a[2]*dir[2] > b[0]*dir[0]+b[1]*dir[1]+b[2]*dir[2]
	})
}

func dominantDir(from, to cube.Pos) cube.Pos {
	dx, dy, dz := to[0]-from[0], to[1]-from[1], to[2]-from[2]
	if absInt(dx) >= absInt(dy) && absInt(dx) >= absInt(dz) {
		if dx >= 0 {
			return cube.Pos{1, 0, 0}
		}
		return cube.Pos{-1, 0, 0}
	}
	if absInt(dy) >= absInt(dx) && absInt(dy) >= absInt(dz) {
		if dy >= 0 {
			return cube.Pos{0, 1, 0}
		}
		return cube.Pos{0, -1, 0}
	}
	if dz >= 0 {
		return cube.Pos{0, 0, 1}
	}
	return cube.Pos{0, 0, -1}
}

func applyTerraform(tx *world.Tx, target cube.Pos, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	r := rand.New(rand.NewSource(1))
	if strings.EqualFold(cfg.Mode, "expand") {
		applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
			if !parse.IsAir(tx.Block(pos)) {
				return
			}
			for _, f := range cube.Faces() {
				if !parse.IsAir(tx.Block(pos.Side(f))) {
					batch.SetBlock(tx, pos, edit.ChooseBlock(blocks, r))
					return
				}
			}
		})
		return
	}
	applyBrushShape(tx, target, cfg, func(pos cube.Pos) {
		if isSurface(tx, pos) {
			batch.SetBlock(tx, pos, mcblock.Air{})
		}
	})
}

func applySchematicBrush(tx *world.Tx, target cube.Pos, dir cube.Direction, cfg BrushConfig, batch *history.Batch) error {
	if len(cfg.Schematics) == 0 {
		return fmt.Errorf("schematic brush has no schematics selected")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	name := cfg.Schematics[0]
	if cfg.RandomSchematic {
		name = cfg.Schematics[r.Intn(len(cfg.Schematics))]
	}
	cb, err := edit.LoadSchematic(name)
	if err != nil {
		return err
	}
	if cfg.RandomRotation {
		dirs := []cube.Direction{cube.North, cube.East, cube.South, cube.West}
		dir = dirs[r.Intn(len(dirs))]
	}
	return edit.PasteClipboard(tx, cb, target, dir, cfg.NoAir, batch)
}

func applyLineBrush(tx *world.Tx, p *player.Player, cfg BrushConfig, blocks []world.Block, batch *history.Batch) {
	start := cube.PosFromVec3(p.Position().Add(p.Rotation().Vec3()))
	step := p.Rotation().Vec3()
	last := start
	for i := 0; i < max(1, cfg.Range); i++ {
		pos := cube.PosFromVec3(p.Position().Add(step.Mul(float64(i + 1))))
		if !cfg.PassThrough && !parse.IsAir(tx.Block(pos)) && i > 0 {
			break
		}
		last = pos
	}
	edit.Line(tx, start, last, max(1, cfg.Thickness), blocks, batch)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
