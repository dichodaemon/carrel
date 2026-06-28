---
name: validate-skill
description: >
  Validate a skill entry's content, naming, and cross-references.
  Use before `carrel config add skill` or to audit existing skills.
  Usage: /validate-skill <name> [--source=<source-alias>]
---

# Validate Skill

Validate a skill entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered skills.

## What to Validate

### Content structure

- The skill lives in its own directory: `omp/skills/<name>/SKILL.md`.
- Frontmatter includes `name` (matching the directory name exactly) and
  `description` (ending with `Usage: /<name> [args]` for non-trivial
  skills).
- The body opens with an H1 heading (`# <Title>`) in human-readable
  form.
- The body includes a one-paragraph purpose statement after the H1.
- Procedural and complex skills include a `## Trigger` section listing
  natural-language invocation phrases.
- Complex skills include `## Arguments`, phased `## Phase N:` sections
  with `### Step N:` subsections, and a `## Rules` section.
- All referenced `carrel` commands use correct syntax.

### Naming conventions

- The skill name is kebab-case (e.g., `create-plan`).
- The skill directory name matches the `name` field in frontmatter
  exactly.
- The skill's filename is always `SKILL.md` (case-sensitive).

### Cross-references

- If the skill references other skills, those skills are registered.
- If the skill references rules, those rules exist in the registry.
- If the skill invokes `carrel config add <type>`, the type is a valid
  configuration type listed in `carrel config types`.
- All internal cross-references (step numbers, phase names) are
  consistent.

## How to Validate

### Pre-registration (new skill)

1. Verify the directory path: `omp/skills/<name>/` with a `SKILL.md`
   inside.
2. Confirm the directory name is kebab-case and matches the `name`
   field in frontmatter.
3. Parse frontmatter: `name` and `description` are present; `name` is
   kebab-case.
4. Check the body has an H1 heading and a purpose paragraph.
5. Confirm the complexity tier is appropriate: check whether
   `## Trigger`, `## Arguments`, `## Phase`, `## Rules` sections are
   present or absent consistently.
6. Scan for `carrel` commands and verify they use valid subcommands and
   type names.

### Post-hoc audit (existing skill)

1. Run `carrel config view skill <name> --meta-only` and confirm
   registration with a valid hash.
2. Run `carrel trace carrel-omp:skill:<name>` to verify slot wiring.
3. Compare on-disk content hash with the registry hash.
4. Run `carrel config list skill` and confirm the skill appears.
5. Verify the skill is slotted by checking `carrel slot list` for an
   entry referencing `<source>:skill:<name>`.
6. Check that the skill's frontmatter `name` still matches the
   directory name (no drift after renames).
