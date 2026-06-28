---
description: "Format, structure, naming, and best practices for append-system entries."
globs:
  - 'omp/*.md'
---

# Append-System Entry Type

Append-system entries are additive system prompt fragments that are concatenated across all configuration sources and injected into the agent's system prompt. Unlike other config types where deeper scopes override shallower scopes, append-system entries **compose** — every registered append-system entry from every source is included in the final output.

This makes append-system ideal for policies, conventions, and behavioral guidance that should accumulate across the configuration hierarchy rather than be replaced.

## Format

- **File extension:** `.md` (Markdown)
- **Location:** `omp/<name>.md` — top-level in the source directory (not in a subdirectory like `omp/rules/`)
- **Frontmatter:** None required. Append-system files are plain markdown injected verbatim.

## Naming Conventions

- **kebab-case:** `carrel-policy.md`, `beads-policy.md`, `folio-policy.md`
- **`-policy` suffix for behavioral policies:** `carrel-policy.md`, `beads-policy.md`
- **Descriptive of content:** `workflow-principles.md`, `shell-guidelines.md`, `security-policy.md`
- **Avoid generic names:** `system.md`, `appendix.md`, `extra.md`

## Content Best Practices

- **Designed for concatenation.** Each file MUST be self-contained — it will be concatenated with other append-system files in deployment order. Don't start with "Continuing from above…" or assume proximity to other content.
- **Use `---` horizontal rules as separators.** Each append-system file SHOULD end with `---` (or start with one) to visually separate it from adjacent content:
  ```markdown
  ## Workflow Principles
  ...
  ---
  ```
- **Section-based organization.** Use H2 (`##`) sections for major topics. Agents navigate system prompt content by heading.
- **Policy content, not reference material.** Append-system is for behavioral guidance (what to do, how to interact). Reference material (API docs, style guides) belongs in `instruction` entries.
- **RFC 2119 keywords** for binding directives: `MUST`, `MUST NOT`, `SHOULD`, `SHOULD NOT`, `MAY`.
- **Concise.** System prompt real estate is finite. Every paragraph in an append-system file consumes context window budget. Be economical.
- **Avoid overlapping with rules.** If the same constraint exists as a rule, reference the rule rather than duplicating. Append-system is for cross-cutting policies that don't fit the rule model.
- **Scope awareness.** Since append-system accumulates from all sources (universal + target-specific + local), be aware that your content will appear alongside other append-system entries. Avoid contradicting universal policies.

## Deployment Behavior

1. **Registration:** `carrel config add append-system <name> --file=omp/<name>.md`
2. **Slot wiring:** `carrel slot sync-all` — typically wired to the `APPEND_SYSTEM.md` slot
3. **Deployment:** `carrel run` composes all append-system entries from all active sources into `.omp/APPEND_SYSTEM.md`
4. **Composition:** Entries are concatenated in source-priority order: universal sources first, then target-specific, then local overrides. Within each source, alphabetical by name.
5. **Injection:** OMP reads `.omp/APPEND_SYSTEM.md` at session startup and injects it into the system prompt.

### Composition vs. Override

This is the key difference from other config types:

- **Rule, skill, agent, etc.:** deeper scope overrides shallower scope. A target-specific rule with the same name as a universal rule replaces it.
- **Append-system:** all entries from all scopes are concatenated. A target-specific append-system entry does NOT replace a universal one — it is appended after it.

This means append-system is *additive only*. To remove a universal policy from a specific target, you must modify the slot wiring, not add a same-named entry.

## Validation

Use the `validate-append-system` skill to check:
- File is valid Markdown
- Content is non-empty and substantive
- No frontmatter (append-system files are raw markdown)
- File is at the top level of the source directory (`omp/<name>.md`), not in a subdirectory
- No duplicate append-system entry with the same name exists in the same source
- Cross-references to rules or skills are resolvable

## Example

```markdown
## Workflow Principles

These govern how you interact with the user across all tasks.

### Propose before implementing

When the user says "propose", "suggest", or asks for options: present the proposal
in chat and stop. Do not create files, edit code, or execute commands until the
user approves. This is a deliberation phase — the user is exploring, not committing.

### Explain before executing

Before starting any non-trivial work, state what you plan to do and which files
you will touch. The user must understand the plan before you act on it.

### Ask before committing

Do not run `git commit` or `git push` without explicit approval. You may ask once
for the entire workflow rather than per-commit, but the default is to stop and ask.
```
