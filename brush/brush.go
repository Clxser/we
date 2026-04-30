package brush

import (
	"iter"
	"reflect"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

var brushes sync.Map

func Lookup(id uuid.UUID) (*Brush, bool) {
	v, _ := brushes.Load(id)
	h, ok := v.(*Brush)
	return h, ok
}

type Brush struct {
	s  Shape
	a  Action
	id uuid.UUID
}

func New(s Shape, a Action) Brush {
	b := Brush{s: s, a: a, id: uuid.New()}
	brushes.Store(b.id, b)
	return b
}

func (b Brush) UUID() uuid.UUID {
	return b.id
}

var bb = cube.Box(-0.125, -0.125, -0.125, 0.125, 0.125, 0.125)

func (b Brush) Use(p *player.Player, tx *world.Tx) {
	const (
		maxDistance  = 128
		maxUndoCount = 40
	)
	vec := p.Rotation().Vec3().Mul(maxDistance)
	pos := p.Position().Add(mgl64.Vec3{0, p.EyeHeight()})

	final := pos.Add(vec)
	if res, ok := trace.Perform(pos, final, tx, bb, withoutPlayer(p)); ok {
		final = res.Position()
	}

	h, _ := LookupHandler(p)
	revert := Perform(cube.PosFromVec3(final), b.s, b.a, tx)
	if len(h.undo) == maxUndoCount {
		h.undo = append(h.undo[1:], revert)
		return
	}
	h.undo = append(h.undo, revert)
}

func withoutPlayer(p *player.Player) trace.EntityFilter {
	return func(seq iter.Seq[world.Entity]) iter.Seq[world.Entity] {
		return func(yield func(world.Entity) bool) {
			for e := range seq {
				if e == p {
					continue
				}
				if !yield(e) {
					return
				}
			}
		}
	}
}

// Bind binds the Brush to the item.Stack i passed and returns a new item.Stack with the Brush bound to it.
func (b Brush) Bind(i item.Stack) item.Stack {
	return i.WithValue("brush", b.UUID().String()).
		WithCustomName(text.Colourf("<white>%v (%v) %v Brush</white>\n<green>[Use]</green>", reflect.ValueOf(b.s).Type().Name(), b.s.Dim()[0]/2, reflect.ValueOf(b.a).Type().Name()))
}

// Unbind unbinds any Brush bound to the item.Stack passed and returns an unbound version of the stack.
func Unbind(i item.Stack) item.Stack {
	return i.WithValue("brush", nil).WithCustomName("")
}

// find looks for a Brush bound to the item.Stack passed and returns it if one was found.
func find(i item.Stack) (Brush, bool) {
	if id, ok := i.Value("brush"); ok {
		if b, ok := brushes.Load(uuid.MustParse(id.(string))); ok {
			return b.(Brush), true
		}
	}
	return Brush{}, false
}
