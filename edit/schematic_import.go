package edit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"

	_ "github.com/Velvet-MC/s2d/legacy" // register .schematic handler
	_ "github.com/Velvet-MC/s2d/sponge" // register .schem handler

	"github.com/Velvet-MC/s2d/schem"
)

// JavaSchematicReport is the per-load summary returned by ImportJavaSchematic.
// It mirrors s2d.UnknownReport so the service/cmd layers can surface
// unknown-block tallies to the player without importing s2d directly.
type JavaSchematicReport struct {
	Format string         // "sponge_v2" or "legacy_schematic"
	Counts map[string]int // canonical Java state string → count
	Total  int            // total cells that fell back to the missing block
	Width  int
	Height int
	Length int
}

// ImportJavaSchematic reads a Sponge v2 (.schem) or legacy MCEdit (.schematic)
// file and returns a Clipboard populated with translated Bedrock blocks.
//
// The clipboard's OriginDir is set to cube.North so paste rotation behaves
// predictably — Java schematics carry no facing of their own. Players use
// //rotate after //paste to orient the result.
//
// Unknown Java blocks are filled with the missing-block fallback configured
// via s2d/translate.SetMissingBlock (default magenta wool) and tallied in
// the returned JavaSchematicReport.
//
// The clipboard's Entries slice is laid out in XYZ index order
// ((x*height+y)*length+z) so the downstream paste fast-path
// (edit.makeDenseBuffer) recognises it as already-ordered and skips a
// redundant ~per-cell reorder allocation. For arena-scale schematics
// (10M+ cells) this saves several GB of transient memory.
func ImportJavaSchematic(path string) (*Clipboard, JavaSchematicReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, JavaSchematicReport{}, err
	}
	defer f.Close()

	tRead := startTrace("import.schem.Read")
	s, err := schem.Read(filepath.Base(path), f)
	tRead.end()
	if err != nil {
		return nil, JavaSchematicReport{}, fmt.Errorf("import %s: %w", filepath.Base(path), err)
	}
	traceAnnotate("import.schem.Read result",
		"width", s.Width, "height", s.Height, "length", s.Length,
		"cells", len(s.Blocks),
		"unknown_kinds", len(s.Unknowns.Counts),
		"unknown_cells", s.Unknowns.Total,
	)

	tAlloc := startTrace("import.cb.Entries.make")
	h, l := s.Height, s.Length
	cb := &Clipboard{
		OriginDir: cube.North,
		Entries:   make([]bufferEntry, len(s.Blocks)),
	}
	tAlloc.end()

	tFill := startTrace("import.cb.Entries.fill")
	for i := range s.Blocks {
		b := &s.Blocks[i]
		idx := (b.Pos[0]*h+b.Pos[1])*l + b.Pos[2]
		e := &cb.Entries[idx]
		e.Offset = cube.Pos{b.Pos[0], b.Pos[1], b.Pos[2]}
		e.Block = b.Block
		if b.Liquid != nil {
			if liq, ok := b.Liquid.(world.Liquid); ok {
				e.Liquid = liq
				e.HasLiq = true
			}
		}
	}
	tFill.end()

	rep := JavaSchematicReport{
		Format: string(s.Format),
		Counts: s.Unknowns.Counts,
		Total:  s.Unknowns.Total,
		Width:  s.Width,
		Height: s.Height,
		Length: s.Length,
	}
	// Drop the s.Blocks reference so the GC can reclaim ~5 GB on arena-scale
	// imports; the data has been copied into cb.Entries.
	tDrop := startTrace("import.s.Blocks=nil+GC")
	s.Blocks = nil
	runtime.GC()
	tDrop.end()
	// Return the freed memory to the OS so subsequent allocations (e.g. the
	// paste path) don't have to ask the kernel for new pages on top of the
	// runtime's now-idle heap. Without this, Sys stays near the peak even
	// after heap drops, and the next big alloc can OOM-kill the process.
	tFree := startTrace("import.debug.FreeOSMemory")
	debug.FreeOSMemory()
	tFree.end()

	return cb, rep, nil
}
