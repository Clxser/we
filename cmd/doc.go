// Package cmd exposes Dragonfly command adapters for WorldEdit operations.
//
// Command files are split by user-facing feature area. They should stay thin:
// parse command arguments, fetch player/session state, call edit or editbrush
// use cases, then format Dragonfly command output.
package cmd
