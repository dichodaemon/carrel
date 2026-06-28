---
title: Carrel CLI Improvements -- Implementation Plan
status: issued
date: 2026-06-28
author: Dizan Vasquez
brief: ../briefs/2026-06-28_carrel-cli-improvements_brief.md
---

# Carrel CLI Improvements -- Implementation Plan

## 1. Implementation Status

**Phases:**
1. **Registry core** — `AddEntry` idempotency and `RemoveEntry` slot cleanup (no upstream deps)
2. **CLI surface** — command removal, rename, and new `slot sync-all` (depends on phase 1)
3. **Documentation** — update existing docs and rules to reflect CLI changes; document streamlined agent workflow (depends on phase 2)
4. **Type docs and validation** — create 16 `rule://type-<type>` docs and 16 `validate-<type>` skills (depends on phase 2)
5. **Cleanup and verification** — registry cleanup, dead interface method removal, final audit, build verification (depends on phases 1-4)

| # | Task | Status |
|---|---|---|
| 1.1 | Make `AddEntry` idempotent in `internal/authoring/authoring.go` — search by source+type+name, reuse UUID if found | Pending |
| 1.2 | Add `UnlinkAllEntrySlots(entryID)` method to `internal/registry/dolt_registry.go` and `mem_registry.go` | Pending |
| 1.3 | Modify `RemoveEntry` in `internal/authoring/authoring.go` to call `UnlinkAllEntrySlots` before removing the entry | Pending |
| 1.4 | Remove `EditEntry` and `UpdateEntryMeta` from `internal/authoring/authoring.go` (dead after `edit` removal) | Pending |
| 1.5 | Remove `TestEditEntry`, `TestUpdateEntryMeta`, and `TestUpdateEntryMetaNotFound` from `internal/authoring/authoring_test.go` (tests for removed functions) | Pending |
| 1.6 | Write test: `TestAddEntryIdempotent` — repeat `AddEntry` with same source+type+name, verify one row, updated hash | Pending |
| 1.7 | Write test: `TestRemoveEntryUnlinksSlots` — add entry to slots, remove entry, verify slot links gone | Pending |
| 1.8 | Write test: `TestUnlinkAllEntrySlots` in `internal/registry/registry_test.go` — link entries to slots, unlink all, verify zero remaining; unlink nonexistent entry (no-op) | Pending |
| 1.9 | Verify: `go test ./internal/authoring/... -count=1` passes | Pending |
| 1.10 | Verify: `go test ./internal/registry/... -run TestUnlinkAllEntrySlots -count=1` passes | Pending |
| 1.11 | Verify: `go build ./internal/authoring/... ./internal/registry/...` succeeds | Pending |
| 2.1 | Remove `crudEditCmd` and `crudUpdateCmd` from `cmd/carrel/crud.go` | Pending |
| 2.2 | Remove `crudEditCmd` and `crudUpdateCmd` from wiring array in `cmd/carrel/config.go` (line 24) | Pending |
| 2.3 | Modify `crudAddCmd` in `cmd/carrel/crud.go` — update error message and help text to remove false stdin reference; idempotent `AddEntry` behavior flows from task 1.1 | Pending |
| 2.4 | Modify `crudRmCmd` in `cmd/carrel/crud.go` — call `authoring.RemoveEntry` instead of raw `reg.RemoveEntry`; remove direct `os.Remove` (authoring layer handles file delete + slot unlinking) | Pending |
| 2.5 | Remove `verifyCmd` function from `cmd/carrel/main.go` (lines 395-417) and from query wiring array (line 39) | Pending |
| 2.6 | Rename scan command: change `Use:` string from `"scan <source-alias>"` to `"scan-sources <source-alias>"` in `cmd/carrel/main.go` line 102; add `Aliases: []string{"scan"}` and `Deprecated: "use scan-sources"` for backward compat with deprecation notice | Pending |
| 2.7 | Add `carrel slot sync-all` command in `cmd/carrel/slot_sync.go` and wire into `slotCmd()` in `cmd/carrel/slots.go`. Discovers unwired entries, wires them to slots, writes slot declarations file. Flags: `--exclude` (compound IDs), `--dry-run` | Pending |
| 2.8 | Write CLI E2E tests in `cmd/carrel/slot_sync_test.go` — `TestSlotSyncAllDryRun` (shows unwired entries, no writes), `TestSlotSyncAll` (wires all unwired entries to slots), `TestSlotSyncAllExclude` (respects `--exclude`), `TestRemovedCommands` (edit/update/verify return "unknown command") | Pending |
| 2.9 | Verify: `go test ./cmd/carrel/... -count=1` passes | Pending |
| 2.10 | Verify: `go build ./cmd/carrel` succeeds | Pending |
| 2.11 | Write test: `TestDeploymentReadsDisk` in `cmd/carrel/run_test.go` — modify a registered entry's source file on disk without re-running `carrel config add`, run `carrel plan`, confirm the deployment plan ContentHash reflects the on-disk file content (not the stale registry entry hash) | Pending |
| 2.12 | Verify: `go test ./cmd/carrel/... -run TestDeploymentReadsDisk -count=1` passes | Pending |
| 2.13 | Verify: `carrel config scan <src>` shows deprecation notice forwarding to `scan-sources` and `carrel config scan-sources <src>` produces identical output to old `scan` command | Pending |
| 3.1 | Update `omp/rules/carrel-configuration.md` — replace `config edit` and `config scan` references with idempotent `add` workflow; document streamlined agent workflow (write source file → `carrel config add --file=` → `carrel slot sync-all` → `carrel run`); remove `carrel verify`; update examples | Pending |
| 3.2 | Update `omp/skills/create-skill/SKILL.md` — replace `carrel config scan` reference (line 97) with `carrel config add --file=` | Pending |
| 3.3 | Update `omp/skills/update-skill/SKILL.md` — replace all `config edit` and `config scan` references with `add --file=`; remove `carrel verify` references | Pending |
| 3.4 | Update `README.md` — remove `carrel verify` reference (line 38); remove `edit` and `update` from CRUD listing at line 42 | Pending |
| 3.5 | Create `docs/workflow/2026-06-28_agent-configuration-workflow_cookbook.md` — describe the streamlined workflow: write source file → `carrel config add --file=` → `carrel slot sync-all` → `carrel run`, with examples per entry type | Pending |
| 3.6 | Verify: `carrel config add rule carrel-configuration --file=omp/rules/carrel-configuration.md` succeeds | Pending |
| 3.7 | Verify: `carrel config add skill create-skill --file=omp/skills/create-skill/SKILL.md` succeeds | Pending |
| 3.8 | Verify: `carrel config add skill update-skill --file=omp/skills/update-skill/SKILL.md` succeeds | Pending |
| 3.9 | Verify: `grep -q "carrel config add --file=" README.md && ! grep -q "carrel verify" README.md` (README references new workflow, old references removed) | Pending |
| 3.10 | Verify: `grep -q "carrel config add --file=" docs/workflow/2026-06-28_agent-configuration-workflow_cookbook.md && grep -q "slot sync-all" docs/workflow/2026-06-28_agent-configuration-workflow_cookbook.md && grep -q "carrel run" docs/workflow/2026-06-28_agent-configuration-workflow_cookbook.md` (cookbook covers full workflow chain) | Pending |
| 3.11 | Verify: `carrel plan` succeeds with all updated documentation entries registered | Pending |
| 4.1 | Create 16 `rule://type-<type>.md` files in `omp/rules/` covering: format, structure, naming, best practices, deployment behavior per type. Each references its `validate-<type>` skill | Pending |
| 4.2 | Create 16 `validate-<type>` skill directories + `SKILL.md` files in `omp/skills/` — validates entry content, naming, cross-references. Works on both new and existing entries | Pending |
| 4.3 | Wire all 16 rules and 16 skills via `carrel slot sync-all` (or slots.yml entries) | Pending |
| 4.4 | Verify: `carrel config list rule | grep type-` shows 16 entries | Pending |
| 4.5 | Verify: `carrel config list skill | grep validate-` shows 16 entries | Pending |
| 4.6 | Verify: `carrel plan` produces expected deployment output with all 32 new entries (16 rules + 16 skills) wired to their slots | Pending |
| 5.1 | Remove `slot-defaults.yml` from registry: `carrel config rm append-system slot-defaults.yml` | Pending |
| 5.2 | Run `carrel slot sync` to ensure no broken references | Pending |
| 5.3 | Verify: `carrel plan` shows no broken slot references and all planned changes are expected, no regressions | Pending |
| 5.4 | Remove `UpdateEntryMeta` method from `Registry` interface (`internal/registry/registry.go`), `MetaUpdates` type, and implementations in `internal/registry/dolt_registry.go` and `internal/registry/mem_registry.go` (dead after `crudUpdateCmd` and authoring wrapper removal) | Pending |
| 5.5 | Verify: `go test ./... -count=1` passes | Pending |
| 5.6 | Verify: `go build ./cmd/carrel` produces working binary | Pending |
| 5.7 | Update `docs/workflow/briefs/2026-06-28_carrel-cli-improvements_brief.md` — note that approach §4.4 (hash recomputation) was already implemented via `resolveOutputContent` in `cmd/carrel/run.go`; the feature predates the brief | Pending |
| 5.8 | Verify: `go build ./cmd/carrel` succeeds | Pending |

