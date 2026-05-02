package cmd

import (
	"sync"

	dcf "github.com/df-mc/dragonfly/server/cmd"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/we/edit"
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
	{"/clearclipboard", "Clear clipboard", []string{"clearclipboard"}, ClearClipboardCommand{}},
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
	{"/overlay", "Overlay top layer", []string{"overlay", "/layer", "layer"}, OverlayCommand{}},
	{"/removeabove", "Remove blocks above player", []string{"removeabove"}, RemoveAboveCommand{}},
	{"/removebelow", "Remove blocks below player", []string{"removebelow"}, RemoveBelowCommand{}},
	{"/removenear", "Remove nearby matching blocks", []string{"removenear"}, RemoveNearCommand{}},
	{"/naturalize", "Naturalize selected terrain", []string{"naturalize"}, NaturalizeCommand{}},
	{"/move", "Move selection", []string{"move"}, MoveCommand{}},
	{"/stack", "Stack selection", []string{"stack"}, StackCommand{}},
	{"/rotate", "Rotate clipboard", []string{"rotate"}, RotateCommand{}},
	{"/flip", "Flip clipboard", []string{"flip"}, FlipCommand{}},
	{"/line", "Draw line from pos1 to pos2", []string{"line"}, LineCommand{}},
	{"/sphere", "Create sphere", []string{"sphere"}, ShapeCommand{Kind: edit.ShapeSphere}},
	{"/cylinder", "Create cylinder", []string{"cylinder"}, ShapeCommand{Kind: edit.ShapeCylinder}},
	{"/pyramid", "Create pyramid", []string{"pyramid"}, ShapeCommand{Kind: edit.ShapePyramid}},
	{"/cone", "Create cone", []string{"cone"}, ShapeCommand{Kind: edit.ShapeCone}},
	{"/cube", "Create cube", []string{"cube"}, ShapeCommand{Kind: edit.ShapeCube}},
	{"/brush", "Bind brush to held item", []string{"brush"}, BrushCommand{}},
	{"/searchitem", "Search registered items", []string{"searchitem", "/search", "search", "/l", "l"}, SearchItemCommand{}},
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
