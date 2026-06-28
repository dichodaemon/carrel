---
name: validate-agent
description: >
  Validate an agent entry's content, naming, and cross-references.
  Use before `carrel config add agent` or to audit existing agents.
  Usage: /validate-agent <name> [--source=<source-alias>]
---

# Validate Agent

Validate an agent entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered agents.

## What to Validate

### Content structure

- The agent file has a recognized format (YAML, TOML, or JSON).
- Required fields are present: agent identity (name/role), model
  configuration, and system prompt or personality description.
- If the agent defines tool access, the tool list references valid tool
  names.
- The model field specifies a valid model identifier supported by the
  OMP backend.

### Naming conventions

- The agent entry name is kebab-case.
- The entry name is unique across all sources (no duplicate agent
  names in different sources unless intentional override).

### Cross-references

- Referenced tools are registered in the tool registry.
- Referenced skills or rules (if any) exist and are slotted.
- The agent does not reference itself or create circular chains.

## How to Validate

### Pre-registration (new agent)

1. Verify the file is at the correct source path.
2. Parse the file format: valid YAML/TOML/JSON.
3. Confirm all required fields are present and non-empty.
4. Validate that the model identifier is a known model.
5. Check that tool references resolve to existing tool entries.

### Post-hoc audit (existing agent)

1. Run `carrel config view agent <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:agent:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify all cross-referenced tools are still registered.
5. Confirm the model identifier is still valid (not deprecated).
