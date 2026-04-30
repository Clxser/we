package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	dcf "github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/editbrush"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/parse"
	"github.com/df-mc/we/session"
)

var registerOnce sync.Once

// commandDefs is the single source of truth for Dragonfly cmd registration.
// Names starting with "/" support the Bedrock double-slash UX: "//set" strips
// one slash and resolves to command name "/set".
var commandDefs = []struct {
	name, desc string
	aliases    []string
	r          dcf.Runnable
}{
	{"/wand", "WorldEdit selection wand", []string{"wand"}, WandCommand{}},
	{"/pos1", "Set WorldEdit position 1", []string{"pos1"}, Pos1Command{}},
	{"/pos2", "Set WorldEdit position 2", []string{"pos2"}, Pos2Command{}},
	{"/set", "Fill selected area", []string{"set", "/fill", "fill"}, SetCommand{}},
	{"/copy", "Copy selected area", []string{"copy"}, CopyCommand{}},
	{"/paste", "Paste clipboard", []string{"paste"}, PasteCommand{}},
	{"/cut", "Cut selected area", []string{"cut"}, CutCommand{}},
	{"/schematic", "Manage schematics", []string{"schematic", "/schem", "schem"}, SchematicCommand{}},
	{"/undo", "Undo WorldEdit change", []string{"undo"}, UndoCommand{}},
	{"/redo", "Redo WorldEdit change", []string{"redo"}, RedoCommand{}},
	{"/center", "Mark selection center", []string{"center"}, CenterCommand{}},
	{"/walls", "Build selection walls", []string{"walls"}, WallsCommand{}},
	{"/drain", "Drain fluids", []string{"drain"}, DrainCommand{}},
	{"/biome", "List or set biomes", []string{"biome"}, BiomeCommand{}},
	{"/replace", "Replace selected blocks", []string{"replace"}, ReplaceCommand{}},
	{"/replacenear", "Replace nearby blocks", []string{"replacenear"}, ReplaceNearCommand{}},
	{"/toplayer", "Replace top layer", []string{"toplayer"}, TopLayerCommand{}},
	{"/overlay", "Overlay top layer", []string{"overlay"}, OverlayCommand{}},
	{"/move", "Move selection", []string{"move"}, MoveCommand{}},
	{"/stack", "Stack selection", []string{"stack"}, StackCommand{}},
	{"/rotate", "Rotate selection copy", []string{"rotate"}, RotateCommand{}},
	{"/flip", "Flip selection copy", []string{"flip"}, FlipCommand{}},
	{"/line", "Draw line from pos1 to pos2", []string{"line"}, LineCommand{}},
	{"/sphere", "Create sphere", []string{"sphere"}, ShapeCommand{Kind: edit.ShapeSphere}},
	{"/cylinder", "Create cylinder", []string{"cylinder"}, ShapeCommand{Kind: edit.ShapeCylinder}},
	{"/pyramid", "Create pyramid", []string{"pyramid"}, ShapeCommand{Kind: edit.ShapePyramid}},
	{"/cone", "Create cone", []string{"cone"}, ShapeCommand{Kind: edit.ShapeCone}},
	{"/cube", "Create cube", []string{"cube"}, ShapeCommand{Kind: edit.ShapeCube}},
	{"/brush", "Bind brush to held item", []string{"brush"}, BrushCommand{}},
}

func registerAll() {
	for _, e := range commandDefs {
		reg(e.name, e.desc, e.aliases, e.r)
	}
}

func init() {
	registerOnce.Do(registerAll)
}

// RegisterCommands is idempotent; commands are normally registered from init
// when this package is imported. Call this if you import this package indirectly
// and need to force registration before init ordering would run.
func RegisterCommands() {
	registerOnce.Do(registerAll)
}

func reg(name, desc string, aliases []string, r ...dcf.Runnable) {
	dcf.Register(dcf.New(name, desc, aliases, r...))
}

type playerCommand struct{}

func (playerCommand) Allow(src dcf.Source) bool {
	_, ok := src.(*player.Player)
	return ok
}

