---
name: validate-slot-defaults
description: >
  Validate a slot-defaults entry's content, naming, and cross-references.
  Use before `carrel config add slot-defaults` or to audit existing slot defaults.
  Usage: /validate-slot-defaults <name> [--source=<source-alias>]
---

# Validate Slot Defaults

Validate a slot-defaults entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered
slot default configurations.

## What to Validate

### Content structure

- The file is valid YAML following the slot defaults schema.
- The top-level key is `contributions`, which maps to a list of slot
  contribution objects.
- Each contribution has a `slot` key (the slot name) and an `entries`
  list.
- Each entry in `entries` uses the format `<type>:<name>` (e.g.,
  `rule:carrel-configuration`, `skill:create-plan`).
- Slot names are kebab-case and unique within the file.

### Naming conventions

- The entry name is kebab-case.
- The filename follows the slot defaults naming convention
  (e.g., `slot-defaults.yml`).
- The `slot` keys use valid slot names that exist or will be created.

### Cross-references

- Every entry reference (`<type>:<name>`) resolves to a registered
  entry of the correct type.
- Referenced types are valid carrel configuration types.
- No duplicate entry references across overlapping slot contributions.
- The slot defaults do not reference entries that have been removed
  from the registry (dangling references).

## How to Validate

### Pre-registration (new slot-defaults)

1. Verify the file path follows the source layout for slot defaults
   (e.g., `omp/slot-defaults.yml`).
2. Parse the YAML: confirm it is syntactically valid.
3. Verify the `contributions` key is present and is a list.
4. For each contribution, check that `slot` and `entries` are present.
5. Validate each entry reference format: `<type>:<name>` with valid
   type and kebab-case name.
6. Cross-check all referenced entries exist in the registry or on disk.

### Post-hoc audit (existing slot-defaults)

1. Run `carrel config view slot-defaults <name> --meta-only` and
   confirm registration.
2. Run `carrel trace <source>:slot-defaults:<name>` to verify
   deployment.
3. Compare on-disk hash with the registry hash.
4. Run `carrel slot list` and cross-reference: every entry in the
   defaults must have a corresponding slot.
5. Check for dangling references: run `carrel config list <type>` for
   each referenced type and confirm the named entry exists.
6. Verify no duplicate slot contributions exist across sources for the
   same consumer.
