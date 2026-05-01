package editbrush

import (
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
)

// SendBrushForm shows the brush configuration form. Submitting binds the resulting
// BrushConfig to the player's held item.
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
