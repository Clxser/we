// Package we wires WorldEdit commands, session state, and item-bound brush
// metadata into a Dragonfly player handler.
//
// Servers attach we.NewHandler to each player. Importing this package also
// imports the cmd subpackage as a side effect, registering the //wand,
// //set, //copy, //schematic, //brush, and other commands with Dragonfly:
//
//	import "github.com/df-mc/we"
//
//	p.Handle(we.NewHandler(p))
//
// Behavior is configured through Option values. Defaults are a 40-entry undo
// stack, a filesystem schematic store rooted at edit.DefaultSchematicDirectory,
// and unlimited edit/selection/shape/brush/stack volumes:
//
//	p.Handle(we.NewHandler(p,
//	    we.WithHistoryLimit(100),
//	    we.WithSchematicDirectory("schematics"),
//	    we.WithMaxSelectionVolume(1_000_000),
//	))
//
// Guardrail options use 0 to mean unlimited, so a server only opts in to
// safety limits by passing a positive value.
//
// Servers that need non-filesystem schematic storage implement
// edit.SchematicStore and pass it through we.WithSchematicStore. The
// //schematic command and the schematic brush both go through that interface.
//
// The package intentionally keeps the Dragonfly adapter at the edge: block
// editing primitives live in edit, application use cases live in service,
// reversible history lives in history, and per-player state lives in session.
package we
