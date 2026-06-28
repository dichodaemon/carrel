---
name: validate-command
description: >
  Validate a command entry's content, naming, and cross-references.
  Use before `carrel config add command` or to audit existing commands.
  Usage: /validate-command <name> [--source=<source-alias>]
---

# Validate Command

Validate a command entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered commands.

## What to Validate

### Content structure

- The command file defines a slash-command with name, description, and
  handler specification.
- The command has a clear trigger pattern (e.g., `/command-name`).
- Arguments are documented: required vs. optional, types, and defaults.
- The command handler references a valid skill, tool, or inline script.
- The description is user-facing and explains what the command does.

### Naming conventions

- The command name is kebab-case (e.g., `create-plan`).
- The slash-command trigger matches the entry name.
- The name is unique across all sources.

### Cross-references

- The handler skill or tool is registered and slotted.
- If the command delegates to multiple skills, all are registered.
- Argument types reference valid OMP type names.

## How to Validate

### Pre-registration (new command)

1. Verify the file path follows the source layout for commands.
2. Check that the command name is kebab-case.
3. Parse the command definition: confirm `name`, `description`, and
   handler are present.
4. Verify the handler reference resolves to an existing skill or tool.
5. Validate argument definitions: names, types, required/optional, and
   defaults are consistent.
6. Confirm the description is user-facing and informative.

### Post-hoc audit (existing command)

1. Run `carrel config view command <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:command:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the handler skill/tool is still registered and slotted.
5. Check that argument types are still valid (no deprecated types).
