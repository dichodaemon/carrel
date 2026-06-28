---
name: validate-hook
description: >
  Validate a hook entry's content, naming, and cross-references.
  Use before `carrel config add hook` or to audit existing hooks.
  Usage: /validate-hook <name> [--source=<source-alias>]
---

# Validate Hook

Validate a hook entry for structural correctness and completeness.
Works for pre-registration checks (before `carrel config add`) and
post-hoc audits of already-registered hooks.

## What to Validate

### Content structure

- The hook file is an executable script (shell, Python, or other
  supported runtime) or a declarative config referencing an executable.
- The hook has a clear lifecycle phase: `pre-session`, `post-session`,
  `pre-command`, or `post-command`.
- If a shell script, it has a shebang line and is marked executable.
- The hook does not block indefinitely or require interactive input.
- Error handling is present: the hook exits with a non-zero code on
  failure.

### Naming conventions

- The hook entry name is kebab-case.
- The name indicates when the hook fires (e.g., `pre-commit`,
  `post-session-cleanup`).

### Cross-references

- If the hook invokes other hooks, tools, or carrel commands, those
  dependencies are documented and available.
- The hook does not create circular chains with other hooks.

## How to Validate

### Pre-registration (new hook)

1. Verify the file path follows the source layout for hooks.
2. Check that the filename is kebab-case.
3. Confirm the lifecycle phase is specified and valid.
4. If a script: verify the shebang is correct and the file is
   executable.
5. Run a static check: does the hook contain any blocking or
   interactive constructs (`read`, `input()`, infinite loops)?
6. Verify exit codes are used appropriately.

### Post-hoc audit (existing hook)

1. Run `carrel config view hook <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:hook:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the hook is still executable and the shebang is valid.
5. Check that dependencies (tools, other hooks) are still present.
