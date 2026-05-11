# Compact Java Schematic Paste Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a memory-efficient Java schematic paste fast path that avoids building 90M-entry clipboards for `//schematic paste <name> -noundo`.

**Architecture:** Keep current clipboard behavior as the fallback. Add a compact palette-backed schematic representation in `edit` and expose a file-store fast path only for Java `.schem`/`.schematic` files when undo and air-skipping are disabled. The compact representation stores unique block/liquid states once and stores per-cell palette indices, then presents that data to Dragonfly as a `world.Structure`.

**Tech Stack:** Go 1.26, Dragonfly `world.Tx`/`world.Structure`, S2D streaming scan API, existing WE service/edit layers.

---

### Task 1: Compact representation and paste

**Files:**
- Create: `edit/compact_schematic.go`
- Test: `edit/compact_schematic_test.go`

- [ ] Write tests that build a small compact schematic with repeated block states, verify palette deduplication, verify dimensions, and verify paste writes expected blocks.
- [ ] Run `go test ./edit -run TestCompact -count=1` and confirm it fails because compact APIs do not exist.
- [ ] Implement `compactBlockState`, `CompactSchematic`, `AddBlock`, `PasteCompactSchematicNoUndo`, and the `world.Structure` adapter.
- [ ] Run `go test ./edit -run TestCompact -count=1` and confirm it passes.

### Task 2: Java scan to compact

**Files:**
- Modify: `edit/schematic_import.go`
- Test: `edit/compact_schematic_test.go`

- [ ] Add a test that imports a Java fixture via S2D into compact form and verifies no clipboard entries are involved.
- [ ] Implement `ImportJavaCompactSchematic(path string)` using `schem.ScanWithInfo` and `CompactSchematic.AddBlock`.
- [ ] Run `go test ./edit -run 'TestCompact|TestImportJavaCompact' -count=1`.

### Task 3: Service fast path

**Files:**
- Modify: `edit/schematic.go`
- Modify: `service/clipboard.go`
- Test: `service/service_test.go`

- [ ] Add a store interface method for compact Java load availability without changing existing `SchematicStore` callers.
- [ ] Add a service test that Java schematic paste with `-noundo` uses the compact path and reports the expected changed count.
- [ ] Wire `Schematic paste` to compact path only when `opts.NoUndo && !-a` and store supports it.
- [ ] Keep existing `store.Load` fallback for undo, `-a`, JSON schematics, and custom stores.

### Task 4: Verification and dependency rollout

**Files:**
- `go.mod` / `go.sum` in WE and Build as needed.

- [ ] Run WE gates: `go test ./...`, `go vet ./...`, `golangci-lint run ./...`.
- [ ] Commit and push WE.
- [ ] Bump Build's WE dependency, run Build gates, commit and push Build.
- [ ] Pull and test on VPS.
