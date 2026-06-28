---
description: >
  Format, structure, naming, and best practices for zsh configuration entries.
  Zsh config is deployed to the shell environment via `carrel os-setup` (container)
  or `carrel host-setup` (host). Files are sourced at shell startup.
  Use `carrel config add zsh <name> --file=omp/config/zsh/<name>.zsh` to register.
  Validate with `validate-zsh` skill before registering.
globs:
  - 'omp/config/zsh/*.zsh'
  - 'omp/config/zsh/*.sh'
---

# Zsh Entry Type

Zsh entries are shell configuration files deployed into the user's Zsh environment. They are sourced at shell startup and can define environment variables, aliases, functions, completions, key bindings, and shell options.

Unlike core config types deployed by `carrel run`, Zsh config is deployed by `carrel os-setup` (container environments) or `carrel host-setup` (bare-metal hosts).

## Format

- **File extension:** `.zsh` (preferred) or `.sh`
- **Location:** `omp/config/zsh/<name>.zsh`
- **Frontmatter:** None. Zsh files are plain shell scripts sourced directly by Zsh.

## Naming Conventions

- **Snake_case or kebab-case:** `zshrc.zsh`, `zprofile.zsh`, `p10k.zsh`, `aliases.zsh`, `completions.zsh`
- **Standard Zsh startup files** have reserved names and meanings:
  - `zshrc.zsh` — sourced for interactive shells (aliases, key bindings, prompt)
  - `zprofile.zsh` — sourced for login shells (environment variables, PATH)
  - `zshenv.zsh` — sourced for all shells (rarely used; prefer `zshrc` or `zprofile`)
- **Descriptive of content:** `python-aliases.zsh`, `docker-completions.zsh`, `git-prompt.zsh` for modular files
- **Avoid generic names:** `config.zsh`, `custom.zsh`

## Content Best Practices

- **Idempotent.** Shell config may be sourced multiple times (subshells, `exec zsh`). Guard against double-execution:
  ```zsh
  # Guard variable pattern
  if [[ -z "${_MY_MODULE_SOURCED:-}" ]]; then
    export _MY_MODULE_SOURCED=1
    # ... module content ...
  fi
  ```
- **Use `[[` not `[`.** Zsh-native conditionals are safer and more featureful than POSIX `[`.
- **Quote all expansions.** `"$VAR"` not `$VAR`. Unquoted expansions in Zsh don't word-split by default, but quoting is still best practice for portability and clarity.
- **`#` comments for documentation.** Each file SHOULD start with a comment block explaining its purpose and any dependencies.
- **Source ordering matters.** Files are sourced in alphabetical order by name. Use numeric prefixes (`10-`, `20-`) if ordering is critical.
- **Avoid side effects on non-interactive shells.** Guard interactive-only configuration:
  ```zsh
  if [[ -o interactive ]]; then
    # interactive-only setup
  fi
  ```
- **Use `$ZDOTDIR` for paths.** Don't hardcode `~/.zshrc`; use `$ZDOTDIR` which carrel configures.
- **Separate local overrides.** User-specific customizations belong in `~/.zshrc.local`, not in carrel-managed files. Carrel-managed files should reference local files:
  ```zsh
  [[ -f ~/.zshrc.local ]] && source ~/.zshrc.local
  ```
- **Completions use standard paths.** Place completion files in standard `$FPATH` locations or use `compinit` with explicit directories.

## Deployment Behavior

1. **Registration:** `carrel config add zsh <name> --file=omp/config/zsh/<name>.zsh`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:**
   - Container: `carrel os-setup` deploys Zsh config to the container's `$ZDOTDIR`
   - Host: `carrel host-setup` deploys to the host's `$ZDOTDIR`
4. **Loading:** Zsh sources the deployed files at shell startup according to standard Zsh startup sequence: `.zshenv` → `.zprofile` → `.zshrc` → `.zlogin`

Zsh config is NOT deployed by `carrel run`. It requires a separate `carrel os-setup` or `carrel host-setup` invocation.

## Validation

Use the `validate-zsh` skill to check:
- File has `.zsh` or `.sh` extension
- Syntax is valid Zsh (`zsh -n <file>` passes)
- No dangerous patterns: `rm -rf /`, unbounded globs, `curl | sh`
- Idempotency guards present where appropriate
- No hardcoded home directory paths (use `$HOME` or `$ZDOTDIR`)
- File is at the correct path under `omp/config/zsh/`

## Example

```zsh
# Git-aware prompt components for interactive shells.
# Depends on: git being installed.
# Sourced by: zshrc.zsh

if [[ -z "${_GIT_PROMPT_SOURCED:-}" ]]; then
  export _GIT_PROMPT_SOURCED=1

  if [[ -o interactive ]]; then
    # Show current git branch in right prompt
    autoload -Uz vcs_info
    precmd() { vcs_info }
    zstyle ':vcs_info:git:*' formats ' %b'
    setopt prompt_subst
    RPROMPT='${vcs_info_msg_0_}'
  fi
fi
```
