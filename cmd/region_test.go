package cmd

import (
	"testing"
)

func TestParseSetArgsNoUndoFlag(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantSpec   string
		wantNoUndo bool
	}{
		{name: "suffix short flag", raw: "stone -noundo", wantSpec: "stone", wantNoUndo: true},
		{name: "prefix short flag", raw: "-noundo stone", wantSpec: "stone", wantNoUndo: true},
		{name: "long flag", raw: "--no-undo stone", wantSpec: "stone", wantNoUndo: true},
		{name: "default", raw: "stone", wantSpec: "stone"},
		{name: "multi block", raw: "-noundo stone,dirt", wantSpec: "stone,dirt", wantNoUndo: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSpec, gotOpts := parseSetArgs(tt.raw)
			if gotSpec != tt.wantSpec {
				t.Fatalf("spec = %q, want %q", gotSpec, tt.wantSpec)
			}
			if gotOpts.NoUndo != tt.wantNoUndo {
				t.Fatalf("NoUndo = %v, want %v", gotOpts.NoUndo, tt.wantNoUndo)
			}
		})
	}
}
