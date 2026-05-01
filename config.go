package we

import (
	"github.com/df-mc/we/edit"
	"github.com/df-mc/we/session"
)

const defaultBrushMaxDistance = 128

// Config controls WorldEdit library behavior. Zero-valued guardrail fields are
// intentionally unlimited so existing servers keep their current behavior unless
// they opt in to limits.
type Config struct {
	// HistoryLimit caps the number of undo/redo batches kept per player stack.
	HistoryLimit int
	// SchematicDirectory is the filesystem directory used by the default JSON
	// schematic helpers.
	SchematicDirectory string
	// BrushMaxDistance is the maximum raycast distance for item-bound editbrush
	// use through Handler.
	BrushMaxDistance float64

	// Future guardrails. A value of 0 means unlimited.
	MaxSelectionVolume int
	MaxShapeVolume     int
	MaxBrushVolume     int
	MaxStackCopies     int
}

// Option customises Config for a Handler.
type Option func(*Config)

// DefaultConfig returns behavior-preserving defaults.
func DefaultConfig() Config {
	return Config{
		HistoryLimit:       session.DefaultHistoryLimit,
		SchematicDirectory: edit.DefaultSchematicDirectory,
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
	if cfg.BrushMaxDistance <= 0 {
		cfg.BrushMaxDistance = defaultBrushMaxDistance
	}
	return cfg
}

// WithHistoryLimit sets the undo/redo stack cap for sessions created by the
// Handler. Values <= 0 keep the default.
func WithHistoryLimit(limit int) Option {
	return func(c *Config) { c.HistoryLimit = limit }
}

// WithSchematicDirectory sets the directory used by the default schematic disk
// helpers. An empty path keeps the default.
func WithSchematicDirectory(dir string) Option {
	return func(c *Config) { c.SchematicDirectory = dir }
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