type WandCommand struct{ playerCommand }

func (WandCommand) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	held, off := p.HeldItems()
	wand := item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1).
		WithValue(keys.WandItemKey, true).
		WithCustomName("WorldEdit Wand")
	if !held.Empty() {
		wand = held.WithValue(keys.WandItemKey, true).WithCustomName("WorldEdit Wand")
	}
	p.SetHeldItems(wand, off)
	o.Print("WorldEdit wand assigned. Break a block for pos1, use on a block for pos2.")
}

type Pos1Command struct{ playerCommand }
type Pos2Command struct{ playerCommand }

func (Pos1Command) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	pos := cube.PosFromVec3(p.Position())
	if session.Ensure(p).SetPos1(pos) {
		o.Printf("pos1 set to %v", pos)
		return
	}
	o.Printf("pos1 unchanged (%v)", pos)
}

func (Pos2Command) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	pos := cube.PosFromVec3(p.Position())
	if session.Ensure(p).SetPos2(pos) {
		o.Printf("pos2 set to %v", pos)
		return
	}
	o.Printf("pos2 unchanged (%v)", pos)
}

type SetCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c SetCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.FillArea(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Set %d blocks.", batch.Len())
}

type CopyCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c CopyCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	args := strings.Fields(string(c.Args))
	only := len(args) > 0 && strings.EqualFold(args[0], "only")
	mask := edit.BlockMask{All: true, IncludeAir: true}
	if only {
		if len(args) < 2 {
			o.Error("copy only requires block types")
			return
		}
		blocks, err := parse.ParseBlockList(strings.Join(args[1:], " "))
		if err != nil {
			o.Error(err)
			return
		}
		mask = edit.BlockMask{Blocks: blocks}
	}
	cb := edit.CopySelection(tx, area, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), mask, only)
	session.Ensure(p).SetClipboard(cb)
	o.Printf("Copied %d blocks.", len(cb.Entries))
}

type PasteCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c PasteCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	cb, ok := session.Ensure(p).Clipboard()
	if !ok {
		o.Error("clipboard is empty")
		return
	}
	batch := history.NewBatch(false)
	if err := edit.PasteClipboard(tx, cb, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), hasFlag(strings.Fields(string(c.Args)), "-a"), batch); err != nil {
		o.Error(err)
		return
	}
	record(p, batch)
	o.Printf("Pasted %d blocks.", batch.Len())
}

type CutCommand struct{ playerCommand }

func (CutCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	cb := edit.CopySelection(tx, area, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), edit.BlockMask{All: true, IncludeAir: true}, false)
	session.Ensure(p).SetClipboard(cb)
	batch := history.NewBatch(false)
	edit.ClearArea(tx, area, batch)
	record(p, batch)
	o.Printf("Cut %d blocks.", batch.Len())
}

type SchematicCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c SchematicCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) == 0 {
		o.Error("usage: //schematic <create|paste|delete|list> [name] [-a]")
		return
	}
	switch strings.ToLower(args[0]) {
	case "create":
		if len(args) < 2 {
			o.Error("schematic create requires a name")
			return
		}
		area, ok := selectedArea(p, o)
		if !ok {
			return
		}
		cb := edit.CopySelection(tx, area, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), edit.BlockMask{All: true, IncludeAir: true}, false)
		if err := edit.SaveSchematic(args[1], cb); err != nil {
			o.Error(err)
			return
		}
		o.Printf("Saved schematic %q.", args[1])
	case "paste":
		if len(args) < 2 {
			o.Error("schematic paste requires a name")
			return
		}
		cb, err := edit.LoadSchematic(args[1])
		if err != nil {
			o.Error(err)
			return
		}
		batch := history.NewBatch(false)
		if err := edit.PasteClipboard(tx, cb, cube.PosFromVec3(p.Position()), p.Rotation().Direction(), hasFlag(args[2:], "-a"), batch); err != nil {
			o.Error(err)
			return
		}
		record(p, batch)
		o.Printf("Pasted schematic %q.", args[1])
	case "delete":
		if len(args) < 2 {
			o.Error("schematic delete requires a name")
			return
		}
		if err := edit.DeleteSchematic(args[1]); err != nil {
			o.Error(err)
			return
		}
		o.Printf("Deleted schematic %q.", args[1])
	case "list":
		names, err := edit.ListSchematics()
		if err != nil {
			o.Error(err)
			return
		}
		o.Print("Schematics: " + strings.Join(names, ", "))
	default:
		o.Error("unknown schematic subcommand")
	}
}

type UndoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c UndoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	brush := optionalB(c.Target)
	if !session.Ensure(p).Undo(tx, brush) {
		o.Error("nothing to undo")
		return
	}
	o.Print("Undo successful.")
}

type RedoCommand struct {
	playerCommand
	Target dcf.Optional[string] `cmd:"target"`
}

func (c RedoCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	brush := optionalB(c.Target)
	if !session.Ensure(p).Redo(tx, brush) {
		o.Error("nothing to redo")
		return
	}
	o.Print("Redo successful.")
}

type CenterCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c CenterCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	pos := edit.Center(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Marked center at %v.", pos)
}

type WallsCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c WallsCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.Walls(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Built walls with %d changes.", batch.Len())
}

type DrainCommand struct {
	playerCommand
	Radius int `cmd:"radius"`
}

func (c DrainCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	if c.Radius < 1 {
		o.Error("radius must be positive")
		return
	}
	batch := history.NewBatch(false)
	edit.Drain(tx, cube.PosFromVec3(p.Position()), c.Radius, batch)
	record(p, batch)
	o.Printf("Drained %d blocks.", batch.Len())
}

type BiomeCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c BiomeCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) == 0 || strings.EqualFold(args[0], "list") {
		bs := world.Biomes()
		names := make([]string, 0, len(bs))
		for _, b := range bs {
			names = append(names, b.String())
		}
		o.Print("Biomes: " + strings.Join(names, ", "))
		return
	}
	if !strings.EqualFold(args[0], "set") || len(args) < 2 {
		o.Error("usage: //biome list | //biome set <biome>")
		return
	}
	b, ok := world.BiomeByName(args[1])
	if !ok {
		o.Errorf("unknown biome %q", args[1])
		return
	}
	area, selected := selectedArea(p, o)
	if !selected {
		return
	}
	batch := history.NewBatch(false)
	area.Range(func(x, y, z int) { batch.SetBiome(tx, cube.Pos{x, y, z}, b) })
	record(p, batch)
	o.Printf("Set biome %s.", b.String())
}

type ReplaceCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c ReplaceCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //replace <all|from> <to>")
		return
	}
	mask, to, err := parseMaskTo(args)
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.ReplaceArea(tx, area, mask, to, batch)
	record(p, batch)
	o.Printf("Replaced %d blocks.", batch.Len())
}

type ReplaceNearCommand struct {
	playerCommand
	Distance int         `cmd:"distance"`
	Args     dcf.Varargs `cmd:"args"`
}

func (c ReplaceNearCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if c.Distance < 1 || len(args) < 2 {
		o.Error("usage: //replacenear <distance> <from> <to>")
		return
	}
	mask, to, err := parseMaskTo(args)
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.ReplaceNear(tx, cube.PosFromVec3(p.Position()), c.Distance, mask, to, batch)
	record(p, batch)
	o.Printf("Replaced %d nearby blocks.", batch.Len())
}

type TopLayerCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c TopLayerCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //toplayer <all|only:types> <to>")
		return
	}
	mask, to, err := parseMaskTo(args)
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.TopLayer(tx, area, mask, to, batch)
	record(p, batch)
	o.Printf("Replaced %d top-layer blocks.", batch.Len())
}

type OverlayCommand struct {
	playerCommand
	Blocks dcf.Varargs `cmd:"blocks"`
}

func (c OverlayCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	blocks, err := parse.ParseBlockList(string(c.Blocks))
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.Overlay(tx, area, blocks, batch)
	record(p, batch)
	o.Printf("Overlay changed %d blocks.", batch.Len())
}

type MoveCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c MoveCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //move <all|only:types> <distance> [-a]")
		return
	}
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		o.Error(err)
		return
	}
	dist, err := strconv.Atoi(args[1])
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.Move(tx, area, edit.DirectionVector(p.Rotation().Direction().Face()), dist, mask, hasFlag(args[2:], "-a"), batch)
	record(p, batch)
	o.Printf("Moved %d blocks.", batch.Len())
}

type StackCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c StackCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 1 {
		o.Error("usage: //stack <amount> [-a]")
		return
	}
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		o.Error(err)
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.Stack(tx, area, edit.DirectionVector(p.Rotation().Direction().Face()), amount, hasFlag(args[1:], "-a"), batch)
	record(p, batch)
	o.Printf("Stacked with %d changes.", batch.Len())
}

type RotateCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c RotateCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 1 {
		o.Error("usage: //rotate <90|180|270|360> [x|y|z]")
		return
	}
	deg, err := strconv.Atoi(args[0])
	if err != nil || (deg != 90 && deg != 180 && deg != 270 && deg != 360) {
		o.Error("rotation must be one of 90, 180, 270, or 360")
		return
	}
	axis := "y"
	if len(args) > 1 {
		axis = args[1]
	}
	if !validAxis(axis) {
		o.Error("axis must be x, y, or z")
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.RotateCopy(tx, area, deg, axis, batch)
	record(p, batch)
	o.Printf("Rotated copy with %d changes.", batch.Len())
}

type FlipCommand struct {
	playerCommand
	Axis dcf.Optional[string] `cmd:"axis"`
}

func (c FlipCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	axis := "x"
	if v, ok := c.Axis.Load(); ok {
		axis = v
	} else {
		switch p.Rotation().Direction() {
		case cube.North, cube.South:
			axis = "z"
		default:
			axis = "x"
		}
	}
	if !validAxis(axis) {
		o.Error("axis must be x, y, or z")
		return
	}
	area, ok := selectedArea(p, o)
	if !ok {
		return
	}
	batch := history.NewBatch(false)
	edit.FlipCopy(tx, area, axis, batch)
	record(p, batch)
	o.Printf("Flipped copy with %d changes.", batch.Len())
}

type LineCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c LineCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	if len(args) < 2 {
		o.Error("usage: //line <blocks> <thickness>")
		return
	}
	blocks, err := parse.ParseBlockList(args[0])
	if err != nil {
		o.Error(err)
		return
	}
	thickness, err := strconv.Atoi(args[1])
	if err != nil {
		o.Error(err)
		return
	}
	pos1, pos2, ok := session.Ensure(p).PosCorners()
	if !ok {
		o.Error("pos1 and pos2 must be set first")
		return
	}
	batch := history.NewBatch(false)
	edit.Line(tx, pos1, pos2, thickness, blocks, batch)
	record(p, batch)
	o.Printf("Drew line with %d changes.", batch.Len())
}

type ShapeCommand struct {
	playerCommand
	Kind edit.ShapeKind `cmd:"-"`
	Args dcf.Varargs    `cmd:"args"`
}

func (c ShapeCommand) Run(src dcf.Source, o *dcf.Output, tx *world.Tx) {
	p := src.(*player.Player)
	args := strings.Fields(string(c.Args))
	hollow := hasFlag(args, "-h")
	args = removeFlags(args, "-h")
	spec, blocks, err := parseShapeArgs(c.Kind, args, hollow)
	if err != nil {
		o.Error(err)
		return
	}
	batch := history.NewBatch(false)
	edit.ApplyShape(tx, cube.PosFromVec3(p.Position()), spec, blocks, batch)
	record(p, batch)
	o.Printf("Created %s with %d changes.", c.Kind, batch.Len())
}

type BrushCommand struct {
	playerCommand
	Args dcf.Varargs `cmd:"args"`
}

