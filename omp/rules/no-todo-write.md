---
ttsr_trigger: "todo_write"
---

Do not use the `todo_write` tool. Use `bd` (beads) for all task tracking instead.

Key `bd` commands:
- `bd create --title="Summary" --description="Context" --type=task --priority=0`
- `bd update <id> --claim`
- `bd close <id>`
- `bd ready` to find unblocked work
- `bd remember` for persistent notes

See the Beads Policy (injected into the system prompt) for the full reference, including
BEADS_DB routing, dependency management, issue creation conventions, and session completion workflow.
