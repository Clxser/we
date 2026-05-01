// Package session owns per-player WorldEdit state.
//
// It stores selections, clipboards, and undo history behind a small API so
// command and handler adapters do not need to manage their own global state.
package session
