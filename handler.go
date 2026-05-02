package we

import (
	"image/color"
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/we/cmd"
	"github.com/df-mc/we/editbrush"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/service"
	"github.com/df-mc/we/session"
	"github.com/df-mc/we/visual"
	"github.com/go-gl/mathgl/mgl64"
)

// Handler is the main world-edit player handler.
type Handler struct {
	player.NopHandler
	p              *player.Player
	selectionTrace visual.Wireframe
	brushTrace     visual.Wireframe

	cfg Config
}

// NewHandler returns a player handler. WorldEdit commands register when the
// cmd package is imported (blank import keeps registration tied to using we).
func NewHandler(p *player.Player, opts ...Option) *Handler {
	cfg := newConfig(opts)
	session.EnsureWithSettings(p, cfg.HistoryLimit, cfg.SchematicStore, cfg.guardrails())
	return &Handler{p: p, cfg: cfg}
}

// HandleItemUse implements item use (brush raycast when bound).
func (h *Handler) HandleItemUse(ctx *player.Context) {
	if cfg, ok := h.heldBrush(); ok {
		ctx.Cancel()
		start := h.brushRayStart()
		target := h.brushTarget(ctx.Val().Tx(), cfg, start)
		if h.applyBrush(ctx.Val().Tx(), target, cfg) {
			h.traceBrush(start, target, cfg)
		} else {
			h.brushTrace.Remove(h.p)
		}
		return
	}
}

// HandleItemUseOnBlock sets pos2 with the wand or applies a brush to the looked-at block.
func (h *Handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, _ mgl64.Vec3) {
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
		target := service.BrushAnchorFromSurface(pos.Side(face), face, cfg)
		if h.applyBrush(ctx.Val().Tx(), target, cfg) {
			h.traceBrush(h.brushRayStart(), target, cfg)
		} else {
			h.brushTrace.Remove(h.p)
		}
		return
	}
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
}

// HandleQuit releases online-only session state while allowing clipboard
// retention for reconnects during the same server lifetime.
func (h *Handler) HandleQuit(*player.Player) {
	h.selectionTrace.Remove(h.p)
	h.brushTrace.Remove(h.p)
	session.Delete(h.p)
}

func (h *Handler) traceSelection(s *session.Session) {
	area, ok := s.SelectionArea()
	if !ok {
		h.selectionTrace.Remove(h.p)
		return
	}
	h.selectionTrace.Draw(h.p, visual.AreaSegments(area), selectionTraceColour)
}

var selectionTraceColour = color.RGBA{R: 0, G: 255, B: 255, A: 255}
var brushTraceColour = color.RGBA{R: 255, G: 180, B: 0, A: 255}

func (h *Handler) heldWand() bool {
	held, _ := h.p.HeldItems()
	_, ok := held.Value(keys.WandItemKey)
	return ok
}

func (h *Handler) heldBrush() (service.BrushConfig, bool) {
	held, _ := h.p.HeldItems()
	return editbrush.ConfigFromItem(held)
}

func (h *Handler) applyBrush(tx *world.Tx, target cube.Pos, cfg service.BrushConfig) bool {
	actor := service.BrushActor{Position: h.p.Position(), Rotation: h.p.Rotation()}
	if err := service.ApplyBrushAndRecord(tx, session.Ensure(h.p), actor, target, cfg, h.cfg.SchematicStore, h.cfg.guardrails()); err != nil {
		h.p.Message(err.Error())
		return false
	}
	return true
}

const brushRaySelfSkipDistance = 1.0

func (h *Handler) brushRayStart() mgl64.Vec3 {
	return h.p.Position().Add(mgl64.Vec3{0, h.p.EyeHeight()})
}

func (h *Handler) traceBrush(start mgl64.Vec3, target cube.Pos, cfg service.BrushConfig) {
	segments := []visual.Segment{visual.LineSegment(start, target.Vec3Centre())}
	if area, ok := service.BrushVolumeBounds(target, cfg); ok {
		segments = append(segments, visual.AreaSegments(area)...)
	} else {
		segments = append(segments, visual.BlockSegments(target, target)...)
	}
	h.brushTrace.Draw(h.p, segments, brushTraceColour)
}

func (h *Handler) brushTarget(tx *world.Tx, cfg service.BrushConfig, start mgl64.Vec3) cube.Pos {
	dir := h.p.Rotation().Vec3()
	end := start.Add(dir.Mul(h.cfg.BrushMaxDistance))
	if pos, face, ok := traceBrushBlock(start, end, tx, brushRaySelfSkipDistance); ok {
		return service.BrushAnchorFromSurface(pos.Side(face), face, cfg)
	}
	surface := cube.PosFromVec3(start.Add(dir.Mul(brushAirDistance(cfg, h.cfg.BrushMaxDistance))))
	return service.BrushAnchorFromSurface(surface, dominantFace(dir), cfg)
}

func traceBrushBlock(start, end mgl64.Vec3, tx *world.Tx, skipDistance float64) (cube.Pos, cube.Face, bool) {
	var (
		hitPos  cube.Pos
		hitFace cube.Face
		hit     bool
	)
	skipDistanceSqr := skipDistance * skipDistance
	trace.TraverseBlocks(start, end, func(pos cube.Pos) bool {
		res, ok := trace.BlockIntercept(pos, tx, tx.Block(pos), start, end)
		if !ok {
			return true
		}
		if res.Position().Sub(start).LenSqr() < skipDistanceSqr {
			return true
		}
		hitPos, hitFace, hit = res.BlockPosition(), res.Face(), true
		return false
	})
	return hitPos, hitFace, hit
}

func brushAirDistance(cfg service.BrushConfig, maxDistance float64) float64 {
	if cfg.Range <= 0 {
		return maxDistance
	}
	return min(float64(cfg.Range), maxDistance)
}

func dominantFace(dir mgl64.Vec3) cube.Face {
	x, y, z := dir[0], dir[1], dir[2]
	ax, ay, az := math.Abs(x), math.Abs(y), math.Abs(z)
	switch {
	case ay >= ax && ay >= az:
		if y < 0 {
			return cube.FaceDown
		}
		return cube.FaceUp
	case ax >= az:
		if x < 0 {
			return cube.FaceWest
		}
		return cube.FaceEast
	default:
		if z < 0 {
			return cube.FaceNorth
		}
		return cube.FaceSouth
	}
}
