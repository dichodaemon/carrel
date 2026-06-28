---
name: validate-tool
description: >
  Validate a tool entry's content, naming, and cross-references.
  Use before `carrel config add tool` or to audit existing tools.
  Usage: /validate-tool <name> [--source=<source-alias>]
---

# Validate Tool

Validate a tool entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered tools.

## What to Validate

### Content structure

- The tool file defines a valid tool integration: name, description,
  and invocation specification.
- The tool schema (input parameters, output format) is complete and
  well-formed.
- Required fields: tool name, description, and command or endpoint.
- Optional fields (timeout, retry, environment variables) use correct
  types and valid values.

### Naming conventions

- The tool entry name is kebab-case.
- The name matches the tool's directory/filename if using a
  file-per-tool layout.
- The name is unique across all sources.

### Cross-references

- If the tool references other tools (composition), those tools exist.
- If the tool requires a binary or system dependency, that dependency
  is documented.
- The tool schema's parameter types are valid and consistent.

## How to Validate

### Pre-registration (new tool)

1. Verify the file path follows the source layout for tools.
2. Check that the entry name is kebab-case.
3. Parse the tool definition: confirm `name`, `description`, and
   invocation specification are present.
4. Validate the parameter schema: all parameter names are valid
   identifiers, types are recognized, and required vs. optional is
   clear.
5. Verify the command or endpoint is syntactically valid.

### Post-hoc audit (existing tool)

1. Run `carrel config view tool <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:tool:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the tool's binary or endpoint is reachable.
5. Check that composed/referenced tools are still registered.
