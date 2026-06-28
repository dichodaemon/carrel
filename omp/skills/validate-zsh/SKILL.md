---
name: validate-zsh
description: >
  Validate a zsh entry's content, naming, and cross-references.
  Use before `carrel config add zsh` or to audit existing zsh config.
  Usage: /validate-zsh <name> [--source=<source-alias>]
---

# Validate Zsh

Validate a zsh configuration entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered zsh
config files.

## What to Validate

### Content structure

- The zsh file is valid shell script with correct syntax.
- The file has a `.zsh` extension (e.g., `zshrc.zsh`, `aliases.zsh`).
- If the file is a sourced fragment (not a standalone script), it does
  not contain a shebang line.
- Variable assignments use proper quoting to prevent word splitting.
- Functions use `kebab-case` or `snake_case` naming consistently.
- The file does not override critical environment variables without
  explicit intent.

### Naming conventions

- The entry name is kebab-case.
- The filename uses lowercase with hyphens and a `.zsh` extension.
- Common names: `zshrc.zsh` (interactive shell), `zprofile.zsh`
  (login shell), `aliases.zsh`, `functions.zsh`.

### Cross-references

- If the file sources other zsh files, those files exist.
- If the file references tools or binaries, those are available in the
  target environment.
- The file does not conflict with other zsh entries deploying to the
  same target path.

## How to Validate

### Pre-registration (new zsh config)

1. Verify the file is at `omp/config/zsh/<name>.zsh` or the
   target-specific equivalent.
2. Check that the filename uses kebab-case and the `.zsh` extension.
3. Run a syntax check: `zsh -n <file>` to validate shell syntax.
4. Check for common issues: unquoted variables, missing shebang (for
   fragments), undefined function calls.
5. Verify that sourced files exist if referenced with explicit paths.

### Post-hoc audit (existing zsh config)

1. Run `carrel config view zsh <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:zsh:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Run `zsh -n` on the deployed file to confirm syntax is still valid.
5. Verify the deployment target path via `carrel inspect` and confirm
   the file is present.
