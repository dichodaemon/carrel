---
name: validate-prompt
description: >
  Validate a prompt entry's content, naming, and cross-references.
  Use before `carrel config add prompt` or to audit existing prompts.
  Usage: /validate-prompt <name> [--source=<source-alias>]
---

# Validate Prompt

Validate a prompt entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered prompts.

## What to Validate

### Content structure

- The prompt file has a recognized format with template variables if
  parameterized.
- If using YAML frontmatter, it includes a `description` field.
- Template variables use a consistent syntax (e.g., `{{variable}}` or
  `${variable}`) and are all documented.
- The prompt body is non-empty and contains coherent prose.

### Naming conventions

- The prompt entry name is kebab-case.
- The name suggests the prompt's purpose (e.g., `code-review`,
  `commit-message`).

### Cross-references

- If the prompt references other prompts, those prompts exist.
- Template variable names do not collide with reserved OMP tokens.

## How to Validate

### Pre-registration (new prompt)

1. Verify the file path follows the source layout for prompts.
2. Check that the filename is kebab-case.
3. Parse frontmatter: confirm `description` is present.
4. Extract all template variables and verify they are consistently
   formatted.
5. Confirm the prompt body is non-empty and the template variables are
   all referenced in the body (no unused variables).

### Post-hoc audit (existing prompt)

1. Run `carrel config view prompt <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:prompt:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify that all template variables are still consistent with the
   prompt body.
5. Check that referenced prompts still exist.
