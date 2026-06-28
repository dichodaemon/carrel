---
name: validate-rule
description: >
  Validate a rule entry's content, naming, and cross-references.
  Use before `carrel config add rule` or to audit existing rules.
  Usage: /validate-rule <name> [--source=<source-alias>]
---

# Validate Rule

Validate a rule entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered rules.

## What to Validate

### Content structure

- The rule file has valid YAML frontmatter delimited by `---` fences.
- Frontmatter includes a `description` field — a one-line summary of
  what the rule governs.
- If the rule scopes to specific file patterns, frontmatter includes a
  `globs` field with at least one glob pattern.
- The body after frontmatter is non-empty and contains the rule's
  directives in prose.
- The rule uses RFC 2119 keywords (`MUST`, `SHOULD`, `MAY`, `NEVER`,
  `AVOID`) consistently where applicable.

### Naming conventions

- The rule filename is kebab-case (e.g., `no-todo-write.md`).
- The filename matches the registry entry name.
- The rule lives under `omp/rules/` (universal) or
  `<target>/.carrel/rules/` (target-specific).

### Cross-references

- If the rule references other rules, the referenced rule exists in the
  registry or on disk.
- If the rule references a skill, the skill is registered and slotted.
- No circular dependency chains (rule A → rule B → rule A).

## How to Validate

### Pre-registration (new rule)

1. Verify the file is at the correct path: `omp/rules/<name>.md` or
   `<target>/.carrel/rules/<name>.md`.
2. Check that the filename is kebab-case and the `.md` extension is
   present.
3. Parse the frontmatter: confirm `description` is present and
   non-empty.
4. If `globs` is present, verify each glob pattern is syntactically
   valid (no unmatched braces, valid characters).
5. Read the body: ensure it is non-empty and contains actionable
   directives, not just a title.
6. Check for broken cross-references by grepping for rule/skill names
   mentioned in prose.

### Post-hoc audit (existing rule)

1. Run `carrel config view rule <name> --meta-only` and confirm the
   entry is registered with a valid hash.
2. Run `carrel trace carrel-omp:rule:<name>` to verify the rule is
   slotted and deployed to at least one consumer.
3. Compare the on-disk content against the registry hash: re-hash the
   file and confirm it matches.
4. Check that the rule appears in `carrel config list rule`.
5. Verify deployed copies match the source by running
   `carrel deployed --type=rule` and checking for consistency.
