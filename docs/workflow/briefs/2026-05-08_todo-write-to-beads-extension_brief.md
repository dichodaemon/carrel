---
title: todo_write → Beads Translation Extension
date: 2026-05-08
author: Dizan Vasquez
---

# todo_write → Beads Translation Extension

## 1. Objective

Build an OMP extension that transparently translates `todo_write` tool calls into `bd` (beads) commands, so the model can use its native task-tracking behavior while beads remains the authoritative system of record. No model retraining or prompt cajoling required.

## 2. Scope

| In scope | Out of scope |
|---|---|
| Intercept `todo_write` calls in the main agent and subagents | Modifying the OMP binary or built-in tool schema |
| Translate all `todo_write` operations (`init`, `start`, `done`, `append`, `drop`, `rm`, `note`) to equivalent `bd` commands | Translating TodoCreate or markdown checklist behavior |
| Store task-to-bead identity as `[todo-ref: ...]` markers in bead descriptions | Two-way sync (beads → todo_write state) |
| Handle `BEADS_DB` routing by workspace | Extending `bd` CLI with new subcommands |
| Deploy as a universal carrel source so all consumers inherit it | Custom `todo_write` UI rendering changes |

## 3. Sources

1. `omp/beads-policy.md` — current beads usage policy and BEADS_DB routing table
2. `docs/references/omp-configuration_reference.md` — extension API, tool_call/tool_result events, extension lifecycle (section 2.1)
3. `docs/references/omp-subagent-delegation_reference.md` — extension API events detail (section 9.2), subagent tool set construction (section 5.3)
4. `docs/references/beads-formulas_reference.md` — `bd` command reference
5. `docs/standards/beads_standards.md` — beads usage conventions
6. `.omp/rules/carrel-configuration.md` — carrel extension deployment via slots

## 4. Approach

### 4.1. Extension Architecture

The extension hooks two events:

1. **`tool_call`** — intercepts `todo_write` before execution. Parses the parameters, translates to `bd` commands, executes them, blocks the original tool call.
2. **`tool_result`** — (fallback) if blocking is insufficient to return a synthetic success, modify the result to indicate translation occurred.

### 4.2. Translation Mapping

| `todo_write` op | Beads equivalent | Notes |
|---|---|---|
| `init` | (1) Close all open beads matching `description=todo-ref:` via `bd query` + `bd close`. (2) `bd create` per task item with `[todo-ref: <task-content>]` in the description. (3) Apply phase labels for grouping. | Replaces the entire task list on each call. |
| `start` | `bd update <id> --claim` | Finds bead by `bd search --desc-contains "todo-ref: <task-content>" --status open --json`. |
| `done` (task) | `bd close <id>` | Same description-based lookup. |
| `done` (phase) | `bd close <id>` per task in phase | Finds all open beads with the phase label, closes each. |
| `append` | `bd create` per new item | Appends new beads under the given phase label. |
| `drop` (task) | No-op or `bd remember` with context | Beads have no "dropped" state. Leave open or annotate. |
| `rm` (task) | Same as `drop` | |
| `rm` (no task/phase) | `bd close` all open `todo-ref:` beads | Equivalent to clearing the todo list. |
| `note` | `bd remember` | Appends note text as a persistent memory. |

### 4.3. Task-to-Bead Identity

Todo tasks are identified by content string (`"Scaffold crate"`); beads are identified by opaque IDs (`abc123`). Instead of an external state file, the mapping lives inside beads itself.

Each bead created by `todo_write init` carries a marker in its description:

```
[todo-ref: Scaffold crate]
```

All lookups use `bd search --desc-contains "todo-ref: <task-content>" --status open --json`. The marker is stable, queryable via standard `bd` commands, and survives session restarts.

**`init` semantics:** `todo_write init` replaces the entire task list. On each `init`, the extension first closes all open beads matching `description=todo-ref:` (via `bd query "description=todo-ref:" --json` + `bd close`), then creates fresh beads. This prevents stale or duplicate entries.

Closed beads retain their `[todo-ref: ...]` markers — this is provenance, not noise. You can trace which todo task spawned each bead.

### 4.4. BEADS_DB Routing

The extension resolves `BEADS_DB` by inspecting `pi.cwd`:

| cwd prefix | BEADS_DB |
|---|---|
| `/workspace/carrel` | `/workspace/carrel/.beads` |
| `/workspace/folio` | `/workspace/folio/.beads` |
| `/workspace/oh-my-pi` | `/workspace/oh-my-pi/.beads` |

All `bd` commands run with `BEADS_DB=<path>` set in the environment. If `cwd` doesn't match any known workspace, the extension logs a warning and passes through the original `todo_write` call.

### 4.5. Implementation Steps

1. Create extension module at `omp/extensions/todo-beads-bridge/index.ts`
2. Implement `tool_call` handler with translation logic and description-based lookups
3. Register as a carrel universal source via `carrel config add extension`
4. Wire to the `extensions` slot for deployment
5. Test with representative `todo_write` call sequences
6. Verify beads database state after each operation

## 5. Constraints

- **Extension runs in-process (no sandbox).** `bd` subprocess invocations are serial per `todo_write` call.
- **tool_call handlers are fail-closed.** An unhandled exception in the handler blocks `todo_write` entirely — the model sees an error. Error handling must be defensive.
- **Tool names must be globally unique.** Cannot shadow the built-in `todo_write` with a custom tool of the same name.
- **Discovery is at startup only.** Extension changes require OMP restart.
- **Subagents inherit extensions.** The translation must work correctly in subagent sessions where `pi.cwd` may differ from the parent.
- **`bd` must be installed** in the execution environment. The extension fails gracefully with a clear error if `bd` is not on `PATH`.

## 6. Open Questions

1. **Can a `tool_call` handler return a synthetic success result?** The documented API supports `{ block: true }` to prevent execution. Whether the handler can also inject a synthetic tool result (so the model sees success instead of an error) needs verification during implementation. If not, the extension will also hook `context` to inject a system note explaining the translation.
2. **What is the OMP extension API surface for `tool_call` handlers?** The reference docs describe the concept but not the full TypeScript interface. Implementation will require inspecting the runtime types or reading OMP source.
3. **Should `bd init` be called automatically if `.beads/` doesn't exist?** The current policy says `bd init` in a project directory when starting work. The extension could auto-init, but that changes the contract. Deferring to the policy for now.
