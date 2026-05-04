package selectionui

import (
	"testing"

	"github.com/df-mc/we/geo"
)

type fakeSelection struct {
	area geo.Area
	ok   bool
}

func (s fakeSelection) SelectionArea() (geo.Area, bool) {
	return s.area, s.ok
}

func TestSelectedBlocksSuffixFormatsVolume(t *testing.T) {
	got := SelectedBlocksSuffix(fakeSelection{area: geo.NewArea(0, 0, 0, 99, 99, 99), ok: true})
	if got != " (1,000,000 blocks selected)" {
		t.Fatalf("suffix = %q, want formatted block count", got)
	}
}

func TestSelectedBlocksSuffixEmptyWithoutCompleteSelection(t *testing.T) {
	if got := SelectedBlocksSuffix(fakeSelection{}); got != "" {
		t.Fatalf("suffix = %q, want empty", got)
	}
}
