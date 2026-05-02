// Package session owns per-player WorldEdit state.
//
// It stores selections, clipboards, and undo history behind a small API so
// command and handler adapters do not need to manage their own global state.
// Clipboards are retained briefly after disconnect so players can reconnect
// during the same server lifetime without losing copied regions.
package session
