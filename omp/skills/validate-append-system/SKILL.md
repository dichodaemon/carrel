---
name: validate-append-system
description: >
  Validate an append-system entry's content, naming, and cross-references.
  Use before `carrel config add append-system` or to audit existing append-system entries.
  Usage: /validate-append-system <name> [--source=<source-alias>]
---

# Validate Append-System

Validate an append-system entry for structural correctness and
completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered
append-system content.

## What to Validate

### Content structure

- The file is valid Markdown with optional YAML frontmatter.
- The body contains system-level directives that are concatenated
  across sources into the OMP system prompt appendix.
- Content is written as system-authored authoritative text — directives
  use RFC 2119 keywords where applicable (`MUST`, `SHOULD`, `NEVER`).
- The file uses `<system-directive>` or similar XML tags if it needs to
  inject authoritative markers into the chat stream.
- The content is self-contained: it does not assume context from other
  append-system files (they are concatenated independently).

### Naming conventions

- The entry name is kebab-case.
- The filename uses lowercase with hyphens and a `.md` extension.
- The name indicates the policy domain (e.g., `beads-policy`,
  `carrel-policy`).

### Cross-references

- If the file references rules, skills, or other configuration, those
  entries exist.
- The content does not contradict other append-system entries deployed
  to the same slot.
- Referenced file paths, commands, or conventions are valid in the
  target environment.

## How to Validate

### Pre-registration (new append-system)

1. Verify the file is in a recognized location for append-system
   content (e.g., `omp/<name>.md`).
2. Check that the filename is kebab-case and uses `.md` extension.
3. Read the body: confirm it contains substantive system-level prose,
   not just boilerplate.
4. Check that XML-style directives (if used) are well-formed.
5. Verify cross-references to rules, skills, and other entries resolve.

### Post-hoc audit (existing append-system)

1. Run `carrel config view append-system <name> --meta-only` and
   confirm registration.
2. Run `carrel trace <source>:append-system:<name>` to verify
   deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the deployed content is consistently concatenated:
   `carrel inspect <consumer>:<deploy-path>` and confirm the entry's
   content is present.
5. Check for conflicts with other append-system entries in the same
   slot (no contradictory directives).
