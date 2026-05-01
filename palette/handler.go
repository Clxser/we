package palette

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/we/internal/msg"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

// handlers stores all Handler values for players currently online.
var handlers sync.Map

// LookupHandler finds the Handler of a specific player.Player, assuming it is currently online.
func LookupHandler(p *player.Player) (*Handler, bool) {
	v, _ := handlers.Load(p)
	h, ok := v.(*Handler)
	return h, ok
}

// Handler handles the selection and storage of palettes during the session of a player.
type Handler struct {
	p        *player.Player
	close    chan struct{}
	palettes sync.Map

	mu        sync.Mutex
	m         Selection
	selecting int
	first     cube.Pos
}

// NewHandler creates a Handler for the *player.Player passed.
func NewHandler(p *player.Player) *Handler {
	h := &Handler{p: p, close: make(chan struct{})}
	handlers.Store(p, h)
	return h
}

// Palette looks up the Palette with the name passed. If found, the Palette returned is non-nil and the bool true.
//
// If "m" or "M" is passed as Palette, the Palette will always be non-nil. Note that this Palette might still,
// however, be zero. This should be checked for using len(Palette.Blocks()).
func (h *Handler) Palette(name string) (Palette, bool) {
	if name == "m" || name == "M" {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.m, true
	}
	p, _ := h.palettes.Load(name)
	b, ok := p.(Blocks)
	return b, ok
}

// HandleItemUseOnBlock handles selection of a block for the palette.
func (h *Handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, _ cube.Face, _ mgl64.Vec3) {
	h.handleSelection(ctx, pos)
}

// HandleBlockBreak handles selection of a block for the palette.
func (h *Handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, _ *[]item.Stack, _ *int) {
	h.handleSelection(ctx, pos)
}

// HandleQuit deletes the Handler from the handlers map.
func (h *Handler) HandleQuit() {
	close(h.close)
	handlers.Delete(h.p)
}

// handleSelection handles the selection of a point for a palette. If no palette is currently being selected,
// handleSelection returns immediately. If the second point was selected, the palette is finalised and
// stored with the name "M".
func (h *Handler) handleSelection(ctx *player.Context, pos cube.Pos) {
	if h.selecting == 0 {
		// Not currently selecting, return immediately.
		return
	}
	ctx.Cancel()

	h.selecting--
	if h.selecting == 1 {
		// Selecting the first point: Store it in the handler and return.
		h.first = pos
		h.p.Message(fmt.Sprintf(msg.FirstPointSelected, pos))
		return
	}
	// First point was selected, we now have a second point so we can create a palette.
	h.p.Message(fmt.Sprintf(msg.SecondPointSelected, pos))
	h.m = NewSelection(h.first, pos, h.p.Tx().World())
	h.p.Message(text.Colourf("<green>"+msg.PaletteCreated+"</green>", h.m.Min, h.m.Max))
}
