package session

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
	"github.com/google/uuid"
)

const DefaultHistoryLimit = 40

// Session contains per-player world-edit state.
type Session struct {
	p *player.Player

	mu         sync.Mutex
	selection  Selection
	clipboard  *edit.Clipboard
	schematics edit.SchematicStore
	history    *history.History
}

// Selection is the cuboid corners before normalisation.
type Selection struct {
	Pos1, Pos2 cube.Pos
	Has1, Has2 bool
}

// Area returns the normalised inclusive cuboid when both corners exist.
func (s Selection) Area() (geo.Area, bool) {
	if !s.Has1 || !s.Has2 {
		return geo.Area{}, false
	}
	return geo.NewArea(s.Pos1[0], s.Pos1[1], s.Pos1[2], s.Pos2[0], s.Pos2[1], s.Pos2[2]), true
}

var sessions sync.Map

func key(p *player.Player) uuid.UUID {
	return p.UUID()
}

// Lookup returns the session for a player if present.
func Lookup(p *player.Player) (*Session, bool) {
	v, _ := sessions.Load(key(p))
	s, ok := v.(*Session)
	return s, ok
}

// Ensure returns the session for p, creating one with the default history limit if needed.
func Ensure(p *player.Player) *Session {
	return EnsureWithHistoryLimit(p, DefaultHistoryLimit)
}

// EnsureWithHistoryLimit returns the session for p, creating one with the
// history limit passed if needed. Existing sessions keep their current history.
func EnsureWithHistoryLimit(p *player.Player, historyLimit int) *Session {
	if s, ok := Lookup(p); ok {
		return s
	}
	return EnsureWithSettings(p, historyLimit, edit.DefaultSchematicStore())
}

// EnsureWithSettings returns the session for p, creating one with the passed
// settings if needed. Existing sessions keep their current history and receive
// the latest non-nil schematic store.
func EnsureWithSettings(p *player.Player, historyLimit int, schematics edit.SchematicStore) *Session {
	if schematics == nil {
		schematics = edit.DefaultSchematicStore()
	}
	if s, ok := Lookup(p); ok {
		s.SetSchematicStore(schematics)
		return s
	}
	s := &Session{p: p, schematics: schematics, history: history.NewHistory(historyLimit)}
	sessions.Store(key(p), s)
	return s
}

// Delete removes state when the player leaves.
func Delete(p *player.Player) {
	sessions.Delete(key(p))
}

// SetPos1 sets position 1.
func (s *Session) SetPos1(pos cube.Pos) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selection.Has1 && s.selection.Pos1 == pos {
		return false
	}
	s.selection.Pos1, s.selection.Has1 = pos, true
	return true
}

// SetPos2 sets position 2.
func (s *Session) SetPos2(pos cube.Pos) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selection.Has2 && s.selection.Pos2 == pos {
		return false
	}
	s.selection.Pos2, s.selection.Has2 = pos, true
	return true
}

// SelectionArea returns the current cuboid if valid.
func (s *Session) SelectionArea() (geo.Area, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selection.Area()
}

// SetClipboard stores the clipboard buffer.
func (s *Session) SetClipboard(c *edit.Clipboard) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clipboard = c
}

// Clipboard returns the stored clipboard if any.
func (s *Session) Clipboard() (*edit.Clipboard, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clipboard, s.clipboard != nil
}

// SetSchematicStore sets the store used by schematic commands and schematic brushes.
func (s *Session) SetSchematicStore(store edit.SchematicStore) {
	if store == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schematics = store
}

// SchematicStore returns the configured schematic store, falling back to the
// default filesystem store when a session was built without one.
func (s *Session) SchematicStore() edit.SchematicStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.schematics == nil {
		return edit.DefaultSchematicStore()
	}
	return s.schematics
}

// PosCorners returns pos1 and pos2 when both are set (for //line).
func (s *Session) PosCorners() (pos1, pos2 cube.Pos, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.selection.Has1 || !s.selection.Has2 {
		return cube.Pos{}, cube.Pos{}, false
	}
	return s.selection.Pos1, s.selection.Pos2, true
}

// Record adds an undo batch (commands vs brush split inside history).
func (s *Session) Record(batch *history.Batch) int {
	return s.history.Record(batch)
}

// Undo runs undo; brush selects the brush-only stack.
func (s *Session) Undo(tx *world.Tx, brush bool) bool {
	return s.history.Undo(tx, brush)
}

// Redo runs redo.
func (s *Session) Redo(tx *world.Tx, brush bool) bool {
	return s.history.Redo(tx, brush)
}
