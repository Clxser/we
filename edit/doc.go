// Package edit contains the core world-editing operations.
//
// This package is the domain core for block changes: it does not know about
// Dragonfly commands, forms, players, or item metadata. Callers provide a world
// transaction, geometry, masks, and a history batch to record reversible edits.
package edit
