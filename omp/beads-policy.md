# Beads Policy

## Task Tracking (Beads)

The `bd` CLI (beads) is available for persistent, structured task tracking. Use it instead of ad-hoc markdown plans when tasks span multiple turns or involve dependencies.

Key commands:
- `bd ready` — list unblocked tasks
- `bd create "Title" -p 0` — create a P0 task
- `bd update <id> --claim` — claim a task
- `bd dep add <child> <parent>` — link tasks as blocking/related
- `bd show <id>` — view task details

Initialize with `bd init` in a project directory when starting work. The database is stored in `.beads/` (Dolt SQL) — gitignore it.

The `bv` CLI (beads viewer) provides a TUI and agent-mode triage engine. It reads `.beads/beads.jsonl` — run `bd export --no-memories -o .beads/beads.jsonl` to update before use.

Key commands:
- `bv` — interactive TUI (blocks session; use only when user requests)
- `bv --robot-triage` — single-call triage: recommendations, blockers, quick wins
- `bv --robot-next` — minimal: top pick + claim command

### Beads rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists.
- Run `bd prime` for detailed command reference and session close protocol.
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files. Search with `bd memories <keyword>`.

### Session completion

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


### Closing discipline

- **Never close a bead unless the work is done.** Closing means the deliverable is verifiably complete, not deferred, not "follow-up," not "assumed done." A bead is a contract — close it when the tests pass and the code is committed.
- **If work cannot be completed,** leave the bead open. Do not close with reasons like "tracked for follow-up" or "not blocking current milestone."
- **Before closing, verify:** (1) the code compiles, (2) tests pass for affected packages, (3) git status shows the intended changes.

---

