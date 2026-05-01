// Package history records reversible world changes for undo and redo.
//
// Batches capture before/after snapshots at edited positions and are compacted
// before being stored on command or brush-specific stacks.
package history
