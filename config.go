package we

import (
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/guardrail"
	"github.com/df-mc/we/session"
)

const defaultBrushMaxDistance = 128

// Config controls WorldEdit library behavior. Zero-valued guardrail fields are
// unlimited.
type Config struct {
	// HistoryLimit caps the number of undo/redo batches kept per player stack.
	HistoryLimit int
	// SchematicDirectory is the filesystem directory used by the default JSON
	// schematic store.
	SchematicDirectory string
	// SchematicStore persists named clipboard schematics. If nil, a filesystem
	// store rooted at SchematicDirectory is used.
	SchematicStore edit.SchematicStore
	// BrushMaxDistance is the maximum raycast distance for item-bound editbrush
	// use through Handler.
	BrushMaxDistance float64

	// Guardrails. A value of 0 means unlimited.
	MaxSelectionVolume int
	MaxShapeVolume     int
	MaxBrushVolume     int
	MaxStackCopies     int
	// MaxEditSubChunks caps how many unique 16x16x16 sub-chunks a single edit
	// may touch. Use this to keep large edits below Dragonfly's pending
	// client-cache blob queue. A value of 0 means unlimited.
	MaxEditSubChunks int
}

// Option customises Config for a Handler.
type Option func(*Config)

// DefaultConfig returns behavior-preserving defaults.
func DefaultConfig() Config {
	dir := edit.DefaultSchematicDirectory
	return Config{
		HistoryLimit:       session.DefaultHistoryLimit,
		SchematicDirectory: dir,
		SchematicStore:     edit.NewFileSchematicStore(dir),
		BrushMaxDistance:   defaultBrushMaxDistance,
	}
}

func newConfig(opts []Option) Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = session.DefaultHistoryLimit
	}
	if cfg.SchematicDirectory == "" {
		cfg.SchematicDirectory = edit.DefaultSchematicDirectory
	}
	if cfg.SchematicStore == nil {
		cfg.SchematicStore = edit.NewFileSchematicStore(cfg.SchematicDirectory)
	}
	if cfg.BrushMaxDistance <= 0 {
		cfg.BrushMaxDistance = defaultBrushMaxDistance
	}
	return cfg
}

func (c Config) guardrails() guardrail.Limits {
	return guardrail.Limits{
		MaxSelectionVolume: c.MaxSelectionVolume,
		MaxShapeVolume:     c.MaxShapeVolume,
		MaxBrushVolume:     c.MaxBrushVolume,
		MaxStackCopies:     c.MaxStackCopies,
		MaxEditSubChunks:   c.MaxEditSubChunks,
	}
}

// WithHistoryLimit sets the undo/redo stack cap for sessions created by the
// Handler. Values <= 0 keep the default.
func WithHistoryLimit(limit int) Option {
	return func(c *Config) { c.HistoryLimit = limit }
}

// WithSchematicDirectory sets the directory used by the default schematic disk
// store. An empty path keeps the default.
func WithSchematicDirectory(dir string) Option {
	return func(c *Config) {
		c.SchematicDirectory = dir
		if dir != "" {
			c.SchematicStore = edit.NewFileSchematicStore(dir)
		}
	}
}

// WithSchematicStore sets the store used by schematic commands and schematic
// brushes. A nil store keeps the default filesystem store.
func WithSchematicStore(store edit.SchematicStore) Option {
	return func(c *Config) { c.SchematicStore = store }
}

// WithBrushMaxDistance sets the maximum raycast distance for item-bound brushes
// handled by Handler. Values <= 0 keep the default.
func WithBrushMaxDistance(distance float64) Option {
	return func(c *Config) { c.BrushMaxDistance = distance }
}

// WithMaxSelectionVolume sets an opt-in selection volume limit. A value of 0
// means unlimited.
func WithMaxSelectionVolume(limit int) Option {
	return func(c *Config) { c.MaxSelectionVolume = limit }
}

// WithMaxShapeVolume sets an opt-in shape volume limit. A value of 0 means
// unlimited.
func WithMaxShapeVolume(limit int) Option {
	return func(c *Config) { c.MaxShapeVolume = limit }
}

// WithMaxBrushVolume sets an opt-in brush volume limit. A value of 0 means
// unlimited.
func WithMaxBrushVolume(limit int) Option {
	return func(c *Config) { c.MaxBrushVolume = limit }
}

// WithMaxStackCopies sets an opt-in stack copy limit. A value of 0 means
// unlimited.
func WithMaxStackCopies(limit int) Option {
	return func(c *Config) { c.MaxStackCopies = limit }
}

// WithMaxEditSubChunks sets the cap for unique sub-chunks touched by one edit.
// A value of 0 disables this client-cache safety check.
func WithMaxEditSubChunks(limit int) Option {
	return func(c *Config) { c.MaxEditSubChunks = limit }
}
