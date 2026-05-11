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

// DefaultSchematicDirectory is the default on-disk folder for //schematic JSON files.
const DefaultSchematicDirectory = ".we-schematics"

// SchematicStore persists clipboard schematics by name.
type SchematicStore interface {
	Save(name string, cb *Clipboard) error
	Load(name string) (*Clipboard, error)
	Delete(name string) error
	List() ([]string, error)
}

// FileSchematicStore stores schematic JSON files in a directory.
type FileSchematicStore struct {
	Dir string
}

// NewFileSchematicStore returns a filesystem-backed schematic store. An empty
// dir keeps DefaultSchematicDirectory.
func NewFileSchematicStore(dir string) FileSchematicStore {
	if dir == "" {
		dir = DefaultSchematicDirectory
	}
	return FileSchematicStore{Dir: dir}
}

// DefaultSchematicStore returns the behavior-preserving filesystem schematic store.
func DefaultSchematicStore() SchematicStore {
	return NewFileSchematicStore(DefaultSchematicDirectory)
}

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

func (s FileSchematicStore) dir() string {
	if s.Dir == "" {
		return DefaultSchematicDirectory
	}
	return s.Dir
}

func (s FileSchematicStore) path(name string) (string, error) {
	if err := validateSchematicName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.dir(), name+".json"), nil
}

// Save writes cb to disk under name. Names are restricted to [A-Za-z0-9_.-].
func (s FileSchematicStore) Save(name string, cb *Clipboard) error {
	if cb == nil || len(cb.Entries) == 0 {
		return fmt.Errorf("selection is empty")
	}
	path, err := s.path(name)
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
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(sf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads a previously saved schematic into a Clipboard.
//
// Java-edition schematics (.schem Sponge v2, .schematic legacy MCEdit) are
// preferred over the native .json format when both exist with the same
// name — they're the format players are likely to upload. Translation of
// Java block-states to Bedrock is delegated to the github.com/Velvet-MC/s2d
// library; the resulting unknown-block report is silently discarded by
// this method (callers that want it should call ImportJavaSchematic
// directly).
func (s FileSchematicStore) Load(name string) (*Clipboard, error) {
	if err := validateSchematicName(name); err != nil {
		return nil, err
	}
	for _, ext := range []string{".schem", ".schematic"} {
		p := filepath.Join(s.dir(), name+ext)
		if _, statErr := os.Stat(p); statErr == nil {
			cb, _, err := ImportJavaSchematic(p)
			return cb, err
		}
	}
	path, err := s.path(name)
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

// Delete removes a saved schematic file.
func (s FileSchematicStore) Delete(name string) error {
	path, err := s.path(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// List returns saved schematic names in alphabetical order.
func (s FileSchematicStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir())
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
