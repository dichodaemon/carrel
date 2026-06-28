---
description: "Format, structure, naming, and best practices for instruction entries."
globs:
  - 'omp/instructions/*.md'
---

# Instruction Entry Type

Instructions provide procedural guidance, reference material, or domain knowledge that is injected into agent sessions. Unlike rules (which constrain behavior) and skills (which are activated on demand), instructions are passive content that agents can reference during any task.

## Format

- **File extension:** `.md` (Markdown)
- **Location:** `omp/instructions/<name>.md`
- **Frontmatter:** None required. Instructions are plain markdown. Optional frontmatter fields:
  - `description` — one-line summary for registry display
  - `priority` — load order when multiple instructions exist (lower = loaded first; default 50)

## Naming Conventions

- **kebab-case:** `python-style-guide`, `git-workflow`, `testing-standards`
- **Descriptive of content:** name should clearly indicate what domain the instruction covers
- **Avoid generic names:** `instructions.md`, `guide.md`, `notes.md` are too vague

## Content Best Practices

- **Topic-focused.** Each instruction file covers one domain or workflow. Don't create a monolithic reference document — split into focused files.
- **Structure with headings.** Use H2 (`##`) sections to organize content. Agents navigate instructions by heading.
- **Include examples.** Abstract guidance is less useful than concrete examples. Show, don't just tell.
- **Version-sensitive content.** If the instruction references external tools or APIs, note the version it was written for.
- **Cross-reference related instructions.** Link to other instruction files by name when relevant.
- **Keep instructions current.** Unlike skills (which encode stable workflows), instructions may need updates as tools and conventions evolve. Review periodically.
- **Avoid duplicating rules.** If a point is enforced by a rule, the instruction should reference the rule, not repeat it.

## Deployment Behavior

1. **Registration:** `carrel config add instruction <name> --file=omp/instructions/<name>.md`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys to `.omp/instructions/<name>.md`
4. **Injection:** OMP injects instructions into the system prompt under `<instructions>` at session startup. They are available for agent reference throughout the session.

Instructions are loaded once at session startup. Changes require a session restart to take effect.

## Validation

Use the `validate-instruction` skill to check:
- File is valid Markdown
- No structural issues (broken links, malformed tables)
- Content is non-empty and substantive
- No duplicate instruction with the same name exists
- Cross-references to other instructions are resolvable

## Example

```markdown
# Python Style Guide

## Imports

- Group imports: standard library → third-party → local
- Use absolute imports for production code
- `import typing as t` for type annotations

## Type Annotations

- All public functions MUST have type annotations
- Use `| None` (Python 3.10+) over `Optional[...]`
- Use `collections.abc` generics for parameters, concrete types for returns

## Docstrings

- Google-style docstrings for all public APIs
- First line: one-line summary
- Sections: `Args:`, `Returns:`, `Raises:`

See also: `testing-standards` instruction for test naming conventions.
```