func (c BrushCommand) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	held, off := p.HeldItems()
	if held.Empty() {
		o.Error("hold an item before running //brush")
		return
	}
	args := strings.Fields(string(c.Args))
	if len(args) == 0 {
		editbrush.SendBrushForm(p)
		o.Print("Opened brush menu.")
		return
	}
	cfg := editbrush.DefaultBrushConfig()
	cfg.Type = strings.ToLower(args[0])
	cfg.Shape = cfg.Type
	if len(args) > 1 {
		blocks, err := parse.ParseBlockList(args[1])
		if err != nil {
			o.Error(err)
			return
		}
		cfg.Blocks = editbrush.StatesFromBlocks(blocks)
	}
	if len(args) > 2 {
		if r, err := strconv.Atoi(args[2]); err == nil {
			cfg.Radius = r
			cfg.Height = r*2 + 1
			cfg.Length = r*2 + 1
			cfg.Width = r*2 + 1
		}
	}
	bound, err := editbrush.BindBrush(held, cfg)
	if err != nil {
		o.Error(err)
		return
	}
	p.SetHeldItems(bound, off)
	o.Printf("Bound %s brush.", cfg.Type)
}

func selectedArea(p *player.Player, o *dcf.Output) (geo.Area, bool) {
	a, ok := session.Ensure(p).SelectionArea()
	if !ok {
		o.Error("pos1 and pos2 must be set first")
		return geo.Area{}, false
	}
	return a, true
}

func record(p *player.Player, batch *history.Batch) {
	session.Ensure(p).Record(batch)
}

func optionalB(o dcf.Optional[string]) bool {
	v, ok := o.Load()
	return ok && strings.EqualFold(v, "b")
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(a, flag) {
			return true
		}
	}
	return false
}

func removeFlags(args []string, flags ...string) []string {
	var out []string
	for _, a := range args {
		remove := false
		for _, f := range flags {
			if strings.EqualFold(a, f) {
				remove = true
				break
			}
		}
		if !remove {
			out = append(out, a)
		}
	}
	return out
}

func parseMaskTo(args []string) (edit.BlockMask, []world.Block, error) {
	mask, err := edit.ParseMask(args[0])
	if err != nil {
		return edit.BlockMask{}, nil, err
	}
	to, err := parse.ParseBlockList(strings.Join(args[1:], " "))
	return mask, to, err
}

func validAxis(axis string) bool {
	switch strings.ToLower(axis) {
	case "x", "y", "z":
		return true
	default:
		return false
	}
}

func parseShapeArgs(kind edit.ShapeKind, args []string, hollow bool) (edit.ShapeSpec, []world.Block, error) {
	if len(args) < 3 {
		return edit.ShapeSpec{}, nil, fmt.Errorf("not enough shape arguments")
	}
	blocks, err := parse.ParseBlockList(args[0])
	if err != nil {
		return edit.ShapeSpec{}, nil, err
	}
	spec := edit.ShapeSpec{Kind: kind, Hollow: hollow}
	switch kind {
	case edit.ShapeSphere:
		r, err1 := strconv.Atoi(args[1])
		h, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			return edit.ShapeSpec{}, nil, fmt.Errorf("radius and height must be numbers")
		}
		spec.Radius, spec.Height = r, h
	case edit.ShapeCylinder, edit.ShapeCone:
		r, err1 := strconv.Atoi(args[1])
		h, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			return edit.ShapeSpec{}, nil, fmt.Errorf("radius and height must be numbers")
		}
		spec.Radius, spec.Height = r, h
	case edit.ShapePyramid, edit.ShapeCube:
		if len(args) < 4 {
			return edit.ShapeSpec{}, nil, fmt.Errorf("length, width, and height are required")
		}
		l, err1 := strconv.Atoi(args[1])
		w, err2 := strconv.Atoi(args[2])
		h, err3 := strconv.Atoi(args[3])
		if err1 != nil || err2 != nil || err3 != nil {
			return edit.ShapeSpec{}, nil, fmt.Errorf("length, width, and height must be numbers")
		}
		spec.Length, spec.Width, spec.Height = l, w, h
	}
	return spec, blocks, nil
}
