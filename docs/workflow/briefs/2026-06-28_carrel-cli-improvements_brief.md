---
title: Carrel CLI Improvements
date: 2026-06-28
author: Dizan Vasquez
---

# Carrel CLI Improvements

## 1. Objective

Simplify carrel's command-line interface to the bare minimum needed for use and
maintenance, while fixing long-standing ergonomic issues — hash-handling
confusion, redundant commands, and the registration-to-slot wiring gap — so that
both humans and agents can manage OMP configuration confidently.

## 2. Scope

| In scope | Out of scope |
|---|---|
| CLI command removal, renaming, and consolidation | OS-level deployment (host-setup, os-setup) |
| Hash model: making deployment compute hashes from disk | Registry storage format or backend changes |
| Agent CRUD workflow: add (idempotent), rm, rename | Validation logic within carrel itself |
| Slot wiring: delta visibility and bulk-wire command | Slot composition mode changes |
| Source folder for type documentation and validation skills | Content of individual type docs or validation skills |
| Fixing `--file=` hash update (absolved by `edit` removal) | UI/TUI dashboard changes |

## 3. Sources

1. Carrel source: `/workspace/carrel/cmd/carrel/` (CLI definitions)
2. Carrel source: `/workspace/carrel/internal/authoring/` (entry CRUD logic)
3. Carrel source: `/workspace/carrel/internal/registry/` (registry and slots)
4. Carrel source: `/workspace/carrel/internal/deployer/` (deployment with hash checks)
5. Carrel source: `/workspace/carrel/internal/scanner/` (source scanning)
6. Carrel source: `/workspace/carrel/internal/query/` (plan, deployed, verify)
7. Registry state: `carrel sources`, `carrel slot list`, `carrel deployed`
8. Slot declarations: `/workspace/carrel/.carrel/slots.yml`, `/workspace/carrel/omp/slot-defaults.yml`
9. Existing rules: `rule://carrel-configuration`

## 4. Approach

1. Remove dead and redundant commands (`config update`, `verify`).
2. Collapse `config edit` into `config add` (make `add` idempotent via source+type+name lookup).
3. Rename `config scan` to `config scan-sources` to clarify it discovers new entries only, not hash resyncs.
4. Make `carrel run` recompute content hashes from disk before deploying — eliminating stale-hash bugs and making the hash transparent to agents.
5. Add `carrel slot sync-all [--exclude <compound-id>] [--dry-run]` for bulk wiring of unwired entries.
6. Remove `slot-defaults.yml` from the registry (registered as `append-system` but consumed as a slot declarations file directly from disk — the registry entry serves no purpose).
7. Ensure `config rm` unlinks the entry from all slots.
8. Document the streamlined agent workflow: write source file → `add --file=` → `slot sync-all` → `carrel run`.
9. Create `rule://type-<type>` documentation and `validate-<type>` skills for every entry type, auto-wired by default.

## 5. Constraints

- Slot model stays unchanged: `.carrel/slots.yml` is authoritative when present; `slot-defaults.yml` from universal sources is the fallback. They remain mutually exclusive, not cumulative.
- `carrel run` remains non-interactive for agents; the interactive prompt is a known stall hazard.
- Validation lives in agent-side skills (`validate-<type>`), not in carrel's add/edit commands.
- OS config types (zsh, wezterm, p10k, helix, nvim) are in scope for documentation and validation but their deployment commands are out of scope.
- `archive/` directory structure follows the document structure standards — this brief lives in `docs/workflow/briefs/`.

## 6. Open Questions

- Should `config add --file=` accept stdin when neither `--file` nor `--content` is given? (Currently supported; preserve or drop?)
- Should `carrel slot sync-all` auto-create the `APPEND_SYSTEM.md` slot if it doesn't exist and unwired append-system entries are found? (Recommended: yes, it's the only slot append-system entries go to.)
