package edit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/df-mc/dragonfly/server/block/cube"

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
// The clipboard's Entries slice is filled directly from s2d's streaming scan
// and laid out in XYZ index order ((x*height+y)*length+z), so the downstream
// paste fast-path (edit.makeDenseBuffer) recognises it as already-ordered and
// skips a redundant ~per-cell reorder allocation. For arena-scale schematics
// (10M+ cells), streaming also avoids materialising s2d's per-cell
// Schematic.Blocks slice before copying into the clipboard.
func ImportJavaSchematic(path string) (*Clipboard, JavaSchematicReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, JavaSchematicReport{}, err
	}
	defer func() { _ = f.Close() }()

	var cb *Clipboard
	var h, l int
	tRead := startTrace("import.schem.Scan")
	info, err := schem.ScanWithInfo(filepath.Base(path), f, func(info schem.ScanInfo) error {
		h, l = info.Height, info.Length
		cb = &Clipboard{
			OriginDir: cube.North,
			Entries:   make([]bufferEntry, info.Width*info.Height*info.Length),
		}
		return nil
	}, func(b schem.Block) error {
		if cb == nil {
			return fmt.Errorf("schematic metadata was not read before block data")
		}
		idx := (b.Pos[0]*h+b.Pos[1])*l + b.Pos[2]
		e := &cb.Entries[idx]
		e.Offset = cube.Pos{b.Pos[0], b.Pos[1], b.Pos[2]}
		e.Block = b.Block
		if b.Liquid != nil {
			e.Liquid = b.Liquid
			e.HasLiq = true
		}
		return nil
	})
	tRead.end()
	if err != nil {
		return nil, JavaSchematicReport{}, fmt.Errorf("import %s: %w", filepath.Base(path), err)
	}
	traceAnnotate("import.schem.Scan result",
		"width", info.Width, "height", info.Height, "length", info.Length,
		"cells", len(cb.Entries),
		"unknown_kinds", len(info.Unknowns.Counts),
		"unknown_cells", info.Unknowns.Total,
	)

	rep := JavaSchematicReport{
		Format: string(info.Format),
		Counts: info.Unknowns.Counts,
		Total:  info.Unknowns.Total,
		Width:  info.Width,
		Height: info.Height,
		Length: info.Length,
	}
	// Drop transient NBT decode buffers before paste starts.
	tDrop := startTrace("import.scan.GC")
	runtime.GC()
	tDrop.end()
	// Return freed decode memory to the OS so subsequent allocations (e.g. the
	// paste path) don't have to ask the kernel for new pages on top of the
	// runtime's now-idle heap.
	tFree := startTrace("import.debug.FreeOSMemory")
	debug.FreeOSMemory()
	tFree.end()

	return cb, rep, nil
}

// ImportJavaCompactSchematic reads a Sponge v2 (.schem) or legacy MCEdit
// (.schematic) file into a compact palette-backed dense schematic.
func ImportJavaCompactSchematic(path string) (*CompactSchematic, JavaSchematicReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, JavaSchematicReport{}, err
	}
	defer func() { _ = f.Close() }()

	var compact *CompactSchematic
	var sourcePalette []uint32
	var sourcePaletteSeen []bool
	tRead := startTrace("import.compact.schem.Scan")
	info, err := schem.ScanWithInfo(filepath.Base(path), f, func(info schem.ScanInfo) error {
		var err error
		compact, err = NewCompactSchematic(info.Width, info.Height, info.Length)
		if info.PaletteSize > 0 {
			sourcePalette = make([]uint32, info.PaletteSize)
			sourcePaletteSeen = make([]bool, info.PaletteSize)
		}
		return err
	}, func(b schem.Block) error {
		if compact == nil {
			return fmt.Errorf("schematic metadata was not read before block data")
		}
		if sourceIndex := int(b.PaletteIndex); b.PaletteIndexOK && sourceIndex < len(sourcePalette) {
			if !sourcePaletteSeen[sourceIndex] {
				sourcePalette[sourceIndex] = compact.AppendPaletteState(b.Block, b.Liquid)
				sourcePaletteSeen[sourceIndex] = true
			}
			compact.setPaletteIndexXYZ(b.Pos[0], b.Pos[1], b.Pos[2], sourcePalette[sourceIndex])
			return nil
		}
		return compact.AddBlock(cube.Pos{b.Pos[0], b.Pos[1], b.Pos[2]}, b.Block, b.Liquid)
	})
	tRead.end()
	if err != nil {
		return nil, JavaSchematicReport{}, fmt.Errorf("import compact %s: %w", filepath.Base(path), err)
	}
	traceAnnotate("import.compact.schem.Scan result",
		"width", info.Width, "height", info.Height, "length", info.Length,
		"cells", compact.Volume(),
		"palette", compact.PaletteLen(),
		"unknown_kinds", len(info.Unknowns.Counts),
		"unknown_cells", info.Unknowns.Total,
	)
	rep := JavaSchematicReport{
		Format: string(info.Format),
		Counts: info.Unknowns.Counts,
		Total:  info.Unknowns.Total,
		Width:  info.Width,
		Height: info.Height,
		Length: info.Length,
	}
	return compact, rep, nil
}
