# we

World editing library for [Dragonfly](https://github.com/df-mc/dragonfly). Import `github.com/df-mc/we` and attach `we.NewHandler` to players. Commands register when the `cmd` package loads (the root package blank-imports `_ "github.com/df-mc/we/cmd"`).

On Bedrock, WorldEdit-style commands are typed with a double slash (`//set`, `//copy`, …). Dragonfly strips one leading slash so the registered names are `/set`, `/copy`, etc.

## Architecture

The project follows a lightweight hexagonal split rather than a heavy DDD/onion
framework:

- `edit`, `geo`, `history`, and `parse` are the core domain packages. They do
  block geometry, reversible edits, and block-state parsing without command or
  form dependencies.
- `session` owns per-player application state: selections, clipboard, and undo
  stacks.
- `service` coordinates use-case validation, history recording, and calls into
  the core packages without depending on Dragonfly command/form adapters.
- `cmd`, `handler.go`, `editbrush`, `palette`, and legacy `brush`/`act` code are
  Dragonfly adapters. They translate player commands, forms, item metadata, and
  events into core edit operations.

Keep new features in that shape: add reusable block mutation logic to `edit`,
coordinate user-facing operations through `service`, store player state through
`session`, and keep Dragonfly command/form code thin.

## Configuration

`we.NewHandler` accepts optional Go configuration. Defaults preserve existing
behavior: 40 history entries per stack, a filesystem schematic store rooted at
`.we-schematics`, and a 128-block brush raycast distance. Guardrail fields
default to `0`, which means unlimited and is reserved for opt-in safety limits.

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

Servers that need non-filesystem schematic persistence can provide a custom
`edit.SchematicStore` with `we.WithSchematicStore(...)`. Leave guardrail options
at `0` to keep them unlimited.

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
| `//overlay` | `//overlay <blocks>` | Stacks layers of `<blocks>` on top of the highest solid blocks in the selection (surface overlay). |

### Clipboard

| Command | Syntax | Description |
|--------|--------|-------------|
| `//copy` | `//copy` or `//copy only <blocks>` | Copies the selection to your clipboard. With `only`, only listed block types are stored. |
| `//cut` | `//cut` | Copies the selection (including air), then clears the region to air. |
| `//paste` | `//paste` or `//paste -a` | Pastes at your feet, rotating to your facing. **`-a`**: do not write air (keep existing blocks where the clipboard has air). |
| `//rotate` | `//rotate <90\|180\|270\|360> [x\|y\|z]` | Rotates blocks **inside the selection** in place (around the region center; default axis **y**). Does not use the clipboard buffer. |
| `//flip` | `//flip` or `//flip <axis>` | Mirrors blocks **inside the selection** across **x**, **y**, or **z**. Without `axis`, defaults to **x** or **z** from your facing. |

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

---

## Optional: palette and legacy brush commands

The `palette` and `brush` packages define extra command types (`/palette set|save|delete`, `/brush bind|unbind|undo`) for older flows. They are **not** registered by `we/cmd`; register them in your server if you use those APIs alongside `we.Handler`.
