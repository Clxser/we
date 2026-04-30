package session

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/history"
)

const defaultHistoryLimit = 40

// Session contains per-player world-edit state.
type Session struct {
	p *player.Player

	mu        sync.Mutex
	selection Selection
	clipboard *edit.Clipboard
	history   *history.History
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

// Lookup returns the session for a player if present.
func Lookup(p *player.Player) (*Session, bool) {
	v, _ := sessions.Load(p)
	s, ok := v.(*Session)
	return s, ok
}

// Ensure returns the session for p, creating one if needed.
func Ensure(p *player.Player) *Session {
	if s, ok := Lookup(p); ok {
		return s
	}
	s := &Session{p: p, history: history.NewHistory(defaultHistoryLimit)}
	sessions.Store(p, s)
	return s
}

// Delete removes state when the player leaves.
func Delete(p *player.Player) {
	sessions.Delete(p)
}

// SetPos1 sets position 1.
func (s *Session) SetPos1(pos cube.Pos) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selection.Pos1, s.selection.Has1 = pos, true
}

// SetPos2 sets position 2.
func (s *Session) SetPos2(pos cube.Pos) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selection.Pos2, s.selection.Has2 = pos, true
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
