---
name: validate-extension
description: >
  Validate an extension entry's content, naming, and cross-references.
  Use before `carrel config add extension` or to audit existing extensions.
  Usage: /validate-extension <name> [--source=<source-alias>]
---

# Validate Extension

Validate an extension entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered
extensions.

## What to Validate

### Content structure

- The extension file has a recognized format (YAML, TOML, or JSON).
- Required fields are present: name, version, entry point, and
  description.
- The entry point references a valid executable or module path.
- The version follows semantic versioning (MAJOR.MINOR.PATCH).
- Dependencies on other extensions or system packages are declared.

### Naming conventions

- The extension entry name is kebab-case.
- The name is unique across all sources.
- The directory layout (if multi-file) follows the extension
  convention.

### Cross-references

- Declared extension dependencies exist and are registered.
- The extension does not create circular dependency chains.
- If the extension provides hooks, those hook names are valid and
  follow hook naming conventions.

## How to Validate

### Pre-registration (new extension)

1. Verify the file path follows the source layout for extensions.
2. Check that the entry name is kebab-case.
3. Parse the extension manifest: confirm `name`, `version`, `entry`,
   and `description` are present.
4. Validate the version string is semver-compliant.
5. Verify the entry point path is reachable relative to the extension
   root.
6. Check declared dependencies against the registry.

### Post-hoc audit (existing extension)

1. Run `carrel config view extension <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:extension:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify all dependencies are still registered.
5. Check that the entry point file still exists on disk.
