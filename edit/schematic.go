package edit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/we/parse"
)

// SchematicDirectory is the on-disk folder for //schematic JSON files (overridable in tests).
var SchematicDirectory = ".we-schematics"

type schematicFile struct {
	OriginDir string           `json:"origin_dir"`
	Entries   []schematicEntry `json:"entries"`
}

type schematicEntry struct {
	Offset [3]int            `json:"offset"`
	Block  parse.SerialBlock `json:"block"`
	Liquid parse.SerialBlock `json:"liquid"`
}

var schematicNameRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func validateSchematicName(name string) error {
	if !schematicNameRE.MatchString(name) {
		return fmt.Errorf("invalid schematic name %q", name)
	}
	return nil
}

func schematicPath(name string) (string, error) {
	if err := validateSchematicName(name); err != nil {
		return "", err
	}
	return filepath.Join(SchematicDirectory, name+".json"), nil
}

func SaveSchematic(name string, cb *Clipboard) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("selection is empty")
	}
	path, err := schematicPath(name)
	if err != nil {
		return err
	}
	sf := schematicFile{OriginDir: cb.OriginDir.String()}
	for _, e := range cb.Entries {
		sf.Entries = append(sf.Entries, schematicEntry{
			Offset: [3]int{e.Offset[0], e.Offset[1], e.Offset[2]},
			Block:  parse.MarshalBlock(e.Block, true),
			Liquid: parse.MarshalBlock(e.Liquid, e.HasLiq),
		})
	}
	if err := os.MkdirAll(SchematicDirectory, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadSchematic(name string) (*Clipboard, error) {
	path, err := schematicPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sf schematicFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	dir := schematicOriginDir(sf.OriginDir)
	cb := &Clipboard{OriginDir: dir}
	for _, se := range sf.Entries {
		b, _, err := parse.UnmarshalBlock(se.Block)
		if err != nil {
			return nil, err
		}
		liqBlock, hasLiq, err := parse.UnmarshalBlock(se.Liquid)
		if err != nil {
			return nil, err
		}
		e := bufferEntry{Offset: cube.Pos{se.Offset[0], se.Offset[1], se.Offset[2]}, Block: b, HasLiq: hasLiq}
		if hasLiq {
			if l, ok := liqBlock.(world.Liquid); ok {
				e.Liquid = l
			} else {
				return nil, fmt.Errorf("schematic liquid at %v is not a liquid", e.Offset)
			}
		}
		cb.Entries = append(cb.Entries, e)
	}
	return cb, nil
}

func DeleteSchematic(name string) error {
	path, err := schematicPath(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func ListSchematics() ([]string, error) {
	entries, err := os.ReadDir(SchematicDirectory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		names = append(names, e.Name()[:len(e.Name())-len(".json")])
	}
	sort.Strings(names)
	return names, nil
}

func schematicOriginDir(s string) cube.Direction {
	switch s {
	case "east":
		return cube.East
	case "south":
		return cube.South
	case "west":
		return cube.West
	default:
		return cube.North
	}
}
