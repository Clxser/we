package edit

import (
	"fmt"
	"os"
	"path/filepath"

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
	Format   string         // "sponge_v2" or "legacy_schematic"
	Counts   map[string]int // canonical Java state string → count
	Total    int            // total cells that fell back to the missing block
	Width    int
	Height   int
	Length   int
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
func ImportJavaSchematic(path string) (*Clipboard, JavaSchematicReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, JavaSchematicReport{}, err
	}
	defer f.Close()

	s, err := schem.Read(filepath.Base(path), f)
	if err != nil {
		return nil, JavaSchematicReport{}, fmt.Errorf("import %s: %w", filepath.Base(path), err)
	}

	cb := &Clipboard{OriginDir: cube.North}
	for _, b := range s.Blocks {
		e := bufferEntry{
			Offset: cube.Pos{b.Pos[0], b.Pos[1], b.Pos[2]},
			Block:  b.Block,
		}
		if b.Liquid != nil {
			if liq, ok := b.Liquid.(world.Liquid); ok {
				e.Liquid = liq
				e.HasLiq = true
			}
		}
		cb.Entries = append(cb.Entries, e)
	}

	rep := JavaSchematicReport{
		Format: string(s.Format),
		Counts: s.Unknowns.Counts,
		Total:  s.Unknowns.Total,
		Width:  s.Width,
		Height: s.Height,
		Length: s.Length,
	}
	return cb, rep, nil
}
