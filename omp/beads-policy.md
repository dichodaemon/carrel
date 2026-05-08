# Beads Policy

## Task Tracking (Beads)

The `bd` CLI (beads) is available for persistent, structured task tracking. Use it instead of ad-hoc markdown plans when tasks span multiple turns or involve dependencies. Do NOT use TodoWrite, TodoCreate, or markdown checklists.

Initialize with `bd init` in a project directory when starting work. The database is stored in `.beads/` (Dolt SQL) — gitignored. Run `bd prime` for full command reference and session close protocol.

Key commands:
```bash
bd ready                        # find unblocked work
bd create --title="Summary" --description="Context" --type=task --priority=0
bd update <id> --claim          # claim work
bd dep add <blocked> <blocker>  # add dependency
bd close <id>                   # complete work
```

### Database routing

Set `BEADS_DB` explicitly on every `bd` call based on which workspace the work targets:

| Workspace | BEADS_DB |
|---|---|
| `/workspace/carrel` | `/workspace/carrel/.beads` |
| `/workspace/folio` | `/workspace/folio/.beads` |
| `/workspace/oh-my-pi` | `/workspace/oh-my-pi/.beads` |

Usage: `BEADS_DB=<path> bd <command>`. When ambiguous, use the DB for the workspace where the code change lands.

## Issue Creation

Required fields: `--title`, `--description`, `--type`, `--priority` (0–4 integer). Descriptions must include target files, what to implement, and done conditions.

For batch creation (more than a handful of issues), use `bd create --graph <plan.json>`. Do not run individual `bd create` and `bd dep add` in a loop.

## Dependencies

| Type | When to use |
|---|---|
| `blocks` | A consumes types, interfaces, or files that B produces. |
| `discovered-from` | New work surfaces during implementation of the current issue. |
| `related` | Same area, no blocking. |
| `parent-child` | Multi-step criteria, epics grouping sub-tasks. |

### `discovered-from` protocol

When working on an issue and you discover unplanned work:
1. Create a new issue immediately.
2. Link: `bd link <new-id> <current-id> --type discovered-from`.
3. Continue working on the current issue.

Do not defer. The new issue captures context while it's fresh. The link preserves provenance.

### Arch-design deviations

If implementation needs to differ from the companion document: create a deviation issue and an arch-design update issue, both linked `discovered-from`. Wire the update as a blocker for any existing issue that depends on the changed contract.

## Memory

Use `bd remember` for persistent knowledge. Search with `bd memories <keyword>`. Do NOT use MEMORY.md files.

## Subagent Dispatch

When dispatching subagents (via `task` tool) for beads-tracked work, include in the shared `context`:
1. **`discovered-from` protocol** — subagents must create issues for unplanned work, not leave TODO comments.
2. **BEADS_DB path** for the target workspace.
3. **Current issue ID** so `discovered-from` links are wired correctly.
4. **Blocking relationships** the subagent should wire.

## Anti-Patterns

- TodoWrite or markdown checklists — use beads exclusively.
- Single mega-issue — one issue per acceptance criterion.
- One-line description — agent cannot execute without context.
- Closing without verifying acceptance criteria.
- Deferring issue creation — context lost.
- Memories in MEMORY.md — use `bd remember`.
- Omitting beads context from subagents.

## Session Completion

When ending a work session, complete ALL steps below. Work is NOT complete until `git push` succeeds.

1. File issues for remaining work — `bd create`
2. Run quality gates (if code changed) — tests, linters, builds
3. Update issue status — `bd close` finished work, update in-progress items
4. Push to remote:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. Verify — all changes committed AND pushed

## Closing Discipline

- Never close a bead unless the work is verifiably complete.
- If work cannot be completed, leave the bead open.
- Before closing, verify: code compiles, tests pass, git status shows intended changes.

---
