package we

import (
	"testing"

	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/session"
)

func TestDefaultConfigPreservesCurrentDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HistoryLimit != session.DefaultHistoryLimit {
		t.Fatalf("HistoryLimit = %d, want %d", cfg.HistoryLimit, session.DefaultHistoryLimit)
	}
	if cfg.SchematicDirectory != edit.DefaultSchematicDirectory {
		t.Fatalf("SchematicDirectory = %q, want %q", cfg.SchematicDirectory, edit.DefaultSchematicDirectory)
	}
	if dir := schematicStoreDir(t, cfg.SchematicStore); dir != edit.DefaultSchematicDirectory {
		t.Fatalf("SchematicStore dir = %q, want %q", dir, edit.DefaultSchematicDirectory)
	}
	if cfg.BrushMaxDistance != defaultBrushMaxDistance {
		t.Fatalf("BrushMaxDistance = %v, want %v", cfg.BrushMaxDistance, defaultBrushMaxDistance)
	}
	if cfg.MaxSelectionVolume != 0 || cfg.MaxShapeVolume != 0 || cfg.MaxBrushVolume != 0 || cfg.MaxStackCopies != 0 {
		t.Fatalf("guardrails should default to unlimited zero values: %+v", cfg)
	}
}

func TestOptionsOverrideConfig(t *testing.T) {
	cfg := newConfig([]Option{
		WithHistoryLimit(99),
		WithSchematicDirectory("schems"),
		WithBrushMaxDistance(64),
		WithMaxSelectionVolume(1),
		WithMaxShapeVolume(2),
		WithMaxBrushVolume(3),
		WithMaxStackCopies(4),
	})
	if cfg.HistoryLimit != 99 || cfg.SchematicDirectory != "schems" || cfg.BrushMaxDistance != 64 {
		t.Fatalf("options did not apply: %+v", cfg)
	}
	if dir := schematicStoreDir(t, cfg.SchematicStore); dir != "schems" {
		t.Fatalf("SchematicStore dir = %q, want schems", dir)
	}
	if cfg.MaxSelectionVolume != 1 || cfg.MaxShapeVolume != 2 || cfg.MaxBrushVolume != 3 || cfg.MaxStackCopies != 4 {
		t.Fatalf("guardrail options did not apply: %+v", cfg)
	}
}

func TestInvalidOptionsFallBackToDefaults(t *testing.T) {
	cfg := newConfig([]Option{
		WithHistoryLimit(0),
		WithSchematicDirectory(""),
		WithBrushMaxDistance(0),
	})
	if cfg.HistoryLimit != session.DefaultHistoryLimit {
		t.Fatalf("HistoryLimit = %d, want default", cfg.HistoryLimit)
	}
	if cfg.SchematicDirectory != edit.DefaultSchematicDirectory {
		t.Fatalf("SchematicDirectory = %q, want default", cfg.SchematicDirectory)
	}
	if dir := schematicStoreDir(t, cfg.SchematicStore); dir != edit.DefaultSchematicDirectory {
		t.Fatalf("SchematicStore dir = %q, want default", dir)
	}
	if cfg.BrushMaxDistance != defaultBrushMaxDistance {
		t.Fatalf("BrushMaxDistance = %v, want default", cfg.BrushMaxDistance)
	}
}

func TestWithSchematicStoreOverridesDirectoryStore(t *testing.T) {
	store := stubSchematicStore{}
	cfg := newConfig([]Option{
		WithSchematicDirectory("schems"),
		WithSchematicStore(store),
	})
	if cfg.SchematicStore != store {
		t.Fatal("custom schematic store was not preserved")
	}
}

type stubSchematicStore struct{}

func (stubSchematicStore) Save(string, *edit.Clipboard) error   { return nil }
func (stubSchematicStore) Load(string) (*edit.Clipboard, error) { return nil, nil }
func (stubSchematicStore) Delete(string) error                  { return nil }
func (stubSchematicStore) List() ([]string, error)              { return nil, nil }

func schematicStoreDir(t *testing.T, store edit.SchematicStore) string {
	t.Helper()
	fileStore, ok := store.(edit.FileSchematicStore)
	if !ok {
		t.Fatalf("SchematicStore type = %T, want edit.FileSchematicStore", store)
	}
	return fileStore.Dir
}
