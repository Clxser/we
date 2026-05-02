package session

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/history"
	"github.com/google/uuid"
)

func TestReleaseKeepsClipboardDuringRetention(t *testing.T) {
	clearSessions(t)
	id := uuid.New()
	now := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	cb := &edit.Clipboard{OriginDir: cube.North}
	s := &Session{
		selection:    Selection{Pos1: cube.Pos{1, 2, 3}, Has1: true, Pos2: cube.Pos{4, 5, 6}, Has2: true},
		clipboard:    cb,
		history:      history.NewHistory(7),
		historyLimit: 7,
	}
	sessions.Store(id, s)

	releaseID(id, now)

	got, ok := lookupID(id, now.Add(ClipboardRetention-time.Nanosecond))
	if !ok {
		t.Fatal("session expired before clipboard retention elapsed")
	}
	if gotClipboard, ok := got.Clipboard(); !ok || gotClipboard != cb {
		t.Fatalf("clipboard = %v, %v; want retained clipboard", gotClipboard, ok)
	}
	if _, _, ok := got.PosCorners(); ok {
		t.Fatal("selection survived player release; want online-only state reset")
	}
}

func TestReleaseExpiresClipboardAfterRetention(t *testing.T) {
	clearSessions(t)
	id := uuid.New()
	now := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	sessions.Store(id, &Session{clipboard: &edit.Clipboard{OriginDir: cube.North}, history: history.NewHistory(1), historyLimit: 1})

	releaseID(id, now)
	if _, ok := lookupID(id, now.Add(ClipboardRetention)); ok {
		t.Fatal("session survived after clipboard retention elapsed")
	}
}

func TestReleaseDropsSessionWithoutClipboard(t *testing.T) {
	clearSessions(t)
	id := uuid.New()
	now := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	sessions.Store(id, &Session{history: history.NewHistory(1), historyLimit: 1})

	releaseID(id, now)
	if _, ok := lookupID(id, now); ok {
		t.Fatal("empty session survived player release")
	}
}

func clearSessions(t *testing.T) {
	t.Helper()
	sessions.Range(func(key, _ any) bool {
		sessions.Delete(key)
		return true
	})
}
