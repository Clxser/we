// Package editbrush implements item-bound WorldEdit brushes.
//
// The package is split into config/form adapters and brush application logic.
// Brush application should delegate reusable block mutation behavior to edit
// and history instead of duplicating command behavior.
package editbrush
