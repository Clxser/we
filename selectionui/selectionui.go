// Package selectionui owns process-wide player-facing selection feedback
// adapters. Outline state is keyed by player UUID for the single Dragonfly
// server hosted by this process.
package selectionui

import (
	"image/color"
	"strconv"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/we/geo"
	"github.com/df-mc/we/visual"
)

var selectionColour = color.RGBA{R: 0, G: 255, B: 255, A: 255}

type selectionArea interface {
	SelectionArea() (geo.Area, bool)
}

type outline struct {
	wireframe visual.Wireframe
}

var outlines sync.Map

// Trace draws or updates p's current selection outline. If the selection is
// incomplete, any existing outline is removed.
func Trace(p *player.Player, s selectionArea) {
	area, ok := s.SelectionArea()
	if !ok {
		Remove(p)
		return
	}
	frame := outlineFor(p)
	frame.wireframe.Draw(p, visual.AreaSegments(area), selectionColour)
}

// Remove clears p's selection outline.
func Remove(p *player.Player) {
	v, ok := outlines.LoadAndDelete(p.UUID())
	if !ok {
		return
	}
	v.(*outline).wireframe.Remove(p)
}

// SelectedBlocksSuffix returns a WorldEdit-style selected block count suffix
// for status messages. Incomplete selections return an empty suffix.
func SelectedBlocksSuffix(s selectionArea) string {
	area, ok := s.SelectionArea()
	if !ok {
		return ""
	}
	return " (" + formatInt(area.Volume()) + " blocks selected)"
}

func outlineFor(p *player.Player) *outline {
	v, _ := outlines.LoadOrStore(p.UUID(), &outline{})
	return v.(*outline)
}

func formatInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	first := len(s) % 3
	if first == 0 {
		first = 3
	}
	out := make([]byte, 0, len(s)+(len(s)-1)/3)
	out = append(out, s[:first]...)
	for i := first; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}
