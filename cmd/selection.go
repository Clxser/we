package cmd

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	dcf "github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/selectionui"
	"github.com/df-mc/we/session"
)

// WandCommand implements //wand — tags the held item (or a wood axe) as the selection wand.
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

// Pos1Command implements //pos1 — sets the first selection corner to the player's block position.
type Pos1Command struct{ playerCommand }

// Pos2Command implements //pos2 — sets the second selection corner.
type Pos2Command struct{ playerCommand }

func (Pos1Command) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	pos := cube.PosFromVec3(p.Position())
	s := session.Ensure(p)
	if s.SetPos1(pos) {
		selectionui.Trace(p, s)
		o.Printf("pos1 set to %v%s", pos, selectionui.SelectedBlocksSuffix(s))
		return
	}
	o.Printf("pos1 unchanged (%v)%s", pos, selectionui.SelectedBlocksSuffix(s))
}

func (Pos2Command) Run(src dcf.Source, o *dcf.Output, _ *world.Tx) {
	p := src.(*player.Player)
	pos := cube.PosFromVec3(p.Position())
	s := session.Ensure(p)
	if s.SetPos2(pos) {
		selectionui.Trace(p, s)
		o.Printf("pos2 set to %v%s", pos, selectionui.SelectedBlocksSuffix(s))
		return
	}
	o.Printf("pos2 unchanged (%v)%s", pos, selectionui.SelectedBlocksSuffix(s))
}
