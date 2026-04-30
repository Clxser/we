package session_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/we/session"
)

func TestSelectionRequiresBothCorners(t *testing.T) {
	var s session.Selection
	if _, ok := s.Area(); ok {
		t.Fatal("zero selection reported valid")
	}
	s.Pos1, s.Has1 = cube.Pos{3, 0, 0}, true
	if _, ok := s.Area(); ok {
		t.Fatal("one-corner selection reported valid")
	}
	s.Pos2, s.Has2 = cube.Pos{1, 2, 0}, true
	area, ok := s.Area()
	if !ok {
		t.Fatal("two-corner selection reported invalid")
	}
	if area.Min != (cube.Pos{1, 0, 0}) || area.Max != (cube.Pos{3, 2, 0}) {
		t.Fatalf("selection area = %v-%v", area.Min, area.Max)
	}
}
