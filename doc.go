// Package we wires WorldEdit commands, session state, and item-bound brush
// metadata into a Dragonfly player handler.
//
// The package intentionally keeps the Dragonfly adapter at the edge: block
// editing primitives live in edit, reversible history lives in history, and
// per-player state lives in session. Servers generally only need NewHandler.
package we
