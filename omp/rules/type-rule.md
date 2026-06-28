---
description: >
  Format, structure, naming, and best practices for rule entries.
  Rules govern agent behavior — conventions, constraints, and prohibitions.
  Use `carrel config add rule <name> --file=omp/rules/<name>.md` to register.
  Validate with `validate-rule` skill before registering.
globs:
  - 'omp/rules/*.md'
---

# Rule Entry Type

Rules are the primary constraint mechanism in the Scriptorium ecosystem. They define behavioral boundaries for agents — what they MUST do, MUST NOT do, SHOULD prefer, and SHOULD avoid. Rules are loaded at OMP session startup and injected into the system prompt.

## Format

- **File extension:** `.md` (Markdown with YAML frontmatter)
- **Location:** `omp/rules/<name>.md`
- **Frontmatter fields:**
  - `description` (required) — YAML folded scalar summarizing the rule's intent and registration command. End with `Validate with validate-rule skill before registering.`
  - `globs` (optional) — file patterns that trigger this rule's injection. Rules with `globs` are only active when the working directory or loaded files match.
  - `ttsr_trigger` (optional) — tool name that triggers this rule when the agent attempts to use it.

## Naming Conventions

- **kebab-case:** `no-push-oh-my-pi`, `carrel-configuration`, `no-todo-write`
- **Descriptive and imperative:** names should convey what the rule enforces or prohibits
- **Prefix conventions (informal):**
  - `no-` for prohibitions: `no-todo-write`, `no-push-oh-my-pi`
  - Bare noun phrases for informational/contextual rules: `carrel-configuration`
  - Verb-first for required actions: `use-bd-for-tracking`

## Content Best Practices

- **Lead with the constraint.** Put the MUST/MUST NOT/SHOULD/NEVER directive in the first paragraph. Agents scan rules quickly; burying the constraint in prose defeats the purpose.
- **Use RFC 2119 keywords** consistently: `MUST`, `MUST NOT`, `SHOULD`, `SHOULD NOT`, `MAY`. Define `NEVER` = `MUST NOT` and `AVOID` = `SHOULD NOT` if needed.
- **Be specific.** "Be careful with files" is not a rule. "NEVER edit `.omp/` directly — it is deployed output" is a rule.
- **Include rationale** only when the constraint is non-obvious. The rule is the primary artifact; rationale supports it, not the other way around.
- **Keep rules focused.** One rule per file. If you find yourself writing "Also, …" or "Additionally, …", split it into a separate rule.
- **Reference related rules** by their path (e.g., `See also omp/rules/carrel-configuration.md`).

## Deployment Behavior

1. **Source registration:** `carrel config add rule <name> --file=omp/rules/<name>.md`
2. **Slot wiring:** `carrel slot sync-all` links the rule to deployment slots
3. **Deployment:** `carrel run` composes and deploys to the target's `.omp/rules/` directory
4. **Injection:** OMP reads deployed rules at session startup and injects them into the system prompt under `<domain-rules>` or `<rules>` tags

Rules with `globs` are only injected when the glob matches the current context. Rules with `ttsr_trigger` fire only when the named tool is about to be invoked.

## Validation

Before registering a rule, run `carrel config validate-rule <name>` (or the `validate-rule` skill) to check:
- Frontmatter is valid YAML
- `description` field is present and non-empty
- `globs` patterns are syntactically valid
- File is at the correct path under `omp/rules/`
- No duplicate rule with the same name exists

## Example

```markdown
---
description: >
  NEVER run `git push` in the oh-my-pi repository.
  The user must explicitly authorize every push to oh-my-pi.
  Validate with validate-rule skill before registering.
globs:
  - '**/oh-my-pi/**'
---

# No Push to oh-my-pi

**NEVER** run `git push` in the oh-my-pi repository (`/workspace/oh-my-pi`).
The user must explicitly authorize every push to oh-my-pi.

This applies to all sessions deployed by carrel.
```