## 2. Architecture

### 2.1. Directory Layout

| File | Change |
|---|---|
| `internal/authoring/authoring.go` | Modify `AddEntry` for idempotent upsert; remove `EditEntry`, `UpdateEntryMeta`; modify `RemoveEntry` to unlink slots |
| `internal/authoring/authoring_test.go` | Remove `TestEditEntry`, `TestUpdateEntryMeta`, `TestUpdateEntryMetaNotFound`; add `TestAddEntryIdempotent`, `TestRemoveEntryUnlinksSlots` |
| `internal/registry/dolt_registry.go` | Add `UnlinkAllEntrySlots(entryID)`; remove `UpdateEntryMeta` |
| `internal/registry/mem_registry.go` | Add `UnlinkAllEntrySlots(entryID)`; remove `UpdateEntryMeta` |
| `internal/registry/registry_test.go` | Add `TestUnlinkAllEntrySlots` |
| `internal/registry/registry.go` | Add `UnlinkAllEntrySlots` to `Registry` interface; remove `UpdateEntryMeta` and `MetaUpdates` type |
| `cmd/carrel/crud.go` | Remove `crudEditCmd`, `crudUpdateCmd`; modify `crudRmCmd` to delegate to authoring layer |
| `cmd/carrel/config.go` | Remove `crudEditCmd` and `crudUpdateCmd` from wiring |
| `cmd/carrel/main.go` | Remove `verifyCmd`; rename scan to `scan-sources` with backward alias |
| `cmd/carrel/slot_sync.go` | Add `slotSyncAllCmd` command |
| `cmd/carrel/slot_sync_test.go` | New file: E2E tests for `slot sync-all` and removed-command verification |
| `cmd/carrel/run_test.go` | New file: add `TestDeploymentReadsDisk` |
| `cmd/carrel/slots.go` | Add `slotSyncAllCmd` to `slotCmd()` subcommand wiring |
| `omp/rules/carrel-configuration.md` | Update references; document streamlined agent workflow |
| `omp/skills/create-skill/SKILL.md` | Update references |
| `omp/skills/update-skill/SKILL.md` | Update references |
| `README.md` | Remove `verify` reference; remove `edit` and `update` from CRUD listing |
| `docs/workflow/2026-06-28_agent-configuration-workflow_cookbook.md` | New file: agent configuration workflow cookbook |
| `omp/rules/type-rule.md` … `omp/rules/type-nvim.md` | 16 new rule files |
| `omp/skills/validate-rule/SKILL.md` … `omp/skills/validate-nvim/SKILL.md` | 16 new skill directories |

