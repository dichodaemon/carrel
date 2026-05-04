## Workflow Principles

These govern how you interact with the user across all tasks.

### Propose before implementing

When the user says "propose", "suggest", or asks for options: present the proposal in chat and stop. Do not create files, edit code, or execute commands until the user approves. This is a deliberation phase -- the user is exploring, not committing.

### Explain before executing

Before starting any non-trivial work, state what you plan to do and which files you will touch. The user must understand the plan before you act on it. A brief numbered list is sufficient -- not a full plan document, just enough that the user can say "go" or "wait."

### Ask before committing

Do not run `git commit` or `git push` without explicit approval. You may ask once for the entire workflow ("I'll commit and push when done -- OK?") rather than per-commit, but the default is to stop and ask.

---

## Universal Dev Framework

`<FOLIO>` is the absolute path to the **folio** repository on this machine (`dichodaemon/folio` on GitHub). Folio manages documentation framework and workflow artifacts.

This workspace uses a documentation framework with strict format definitions and authoring standards. The authoritative source is:

```
<FOLIO>/
  doc-definitions/    Format and structure requirements for each document type
  standards/          Conventions governing documentation practices
  references/         Factual reference material
```

### Rules

- **Before authoring any document** (spec, plan, design study, cookbook, README, or reference), read the corresponding doc-definition. Do not guess at format -- the definitions are normative.
- **Before making structural decisions** about documentation (naming, placement, directory layout), read `standards/document-structure_standards.md`.
- **File naming follows a strict pattern**: `[YYYY-MM-DD_]<topic>_<doctype>.md`. The separator rule is: underscores between structural elements, hyphens within elements. The doctype suffix inventory is closed -- do not invent new suffixes without checking the standard.
- **Document placement is scope-driven**: component docs stay local, cross-cutting docs go in `docs/`, universal framework docs live in the folio repo root. See section 3 of the document structure standards.
- **Cookbooks are append-only**: add numbered entries at the end. Never reorder, renumber, or restructure existing entries.

---

## OMP Configuration Source of Truth

The `.omp/` directory in the target workspace is **deployed** by carrel's `carrel run` command. Never edit `.omp/` directly in a target repo -- changes will be overwritten by the next `carrel run` deployment.

Before creating, modifying, or deleting any OMP configuration (skills, rules, commands, extensions, agents, tools, hooks, prompts, instructions, or system prompt files), use `carrel <type> add|rm|edit` commands. Configuration lives in registered sources managed by the carrel registry.

---

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
