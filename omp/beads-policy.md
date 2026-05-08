# Beads Policy

## Task Tracking (Beads)

The `bd` CLI (beads) is available for persistent, structured task tracking. Use it instead of ad-hoc markdown plans when tasks span multiple turns or involve dependencies. Do NOT use TodoWrite, TodoCreate, or markdown checklists.

Initialize with `bd init` in a project directory when starting work. The database is stored in `.beads/` (Dolt SQL) — gitignored. Run `bd prime` for full workflow context and session close protocol.

### Key commands

```bash
bd ready                        # find unblocked work
bd create --title="Summary" --description="Context" --type=task --priority=0
bd update <id> --claim          # claim work
bd dep add <blocked> <blocker>  # add dependency
bd close <id>                   # complete work
bd show <id>                    # view issue details
```

### Database routing

Set `BEADS_DB` explicitly on every `bd` call based on which workspace the work targets:

| Workspace | BEADS_DB |
|---|---|
| `/workspace/carrel` | `/workspace/carrel/.beads` |
| `/workspace/folio` | `/workspace/folio/.beads` |
| `/workspace/oh-my-pi` | `/workspace/oh-my-pi/.beads` |

Usage: `BEADS_DB=<path> bd <command>`. When ambiguous, use the DB for the workspace where the code change lands.

### The `bv` CLI

`bv` (beads viewer) provides a TUI and agent-mode triage engine. It reads `.beads/beads.jsonl` — run `bd export --no-memories -o .beads/beads.jsonl` to update before use.

```bash
bv                   # interactive TUI (blocks session; use only when user requests)
bv --robot-triage    # single-call triage: recommendations, blockers, quick wins
bv --robot-next      # minimal: top pick + claim command
```

## Issue Creation

### Required fields

- `--title` — 5–10 words describing what is being done, not how.
- `--description` — implementation context. Must include: target files (exact paths), what to implement, done conditions.
- `--type` — `task`, `bug`, `feature`, `epic`, or `chore`.
- `--priority` — 0–4 (0=critical, 2=medium, 4=backlog). Integers, not words.
- `--acceptance` — testable done conditions (required for `task` and `feature`).

### Description quality

An issue description must contain enough context for an agent to execute the task without reading the full companion document. Include:

- **Target files** — which files to create or modify. Exact paths.
- **What to implement** — the specific contract, type, or behavior.
- **Done conditions** — what the agent verifies before closing.
- **Edge cases** — non-obvious failure modes or boundary conditions (when applicable).
- **Dependencies** — what this issue consumes that another issue produces (when applicable).

### Supplementary fields

- `--design` — implementation-level decisions (build system quirks, test fixture choices).
- `--deps` — dependencies in `type:id` format. Add at creation when known.

### Batch creation

When creating more than a handful of issues, use `bd create --graph <plan.json>` to create the full issue graph (issues + dependencies) in a single operation. Do not run individual `bd create` and `bd dep add` commands in a loop.

## Dependencies

| Type | Meaning | When to use |
|---|---|---|
| `blocks` | B must complete before A can start. | A consumes types, interfaces, or files that B produces. |
| `discovered-from` | B was discovered while working on A. | New work surfaces during implementation. |
| `related` | Soft connection, no blocking. | Issues touch the same area but neither blocks the other. |
| `parent-child` | Hierarchical. | Multi-step criteria, epics grouping sub-tasks. |

### `discovered-from` protocol

When working on an issue and you discover unplanned work:

1. Create a new issue immediately.
2. Link it: `bd link <new-id> <current-id> --type discovered-from`.
3. Continue working on the current issue.

Do not defer. The new issue captures context while it's fresh. The link preserves provenance.

### Arch-design deviations

If the implementation needs to differ from the companion document:

1. Create a deviation issue for the code change. Link with `discovered-from`.
2. Create an arch-design update issue for updating the document. Link with `discovered-from`.
3. Wire the update as a blocker for any existing issue that depends on the changed contract.

## Memory

Use `bd remember` for persistent knowledge. Search with `bd memories <keyword>`. Do NOT use MEMORY.md files.

```bash
bd remember "pre-commit hooks fail; always use --no-verify" --key hooks
bd recall hooks                # retrieve by key
bd memories "pre-commit"       # search by keyword
```

**When to remember:** environment quirks, implementation decisions too granular for companion docs but too important to forget, corrections to wrong assumptions.

**When NOT to remember:** information already in a companion document (don't duplicate), temporary state (use `bd update --claim`).

## Hygiene

| Command | Purpose | When |
|---|---|---|
| `bd lint` | Check for missing description sections. | Before starting work on an issue. |
| `bd stale` | Find issues with no recent activity. | At session start. |
| `bd preflight` | Pre-PR checks (lint, stale, orphans). | Before opening a PR. |
| `bd orphans` | Find broken dependency references. | After closing a batch of issues. |

## Subagent Dispatch

When dispatching subagents (via `task` tool) for beads-tracked work, the orchestrator **MUST** include in the shared `context` field:

1. **`discovered-from` protocol** — subagents that encounter unplanned work, missing prerequisites, or arch-design deviations must create a beads issue and link it `discovered-from` the current issue. They must NOT leave TODO comments, skip the work, or defer it informally.
2. **BEADS_DB path** for the target workspace, so subagents can run `bd create` and `bd link`.
3. **Current issue ID** the subagent is working on, so `discovered-from` links are wired correctly.
4. **Blocking relationships** the subagent should wire if the discovered work blocks downstream issues.

## Anti-Patterns

- **TodoWrite or markdown checklists** — use beads exclusively.
- **Single mega-issue** — cannot parallelize, dependencies invisible. One issue per acceptance criterion.
- **One-line description** — agent cannot execute without reading the full companion document.
- **Closing without verifying** — acceptance criteria unchecked. Verify before `bd close`.
- **Deferring issue creation** — context lost. Create immediately.
- **Memories in MEMORY.md** — use `bd remember`.
- **Omitting beads context from subagents** — subagent has no way to follow protocol it was never told about.

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

- **Never close a bead unless the work is done.** Closing means the deliverable is verifiably complete, not deferred, not "follow-up."
- **If work cannot be completed,** leave the bead open. Do not close with reasons like "tracked for follow-up."
- **Before closing, verify:** (1) code compiles, (2) tests pass for affected packages, (3) git status shows intended changes.

---

