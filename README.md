# we

World editing library for [Dragonfly](https://github.com/df-mc/dragonfly).

On Bedrock, WorldEdit-style commands are typed with a double slash (`//set`, `//copy`, …). Dragonfly strips one leading slash so the registered names are `/set`, `/copy`, etc.

## Setup

Import the package and attach `we.NewHandler` to each player. Commands register
automatically through the `we` package's blank import of `we/cmd`, so server
owners do not need to call any registration function themselves.

```go
import "github.com/df-mc/we"

// inside the player join callback
p.Handle(we.NewHandler(p))
```

That single line is enough to register all `//`-prefixed commands and let
players bind brushes to held items.

## Architecture

The project follows a lightweight hexagonal split rather than a heavy DDD/onion
framework:

- `edit`, `geo`, `history`, and `parse` are the core domain packages. They do
  block geometry, reversible edits, and block-state parsing without command or
  form dependencies.
- `session` owns per-player application state: selections, clipboard, and undo
  stacks.
- `service` coordinates use-case validation, history recording, brush execution,
  and calls into the core packages without depending on Dragonfly command/form
  adapters.
- `cmd`, `handler.go`, and `editbrush` are Dragonfly adapters. They
  translate player commands, forms, item metadata, and events into service/core
  operations.
- `guardrail` holds opt-in limits surfaced through `we.Config`.

Keep new features in that shape: add reusable block mutation logic to `edit`,
coordinate user-facing operations through `service`, store player state through
`session`, and keep Dragonfly command/form code thin.

## Configuration

`we.NewHandler` accepts variadic Go options. Calling it with no options is a
sensible default for development servers; production servers usually opt in to
guardrails and may swap the schematic store.

| Option | Default | What it controls |
|--------|---------|------------------|
| `WithHistoryLimit(n)` | `40` | Undo/redo stack cap per player. Values `<= 0` keep the default. |
| `WithSchematicDirectory(dir)` | `.we-schematics` | Directory used by the default filesystem schematic store. Empty keeps the default. |
| `WithSchematicStore(store)` | filesystem | Custom `edit.SchematicStore` implementation; see below. `nil` keeps the default. |
| `WithBrushMaxDistance(d)` | `128` | Maximum raycast distance for item-bound brushes. Values `<= 0` keep the default. |
| `WithMaxSelectionVolume(n)` | `0` (unlimited) | Reject selections with more blocks than `n`. |
| `WithMaxShapeVolume(n)` | `0` (unlimited) | Reject shape commands whose bounding volume exceeds `n`. |
| `WithMaxBrushVolume(n)` | `0` (unlimited) | Reject brush configurations whose footprint exceeds `n`. |
| `WithMaxStackCopies(n)` | `0` (unlimited) | Reject `//stack` requests beyond `n` copies. |

Every guardrail uses `0` to mean **unlimited**. There is no special "off" value
beyond that, and no need to opt out explicitly.

```go
p.Handle(we.NewHandler(p,
	we.WithHistoryLimit(100),
	we.WithSchematicDirectory("schematics"),
	we.WithBrushMaxDistance(96),
	we.WithMaxSelectionVolume(1_000_000),
	we.WithMaxShapeVolume(250_000),
	we.WithMaxBrushVolume(50_000),
	we.WithMaxStackCopies(16),
))
```

### Custom schematic store

`edit.SchematicStore` is a small interface (`Save`, `Load`, `Delete`, `List`)
for persisting clipboards by name. Servers that need database-backed or remote
storage implement it on their own type and pass an instance via
`we.WithSchematicStore`. The `//schematic` command and the schematic brush both
go through this interface, so a custom store is picked up everywhere
automatically.

## Selection

Most edits need two corners of an axis-aligned box.

| Command | Aliases | What it does |
|--------|---------|----------------|
| `//wand` | `wand` | Puts the selection wand on the held stack (or a wooden axe): **break** a block → `pos1`, **use on** a block → `pos2`. |
| `//pos1` | `pos1` | Sets `pos1` to the block under your feet (integer position). |
| `//pos2` | `pos2` | Sets `pos2` the same way. |

You can also use `//pos1` / `//pos2` instead of the wand. A valid selection requires both corners; the cuboid is their inclusive bounding box.

## Block lists and masks

- **Block lists** are comma-, semicolon-, or whitespace-separated names resolved by Dragonfly (`stone`, `oak_log`, …). Multiple blocks are chosen at random when filling.
- **Masks** (for replace, move, etc.):
  - `all` — every block, including air.
  - `only:type1,type2` or shorthand after stripping `only:` — only those block types (no air unless listed).

## Commands

### Region fill and clear

| Command | Syntax | Description |
|--------|--------|-------------|
| `//set` | `//set <blocks>` | Aliases: `set`, `/fill`, `fill`. Fills the whole selection with random picks from `<blocks>`. |
| `//replace` | `//replace <mask> <to-blocks>` | Replaces blocks matching `<mask>` inside the selection with `<to-blocks>`. |
| `//replacenear` | `//replacenear <distance> <mask> <to-blocks>` | Same replacement logic in a sphere (by Manhattan bounding box around you), not limited to selection. |
| `//toplayer` | `//toplayer <mask> <to-blocks>` | Within the selection, replaces only the **topmost** matching blocks per column. |
| `//overlay` | `//overlay <blocks>` | Aliases: `//layer`, `layer`. Stacks layers of `<blocks>` above the highest solid blocks in each selected column, including one block above the selection's max Y. |

### Clipboard

| Command | Syntax | Description |
|--------|--------|-------------|
| `//copy` | `//copy` or `//copy only <blocks>` | Copies the selection to your clipboard. With `only`, only listed block types are stored. |
| `//cut` | `//cut` | Copies the selection (including air), then clears the region to air. |
| `//paste` | `//paste` or `//paste -a` | Pastes at your feet, rotating to your facing. **`-a`**: do not write air (keep existing blocks where the clipboard has air). |
| `//clearclipboard` | `//clearclipboard` | Clears the stored clipboard. |
| `//rotate` | `//rotate <90\|180\|270\|360> [x\|y\|z]` | Rotates the clipboard around its origin; default axis is **y**. |
| `//flip` | `//flip` or `//flip <axis>` | Mirrors the clipboard across **x**, **y**, or **z**. Without `axis`, defaults to **x** or **z** from your facing. |

### Transform selection (world edits)

| Command | Syntax | Description |
|--------|--------|-------------|
| `//move` | `//move <mask> <distance> [-a]` | Shifts matching blocks by `<distance>` along **your horizontal facing**. **`-a`**: skip writing air when placing the moved slice. |
| `//stack` | `//stack <amount> [-a]` | Repeats the selection `<amount>` times in your facing direction. **`-a`**: same air behavior as paste/move. |

### Shapes and lines

Shapes are centered on **your position** (not the selection). Optional **`-h`** anywhere in the args builds a hollow shell where supported.

| Command | Syntax | Notes |
|--------|--------|--------|
| `//line` | `//line <blocks> <thickness>` | Line from `pos1` to `pos2` (both must be set). |
| `//sphere` | `//sphere [-h] <blocks> <radius> <height>` | **Ellipsoid** around you: horizontal radius `<radius>`, vertical size `<height>` (anchor-centred). |
| `//cylinder` | `//cylinder [-h] <blocks> <radius> <height>` | Vertical cylinder. |
| `//cone` | `//cone [-h] <blocks> <radius> <height>` | |
| `//pyramid` | `//pyramid [-h] <blocks> <length> <width> <height>` | |
| `//cube` | `//cube [-h] <blocks> <length> <width> <height>` | Axis-aligned box. |

Use commas for multiple block types in the first argument (e.g. `stone,dirt`).

### Utilities

| Command | Syntax | Description |
|--------|--------|-------------|
| `//center` | `//center <blocks>` | Places one block from `<blocks>` at the **center block** of the selection. |
| `//walls` | `//walls <blocks>` | Fills only the **outer shell** of the selection cuboid. |
| `//drain` | `//drain <radius>` | Removes fluids around you within `<radius>` (positive integer). |
| `//removeabove` | `//removeabove [height] [radius]` | Clears blocks above you. Defaults to height `64`, radius `0`. |
| `//removebelow` | `//removebelow [height] [radius]` | Clears blocks below you. Defaults to height `64`, radius `0`. |
| `//removenear` | `//removenear <blocks> <radius>` | Clears matching blocks in a sphere around you. |
| `//naturalize` | `//naturalize` | Converts selected terrain columns to grass, dirt, then stone layers. |
| `//searchitem` | `//searchitem <query>` | Aliases: `//search`, `//l`. Lists matching registered item/block IDs. |
| `//biome` | `//biome list` or `//biome set <biome>` | Lists registered biomes, or sets biome for all blocks in the **selection** for `set`. Your server must register biomes with Dragonfly or names will be missing / unknown. |

### Schematics

Saved files are handled by the configured `edit.SchematicStore`; the default
implementation writes JSON files under `.we-schematics`.

| Subcommand | Syntax | Description |
|------------|--------|-------------|
| create | `//schematic create <name>` | Aliases: `schematic`, `schem`, `/schem`. Saves current selection (full copy including air). |
| paste | `//schematic paste <name> [-a]` | Pastes like `//paste`; **`-a`** skips writing air. |
| delete | `//schematic delete <name>` | Removes a saved schematic. |
| list | `//schematic list` | Lists saved schematic names. |

### History

| Command | Syntax | Description |
|--------|--------|-------------|
| `//undo` | `//undo` or `//undo b` | Reverts the last main edit. **`b`** targets the **brush** undo stack only (see `//brush`). |
| `//redo` | `//redo` or `//redo b` | Redo; **`b`** same as undo. |

### Brush (bound item)

| Command | Syntax | Description |
|--------|--------|-------------|
| `//brush` | `//brush` | Opens the brush configuration **form** for the **held item** (must hold an item). |
| `//brush` | `//brush <type> [blocks] [radius]` | Quick bind: sets brush kind, optional comma-separated blocks, optional radius (also drives default height/length/width). |

Brushes are stored on the item; **use** the item to raycast and apply. Undo for brush ops is recorded separately; use `//undo b` / `//redo b` when you need brush-specific history.

Brush types available in the form include shapes (`sphere`, `cylinder`, …), `fill`, `toplayer`, `overlay`, `replace`, `line`, `schematic`, terrain tools, etc. (see `editbrush` package).
