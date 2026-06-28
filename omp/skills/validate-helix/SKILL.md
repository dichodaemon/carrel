---
name: validate-helix
description: >
  Validate a helix entry's content, naming, and cross-references.
  Use before `carrel config add helix` or to audit existing helix config.
  Usage: /validate-helix <name> [--source=<source-alias>]
---

# Validate Helix

Validate a Helix editor configuration entry for structural correctness
and completeness. Works for pre-registration checks (before
`carrel config add`) and post-hoc audits of already-registered Helix
config files.

## What to Validate

### Content structure

- The file is valid TOML compatible with Helix's configuration format.
- The file uses Helix's configuration sections: `[editor]`, `[keys]`,
  and/or language-specific `[[language]]` tables.
- Editor settings use valid Helix option names (e.g., `line-number`,
  `mouse`, `auto-format`).
- Keybindings in `[keys.normal]`, `[keys.insert]`, and
  `[keys.select]` use valid Helix command names.
- Language configurations reference valid `name` and `language-server`
  settings.
- Theme references (`theme` key) match installed themes.

### Naming conventions

- The entry name is kebab-case.
- The filename uses lowercase with hyphens and a `.toml` extension.
- Common name: `config.toml`.

### Cross-references

- Referenced language servers are installed or declared as
  dependencies.
- Keybinding commands reference valid Helix built-in commands.
- If the config extends another Helix config, the base config exists.

## How to Validate

### Pre-registration (new helix config)

1. Verify the file is at `omp/config/helix/<name>.toml` or the
   target-specific equivalent.
2. Check that the filename uses kebab-case and the `.toml` extension.
3. Parse the TOML: confirm it is syntactically valid.
4. Verify top-level sections are valid Helix configuration sections.
5. Check keybinding commands against the Helix command reference.
6. Validate language server names are recognized LSP implementations.

### Post-hoc audit (existing helix config)

1. Run `carrel config view helix <name> --meta-only` and confirm
   registration.
2. Run `carrel trace <source>:helix:<name>` to verify deployment.
3. Compare on-disk hash with the registry hash.
4. Verify the TOML syntax is still valid.
5. Check that referenced themes, language servers, and keybindings are
   still valid for the installed Helix version.
