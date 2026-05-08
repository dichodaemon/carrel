---
title: Core Prompt Reorganization
date: 2026-05-08
author: Dizan Vasquez
---

# Core Prompt Reorganization

## 1. Objective

Split the monolithic `APPEND_SYSTEM.md` into per-aspect `append-system` entries so that carrel, folio, and beads policies are independently maintainable. The universal `APPEND_SYSTEM.md` retains only workflow principles; the three extracted aspects become separate entries concatenated at deployment by the existing slot.

## 2. Scope

| In scope | Out of scope |
|---|---|
| Split `carrel-omp:append-system:APPEND_SYSTEM.md` into four entries | Modifying the carrel binary or slot machinery |
| Create `carrel-policy`, `folio-policy`, `beads-policy` as new `append-system` entries | Changing the Folio or Beads tooling itself |
| Identify overlapping content in other config sources for later cleanup | Removing or updating those overlapping sources |
| Verify `carrel plan` shows correct concatenation | Changing any existing `carrel-configuration` rule content |

## 3. Sources

1. **Current APPEND_SYSTEM.md** — `/workspace/carrel/omp/APPEND_SYSTEM.md` (registered as `carrel-omp:append-system:APPEND_SYSTEM.md`)
2. **Carrel registry** — `/workspace/.carrel/registry`
3. **Slot** — `APPEND_SYSTEM.md` slot maps all `append-system` entries to deployment path `APPEND_SYSTEM.md` (1 entry currently)
4. **Carrel-configuration rule** — `carrel-omp:rule:carrel-configuration` at `/workspace/carrel/omp/rules/carrel-configuration.md`
5. **AGENTS.md** — `/workspace/carrel/AGENTS.md` (contains overlapping beads reference content; deployed by carrel for non-opt-in repos)

## 4. Approach

### 4.1. Current structure of APPEND_SYSTEM.md

| Lines | Section | Aspect |
|---|---|---|
| 1–16 | Workflow Principles | Universal |
| 19–39 | Universal Dev Framework | Folio |
| 42–46 | OMP Configuration Source of Truth | Carrel |
| 50–98 | Task Tracking (Beads) + Session completion + Closing discipline | Beads |

### 4.2. Target entries

| Entry name | Content | Source |
|---|---|---|
| `APPEND_SYSTEM.md` | Workflow Principles only (lines 1–16) | Existing, edited down |
| `carrel-policy` | OMP Configuration Source of Truth (lines 42–46), expanded with essential carrel usage pointers | New |
| `folio-policy` | Universal Dev Framework (lines 19–39) | New |
| `beads-policy` | Task Tracking + Session completion + Closing discipline (lines 50–98) | New |

### 4.3. Steps

1. Create three new `append-system` entries via `carrel config add` with content extracted from current APPEND_SYSTEM.md.
2. Edit the existing `APPEND_SYSTEM.md` entry to contain only the Workflow Principles section.
3. Run `carrel plan` to verify the four entries concatenate correctly and produce the intended output.
4. Run `carrel config scan carrel-omp` to resync registry hashes.

### 4.4. Overlapping content identified for later cleanup

| Source | Overlap | Action deferred |
|---|---|---|
| `AGENTS.md` (line 8–12: "Quick Reference") | Duplicates beads commands also in `beads-policy` | Trim after verifying beads-policy is deployed |
| `AGENTS.md` (line 27–31: "Beads Issue Tracker") | Duplicates beads rules and session completion | Trim after verifying beads-policy is deployed |
| `AGENTS.md` (line 39–49: "Session Completion" block) | Full duplicate of beads session completion steps | Trim after verifying beads-policy is deployed |
| `carrel-configuration` rule | Covers carrel mechanics in detail; `carrel-policy` would be a concise policy summary | No change needed; rule and policy serve different purposes |

## 5. Constraints

- **Carrel CRUD required.** All configuration changes must use `carrel config add|edit|rm` or direct edit + `carrel config scan`. Never hand-edit deployed `.omp/` files.
- **Slot is append-system.** The `APPEND_SYSTEM.md` slot concatenates all `append-system` entries. Order is determined by entry name lexicographic sort: `APPEND_SYSTEM.md` < `beads-policy` < `carrel-policy` < `folio-policy`. This places the universal section first, followed by the three aspects in alphabetical order — acceptable.
- **AGENTS.md is deploy-managed.** For non-opt-in repos, `carrel run` deploys `AGENTS.md`. Changes to overlapping content there must be coordinated with the `context-file` deployment path or handled separately.
- **Session restart required.** Changes to `append-system` entries take effect on next `carrel run` deployment.

## Appendix A: Implementation Plan

### Epic 1 — Split APPEND_SYSTEM.md (main)

| # | Task | Depends on |
|---|---|---|
| 1 | Create `carrel-policy` append-system entry from lines 42–46 of current APPEND_SYSTEM.md, expanded with essential carrel usage pointers | — |
| 2 | Create `folio-policy` append-system entry from lines 19–39 | — |
| 3 | Create `beads-policy` append-system entry from lines 50–98 | — |
| 4 | Edit `APPEND_SYSTEM.md` down to lines 1–16 (Workflow Principles only) | 1, 2, 3 |
| 5 | Run `carrel plan` and verify concatenation output | 4 |

### Epic 2 — Remove beads duplication from AGENTS.md (blocked by Epic 1)

| # | Task | Depends on |
|---|---|---|
| 1 | Remove "Quick Reference" beads commands (lines 8–12) from AGENTS.md | Epic 1 |
| 2 | Remove "Beads Issue Tracker" reference block (lines 27–31) from AGENTS.md | Epic 1 |
| 3 | Remove "Session Completion" block (lines 39–49) from AGENTS.md | Epic 1 |
| 4 | Verify `carrel plan` still produces valid AGENTS.md output | 1, 2, 3