### 2.2. Dependency Graph

```
  Phase 1 ──> Phase 2 ──> Phase 3
                 │
                 └──> Phase 4

  Phase 1-4 ──> Phase 5
```

## 3. Interface Changes

### 3.1. Registry Interface — `UnlinkAllEntrySlots` (new)

```go
// UnlinkAllEntrySlots removes all slot links for an entry.
UnlinkAllEntrySlots(entryID uuid.UUID) error
```

Added to `internal/registry/registry.go` `Registry` interface. Implemented in both `DoltRegistry` and `MemRegistry`.

Dolt implementation:
```go
func (r *DoltRegistry) UnlinkAllEntrySlots(entryID uuid.UUID) error {
    _, err := r.db.Exec(`DELETE FROM entry_slots WHERE entry_id = ?`, entryID.String())
    return err
}
```

### 3.2. `AddEntry` signature change

Before:
```go
func AddEntry(reg registry.Registry, typ registry.CapabilityType, name string, sourceAlias string, content []byte) (registry.Entry, error)
```

After — same signature, but behavior changes: if an entry with the same (source, type, name) already exists, the existing UUID is reused and content is updated (upsert via `RegisterEntry`'s `ON DUPLICATE KEY UPDATE`). Otherwise, a new UUID is generated. Stdin input is dropped — `crudAddCmd`'s `else` branch returns an error rather than reading stdin; the error message and help text are updated to remove the false stdin reference.

### 3.3. Functions removed

| Function | File | Replacement |
|---|---|---|
| `EditEntry` | `internal/authoring/authoring.go` | `AddEntry` (idempotent) |
| `UpdateEntryMeta` (authoring wrapper) | `internal/authoring/authoring.go` | None |
| `UpdateEntryMeta` (Registry method) | `internal/registry/registry.go` | None |
| `MetaUpdates` (type) | `internal/registry/registry.go` | None |
| `crudEditCmd` | `cmd/carrel/crud.go` | `crudAddCmd` |
| `crudUpdateCmd` | `cmd/carrel/crud.go` | None |
| `verifyCmd` | `cmd/carrel/main.go` | `carrel deployed` |

## 4. Solution Breakdown

### 4.1. `AddEntry` idempotency (`internal/authoring/authoring.go`)

**Logic:**
1. Resolve source by alias (existing behavior)
2. Scan existing entries in the source for matching (type, name)
3. If found: reuse `entry.ID`, update `entry.ContentHash` from new content, write file, call `reg.RegisterEntry` (upserts by UUID)
4. If not found: generate `uuid.New()`, create entry, write file, call `reg.RegisterEntry`

**Edge cases:**
- Source not found → return error (existing behavior)
- File write fails → return error before registry write (atomicity concern: file written, registry not updated — acceptable; `add` is idempotent, re-running fixes it)
- Stdin is no longer supported — `crudAddCmd` error message updated to remove the false claim; `--file=` is the standard input path

**Dependencies:** produces the idempotent `add` behavior that phase 2 CLI relies on.

**Done condition:** Task 1.6 — `TestAddEntryIdempotent`; verified by task 1.9.

### 4.2. `carrel slot sync-all` (`cmd/carrel/slot_sync.go`)

**Logic:**
1. Resolve consumer (from arg or cwd)
2. List all registered entries for the consumer's sources
3. List all current slots and their linked entries
4. Compute delta: entries not linked to any slot
5. If `--dry-run`: print delta and exit
6. Filter out entries matching `--exclude` compound IDs
7. Determine slot declarations file: `.carrel/slots.yml` if present, else `slot-defaults.yml`
8. For each unwired entry:
   - Determine slot name (entry name for rules/skills; `APPEND_SYSTEM.md` for append-system entries)
   - Auto-create slot if it doesn't exist (compose mode: override for singles, concat for append-system)
   - Link entry to slot
9. Write updated slot declarations file
10. Print summary

**Edge cases:**
- `append-system` entry but no `APPEND_SYSTEM.md` slot → auto-create it
- Slot already exists with wrong compose mode → warn, don't change
- Entry excluded but already wired → warn or skip silently

**Dependencies:** requires registry methods to discover unwired entries efficiently; wired into `slotCmd()` in `cmd/carrel/slots.go`.

**Done condition:** Task 2.9 — `TestSlotSyncAllDryRun`, `TestSlotSyncAll`, `TestSlotSyncAllExclude` pass.

### 4.3. CLI command removal and rename (`cmd/carrel/crud.go`, `cmd/carrel/config.go`, `cmd/carrel/main.go`)

**Changes:**
- Delete `crudEditCmd` and `crudUpdateCmd` functions from `crud.go` (tasks 2.1, 2.2)
- Modify `crudAddCmd` to use idempotent `authoring.AddEntry` and drop the false stdin reference in error message and help text (task 2.3)
- Modify `crudRmCmd` to delegate to `authoring.RemoveEntry` instead of direct `reg.RemoveEntry` + manual `os.Remove` (task 2.4)
- Delete `verifyCmd` function from `main.go` and remove from query wiring array (task 2.5)
- Rename `scanCmd`'s `Use:` string from `"scan <source-alias>"` to `"scan-sources <source-alias>"`, add `Aliases: []string{"scan"}` (task 2.6)

**Dependencies:** requires idempotent `AddEntry` from phase 1 (task 2.3); the other changes are purely subtractive.

**Done condition:** Tasks 2.9, 2.13 — `TestRemovedCommands` confirms removed commands return "unknown command"; scan deprecation notice and scan-sources parity verified.

### 4.4. Authoring dead code removal (`internal/authoring/authoring.go`, `internal/authoring/authoring_test.go`)

**Changes:**
- Remove `EditEntry` function from `internal/authoring/authoring.go` — dead after `crudEditCmd` removal makes idempotent `AddEntry` the sole entry-update path (task 1.4)
- Remove `UpdateEntryMeta` wrapper from `internal/authoring/authoring.go` — dead after `crudUpdateCmd` removal (task 1.4)
- Remove `TestEditEntry`, `TestUpdateEntryMeta`, `TestUpdateEntryMetaNotFound` from `internal/authoring/authoring_test.go` (task 1.5)

**Edge cases:**
- Verify no remaining callers outside the authoring package before removing (grep for `EditEntry` and `UpdateEntryMeta` across the codebase)

**Dependencies:** follows CLI command removal (§4.3); `crudEditCmd` and `crudUpdateCmd` must be deleted first.

**Done condition:** Task 1.11 — `go build ./internal/authoring/...` succeeds with no compile errors from dead code callers.

### 4.5. `RemoveEntry` slot cleanup (`internal/authoring/authoring.go`)

**Logic:**
1. Find entry by type+name (existing behavior)
2. Call `reg.UnlinkAllEntrySlots(entry.ID)` — new call
3. Remove file from disk (existing behavior, tolerates missing)
4. Call `reg.RemoveEntry(entry.ID)` (existing behavior)

**Edge cases:**
- Entry not found → return error (existing behavior)
- `UnlinkAllEntrySlots` fails → return error; entry and file are untouched
- File already deleted → `os.IsNotExist` is tolerated (existing behavior)

**Dependencies:** requires `UnlinkAllEntrySlots` from phase 1.

**Done condition:** Task 1.7 — `TestRemoveEntryUnlinksSlots`; verified by task 1.9.


### 4.6. Documentation update (Phase 3)

**Changes:** Update four existing files and create one new cookbook to reflect the streamlined CLI:
1. `omp/rules/carrel-configuration.md` — replace `config edit`/`config scan` with idempotent `add --file=` workflow; remove `carrel verify`; add `slot sync-all` step (task 3.1)
2. `omp/skills/create-skill/SKILL.md` — replace `carrel config scan` with `carrel config add --file=` (task 3.2)
3. `omp/skills/update-skill/SKILL.md` — replace `config edit`/`config scan`/`carrel verify` with `add --file=` (task 3.3)
4. `README.md` — remove `carrel verify` reference (line 38); remove `edit` and `update` from CRUD listing (line 42) (task 3.4)
5. `docs/workflow/2026-06-28_agent-configuration-workflow_cookbook.md` — new cookbook describing: write source file → `carrel config add --file=` → `carrel slot sync-all` → `carrel run` (task 3.5)

**Dependencies:** all doc changes reference commands modified in phase 2.

**Done condition:** Tasks 3.6–3.11 — all Verify: tasks pass, including `carrel plan` with all updated documentation entries.

### 4.7. Type documentation and validation skills (Phase 4)

**Changes:** Create 32 new files:
1. 16 `rule://type-<type>.md` files in `omp/rules/` — one per config type, covering format, structure, naming, best practices, deployment behavior. Each references its `validate-<type>` skill (task 4.1)
2. 16 `validate-<type>` skill directories + `SKILL.md` files in `omp/skills/` — validates entry content, naming, and cross-references. Works on both new and existing entries (task 4.2)

**Dependencies:** entries are registered via `carrel config add --file=` (phase 2 workflow); wired via `carrel slot sync-all` (task 2.7 / 4.3).

**Done condition:** Tasks 4.4–4.6 — all 32 entries listed in registry and deployable.

### 4.8. Registry cleanup and dead code removal (Phase 5)

**Changes:**
1. Remove `slot-defaults.yml` from the registry (task 5.1)
2. Run `carrel slot sync` to verify no broken references (task 5.2)
3. Verify deployment plan integrity (task 5.3)
4. Remove `UpdateEntryMeta` from `Registry` interface, `MetaUpdates` type, and Dolt/Mem implementations (task 5.4)
5. Update the companion brief to note approach §4.4 was already implemented (task 5.7)

**Dependencies:** phases 1-4 must be complete (all new entries registered, all doc updates done, all dead code identified).

**Done condition:** Tasks 5.5, 5.8 — full test suite and build pass.

## 5. Design Decisions

### 5.1. `scan` → `scan-sources` with backward alias

**Decision:** Rename the command but keep `scan` as an alias (`Aliases: []string{"scan"}`).

**Considered:** Hard rename with no backward compat.
**Why:** Existing beads issues and briefs reference `carrel config scan`. Breaking scripts immediately creates friction with no benefit. The alias can be removed later.

### 5.2. `rm` delegates to authoring layer

**Decision:** `crudRmCmd` calls `authoring.RemoveEntry` instead of direct `reg.RemoveEntry` + manual file delete.

**Considered:** Keep `rm` doing file delete + registry delete directly, add slot unlinking there.
**Why:** The authoring layer already owns the file-path derivation logic. Adding slot unlinking there keeps a single authority for entry lifecycle.

### 5.3. `carrel run` hash recomputation not needed

**Decision:** No code change for hash recomputation in deployment.

**Why:** `resolveOutputContent` already reads source files from disk and computes `ContentHash` at deploy time. The stale-hash problem was only in the registry entry hash (used for `carrel sources` drift display), not in the deployment pipeline. Making `add` idempotent with correct hash computation addresses the only remaining gap.

**Deviation from brief §4.4:** the brief proposed making `carrel run` recompute hashes from disk. This was already implemented — `resolveOutputContent` in `cmd/carrel/run.go` reads source files from disk and computes `ContentHash` at deploy time. The brief's investigation predated discovery of this existing behavior. No code change required; adding test 2.11 for regression prevention and task 5.7 to update the brief.

### 5.4. `config add` stdin dropped

**Decision:** Drop stdin support for `config add`. The `else` branch in `crudAddCmd` currently returns an error claiming stdin is supported, but no stdin-reading code exists.

**Considered:** Implement actual stdin reading to match the advertised behavior.
**Why:** The `--content` and `--file` flags cover all real use cases. Stdin support was never implemented — only the error message falsely advertised it. Dropping the false claim is simpler than implementing a third input path. Agent workflows already use `--file=`, which is more explicit and auditable. Task 2.3 updates `crudAddCmd`'s error message and help text to remove the false stdin reference.

### 5.5. `Registry.UpdateEntryMeta` removal

**Decision:** Remove `UpdateEntryMeta` from the `Registry` interface, the `MetaUpdates` type, and both `DoltRegistry`/`MemRegistry` implementations.

**Considered:** Retain for potential programmatic use.
**Why:** After removing `crudUpdateCmd` (the only CLI caller) and the authoring wrapper, the method has zero callers. Retaining it invites future confusion. If metadata update capabilities are needed later, they can be re-added with a clear use case.

### 5.6. `slot sync-all` auto-creates `APPEND_SYSTEM.md` slot

**Decision:** `carrel slot sync-all` auto-creates the `APPEND_SYSTEM.md` slot if it doesn't exist and unwired append-system entries are found.

**Considered:** Require the slot to exist beforehand and error if missing. This would force an extra manual step before `slot sync-all` can wire append-system entries.
**Why:** The `APPEND_SYSTEM.md` slot is the only destination for append-system entries. Auto-creating it with compose mode `concat` removes a manual step without ambiguity — there is no other slot an append-system entry would go to. The slot's path and compose mode are deterministic given the slot-declarations file context.

## 6. Success Criteria

### Registry core
- [ ] `AddEntry` with existing (source, type, name) reuses UUID instead of creating duplicate  **(→ 1.9)**
- [ ] `RemoveEntry` removes all slot links before removing the entry  **(→ 1.9)**
- [ ] `EditEntry` and `UpdateEntryMeta` are deleted; all callers compile  **(→ 1.11)**

### CLI surface
- [ ] `carrel config edit` returns "unknown command"  **(→ 2.9)**
- [ ] `carrel config update` returns "unknown command"  **(→ 2.9)**
- [ ] `carrel verify` returns "unknown command"  **(→ 2.9)**
- [ ] `carrel config scan <src>` shows deprecation notice and delegates to scan-sources  **(→ 2.13)**
- [ ] `carrel config scan-sources <src>` works identically to old scan  **(→ 2.13)**
- [ ] `carrel slot sync-all --dry-run` shows unwired entries  **(→ 2.9)**
- [ ] `carrel slot sync-all` wires all unwired entries  **(→ 2.9)**
- [ ] `carrel slot sync-all --exclude carrel-omp:skill:audit-document` wires all except audit-document  **(→ 2.9)**
- [ ] `carrel plan` deployment ContentHash reflects on-disk file content, not stale registry hash  **(→ 2.12)**

### Documentation
- [ ] `carrel-configuration.md` no longer references `config edit`, `config scan`, `carrel verify`  **(→ 3.6)**
- [ ] `carrel-configuration.md` documents streamlined agent workflow: write → add → slot sync-all → run  **(→ 3.6)**
- [ ] `create-skill/SKILL.md` uses `add --file=` workflow, no `config scan` references  **(→ 3.7)**
- [ ] `update-skill/SKILL.md` uses `add --file=` workflow, no `config edit`/`config scan`/`carrel verify` references  **(→ 3.8)**
- [ ] `README.md` references `carrel config add --file=` and no longer references `carrel verify` or removed commands  **(→ 3.9)**
- [ ] Agent configuration workflow cookbook exists and covers the full workflow chain  **(→ 3.10)**

### Registry cleanup
- [ ] `slot-defaults.yml` is no longer in the registry  **(→ 5.5)**
- [ ] No broken slot references  **(→ 5.3)**
- [ ] `UpdateEntryMeta` and `MetaUpdates` removed from Registry interface and implementations  **(→ 5.5)**

## 7. Document Staleness Audit

| Document | Invalidated? | Action |
|---|---|---|
| `omp/rules/carrel-configuration.md` | Yes — describes `config edit`, `config scan`, `carrel verify`; lacks streamlined workflow | Task 3.1: update with new workflow |
| `omp/skills/create-skill/SKILL.md` | Yes — references `carrel config scan` | Task 3.2: update |
| `omp/skills/update-skill/SKILL.md` | Yes — references `config edit`, `config scan`, `carrel verify` | Task 3.3: update |
| `README.md` | Yes — references `carrel verify` and lists removed `edit`/`update` commands | Task 3.4: update |
| `docs/workflow/briefs/2026-06-28_carrel-cli-improvements_brief.md` | Yes — approach §4.4 proposed hash recomputation that was already implemented | Task 5.7: note that `resolveOutputContent` predates the brief |
| `docs/workflow/briefs/2026-05-08_bootstrap-configuration-file_brief.md` | No — point-in-time brief; any stale command references belong to the time of writing | None |
| `docs/workflow/briefs/2026-05-08_core-prompt-reorganization_brief.md` | No — point-in-time brief | None |

## 8. Cleanup

No diagnostic instrumentation planned. If any temporary logging is added during `slot sync-all` debugging, it will be removed before the phase 5 verification.
