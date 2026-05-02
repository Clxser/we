package we

import (
	"image/color"
	"iter"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/we/act"
	_ "github.com/df-mc/we/cmd"
	"github.com/df-mc/we/editbrush"
	"github.com/df-mc/we/history"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/palette"
	"github.com/df-mc/we/service"
	"github.com/df-mc/we/session"
	"github.com/df-mc/we/visual"
	"github.com/go-gl/mathgl/mgl64"
)

// Handler is the main world-edit player handler.
type Handler struct {
	player.NopHandler
	p              *player.Player
	ph             *palette.Handler
	selectionTrace visual.Wireframe

	cfg Config
}

// NewHandler returns a player handler. WorldEdit commands register when the
// cmd package is imported (blank import keeps registration tied to using we).
func NewHandler(p *player.Player, opts ...Option) *Handler {
	cfg := newConfig(opts)
	session.EnsureWithSettings(p, cfg.HistoryLimit, cfg.SchematicStore, cfg.guardrails())
	return &Handler{p: p, ph: palette.NewHandler(p), cfg: cfg}
}

// HandleItemUse implements item use (brush raycast when bound).
func (h *Handler) HandleItemUse(ctx *player.Context) {
	if cfg, ok := h.heldBrush(); ok {
		ctx.Cancel()
		h.applyBrush(ctx.Val().Tx(), h.brushTarget(ctx.Val().Tx()), cfg)
		return
	}
}

// HandleItemUseOnBlock sets pos2 with the wand or applies a brush to a block face.
func (h *Handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, vec mgl64.Vec3) {
	if h.heldWand() {
		ctx.Cancel()
		s := session.Ensure(h.p)
		if s.SetPos2(pos) {
			h.p.Messagef("pos2 set to %v", pos)
		}
		h.traceSelection(s)
		return
	}
	if cfg, ok := h.heldBrush(); ok {
		ctx.Cancel()
		h.applyBrush(ctx.Val().Tx(), pos.Side(face), cfg)
		return
	}
	h.ph.HandleItemUseOnBlock(ctx, pos, face, vec)
}

// HandleBlockBreak sets pos1 when breaking with the wand.
func (h *Handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	if h.heldWand() {
		ctx.Cancel()
		s := session.Ensure(h.p)
		if s.SetPos1(pos) {
			h.p.Messagef("pos1 set to %v", pos)
		}
		h.traceSelection(s)
		return
	}
	h.ph.HandleBlockBreak(ctx, pos, drops, xp)
}

// HandleQuit cleans up session state.
func (h *Handler) HandleQuit(*player.Player) {
	h.ph.HandleQuit()
	h.selectionTrace.Remove(h.p)
	session.Delete(h.p)
}

func (h *Handler) traceSelection(s *session.Session) {
	area, ok := s.SelectionArea()
	if !ok {
		h.selectionTrace.Remove(h.p)
		return
	}
	h.selectionTrace.Draw(h.p, visual.BoxSegments(visual.AreaBox(area)), selectionTraceColour)
}

var selectionTraceColour = color.RGBA{R: 0, G: 255, B: 255, A: 255}

func (h *Handler) heldWand() bool {
	held, _ := h.p.HeldItems()
	_, ok := held.Value(keys.WandItemKey)
	return ok
}

func (h *Handler) heldBrush() (service.BrushConfig, bool) {
	held, _ := h.p.HeldItems()
	return editbrush.ConfigFromItem(held)
}

func (h *Handler) applyBrush(tx *world.Tx, target cube.Pos, cfg service.BrushConfig) {
	batch := history.NewBatch(true)
	if err := service.ApplyBrush(tx, service.BrushActor{Position: h.p.Position(), Rotation: h.p.Rotation()}, target, cfg, h.cfg.SchematicStore, h.cfg.guardrails(), batch); err != nil {
		h.p.Message(err.Error())
		return
	}
	session.Ensure(h.p).Record(batch)
}

var brushTraceBox = cube.Box(-0.125, -0.125, -0.125, 0.125, 0.125, 0.125)

func (h *Handler) brushTarget(tx *world.Tx) cube.Pos {
	start := h.p.Position().Add(mgl64.Vec3{0, h.p.EyeHeight()})
	end := start.Add(h.p.Rotation().Vec3().Mul(h.cfg.BrushMaxDistance))
	filter := func(seq iter.Seq[world.Entity]) iter.Seq[world.Entity] {
		return func(yield func(world.Entity) bool) {
			for e := range seq {
				if e == h.p {
					continue
				}
				if !yield(e) {
					return
				}
			}
		}
	}
	if res, ok := trace.Perform(start, end, tx, brushTraceBox, filter); ok {
		return cube.PosFromVec3(res.Position())
	}
	return cube.PosFromVec3(end)
}
