---
name: validate-instruction
description: >
  Validate an instruction entry's content, naming, and cross-references.
  Use before `carrel config add instruction` or to audit existing instructions.
  Usage: /validate-instruction <name> [--source=<source-alias>]
---

# Validate Instruction

Validate an instruction entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered
instructions.

## What to Validate

### Content structure

- The instruction file is valid Markdown with optional YAML frontmatter.
- The body contains instructional prose — directives, guidance, or
  procedural steps for the agent.
- If frontmatter is present, it includes a `description` field
  summarizing the instruction's purpose.
- The instruction does not overlap with a skill: instructions are
  passive (injected content), not interactive workflows.

### Naming conventions

- The instruction entry name is kebab-case.
- The filename follows the source convention for instructions.

### Cross-references

- If the instruction references rules or skills, those entries exist
  and are registered.
- The instruction does not duplicate content from another instruction
  with the same scope.

## How to Validate

### Pre-registration (new instruction)

1. Verify the file path follows the source layout for instructions.
2. Check that the filename is kebab-case.
3. Parse frontmatter (if present): confirm valid YAML and `description`
   field.
4. Read the body: ensure it contains substantive instructional content,
   not just a heading.
5. Check for references to rules/skills and verify they exist.

### Post-hoc audit (existing instruction)

1. Run `carrel config view instruction <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:instruction:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify referenced rules and skills are still valid.
5. Check for content drift between source and deployed copies via
   `carrel deployed --type=instruction`.
