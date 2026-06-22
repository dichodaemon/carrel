---
description: >
  Carrel configuration model, registry, deployment pipeline, and source layout.
  Read before creating, modifying, or deleting any OMP configuration.
globs:
  - '**/.omp/**'
---

# Carrel Configuration

Carrel is the configuration manager for the Scriptorium ecosystem. It assembles OMP and OS tool configuration from multiple sources, deploys to the paths OMP expects, and provides a queryable registry for humans and agents.

- **On disk:** `/workspace/carrel` (carrel binary and universal config)
- **On GitHub:** `dichodaemon/carrel`
- **Registry:** `/workspace/.carrel/registry` (Dolt-backed)

When a task requires modifying OMP configuration, use `carrel config` subcommands — not direct file edits in deployed `.omp/` directories.

## Source hierarchy

Configuration comes from sources registered in the carrel registry. Sources are ordered by scope:

| Scope | Applies to | Example |
|---|---|---|
| Universal | All consumers | `/workspace/carrel/omp/` |
| Target-specific | One consumer | `<target-repo>/.carrel/` or a sibling `<target>-config/` repo |
| User-personal | OS tool customization | User dotfiles repo |
| Host-local | Host-level overrides | `config/local/` |

Universal sources are inherited by all consumers. Deeper scopes override shallower scopes.

## Deployment (`carrel run`)

`carrel run` resolves the consumer from the current working directory, composes configuration from all applicable sources, deploys to the locations OMP discovers, then execs `omp`.

- `.omp/` in the target repo is the deployed output — **never edit it directly**.
- `AGENTS.md` at the repo root is deployed by carrel for non-opt-in repos and listed in `.git/info/exclude`.
- For opt-in repos (those with `.carrel/`), carrel deploys to `.omp/` only; `AGENTS.md` is user-managed.

**Preview before deploying:**
```bash
carrel plan          # What would the next deployment produce?
```

## Key rules

- **Never edit `.omp/` directly.** It is deployed by `carrel run`. Changes are overwritten.
- **Use `carrel config` commands to manage configuration.** All CRUD goes through `carrel config <subcommand> <type> <name>`.
- **Source location determines scope:**
  - Universal: `/workspace/carrel/omp/<type>/` — inherited by all consumers.
  - Target-specific: `<target>/.carrel/<type>/` or `<target>-config/omp/<type>/` — only that consumer.
- **Restart the OMP session** after modifying configuration (most config is loaded at init).
- **Flag gaps, don't decide them.** When porting or replacing a system, any feature present in the source system that is absent in the target is a gap. Surface it immediately. Do not defer, skip, or mark as "non-critical" without asking.
- **Three ways to edit source files:**
  - `carrel config edit <type> <name> --content="..."` — updates file + registry hash atomically.
  - `carrel config edit <type> <name> --file=<path>` — re-reads the file from disk and updates the registry hash. Use after editing a source file directly with complex multi-line changes.
  - Edit source file directly, then `carrel config scan <source-alias>` — picks up *new* entries not yet in the registry. **Does not resync hashes for existing entries.** Use `edit --file=` instead for those.

## Available configuration types

| Type | Description |
|---|---|
| `rule` | Rules that govern agent behavior (conventions, constraints, prohibitions) |
| `skill` | Agent skills for specialized workflows |
| `command` | Slash-command definitions |
| `extension` | OMP extensions |
| `agent` | Agent personality and model configurations |
| `tool` | External tool integrations |
| `hook` | Lifecycle hooks (pre/post session operations) |
| `prompt` | Reusable prompt templates |
| `instruction` | Instructional content injected into sessions |
| `context-file` | Project-level context files (AGENTS.md, CLAUDE.md) |
| `append-system` | System prompt appendix content (concatenated across sources) |
| `zsh` | Zsh configuration files |
| `nvim` | Neovim configuration files |
| `wezterm` | WezTerm configuration files |
| `p10k` | Powerlevel10k configuration |

List all types: `carrel config types`

## Configuration CRUD

All configuration management goes through `carrel config <subcommand>`.

### Add an entry

```bash
carrel config add rule no-push-master --source=carrel-omp --content='Never push to master'
carrel config add skill my-skill --source=carrel-omp --file=./SKILL.md
echo "content" | carrel config add hook pre-commit --source=carrel-omp
```

Input via `--content`, `--file`, or stdin. Requires `--source`.

### List entries of a type

```bash
carrel config list rule
carrel config list skill
```

### View an entry

```bash
carrel config view rule no-push-master
carrel config view rule no-push-master --meta-only
carrel config view skill validate --content-only
```

### Edit an entry

```bash
carrel config edit rule no-push-master --content='Updated content'
carrel config edit skill validate --file=./updated-validate.md
```

Alternatively, edit the source file directly then resync the hash:
```bash
# Re-read the on-disk file and update the registry hash:
carrel config edit skill validate --file=omp/skills/validate/SKILL.md
```

**Note:** `carrel config scan` only registers *new* entries. It does
not update hashes for entries already in the registry. After editing
an existing source file directly, use `edit --file=` to resync.


### Rename an entry

```bash
carrel config rename rule old-name new-name
carrel config rename skill old-validate new-validate
```

### Remove an entry

```bash
carrel config rm rule no-push-master
carrel config rm skill validate
```

Deletes the file from disk and removes the registry entry.

## Querying state

```bash
carrel sources           # All registered entries with on-disk status (EXISTS / MISSING)
carrel sources --type=rule --source=carrel-omp   # Filtered
carrel deployed          # Deployed files with verification status
carrel inspect carrel-omp:rule:carrel-configuration   # Inspect a source file
carrel inspect <consumer>:<absolute-path>   # Inspect deployed file
carrel trace carrel-omp:rule:carrel-configuration    # Trace composition provenance
carrel feeds <consumer>           # Which sources feed a consumer
carrel dependents carrel-omp   # Which consumers depend on a source
carrel status            # Registry state (consumers + sources)
carrel verify            # Check deployed state against registry claims
carrel plan              # Preview next deployment output
carrel discover          # Workspace repos and registration status
```

## Registering a new target repo

1. `carrel discover` — lists unregistered repos.
2. `carrel bootstrap` — initializes the registry if not done.
3. For opt-in repos: add `.carrel/` with configuration. For opt-out repos: create a sibling `<target>-config/` repo.
4. Run `carrel run` from the target repo to deploy and start an OMP session.

## OS tool configuration

Carrel also manages OS-level configuration (zsh, nvim, wezterm, p10k). These types use the same `carrel config` CRUD commands:

```bash
carrel config list zsh
carrel config add zsh my-custom --source=carrel-omp --file=./custom.zsh
```

Deployed via:
```bash
carrel os-setup          # Deploy OS config in container
carrel host-setup        # Deploy OS config on host
```

## Output slots

Slots map configuration entries to deployment paths. Manage with `carrel slot`:

```bash
carrel slot list
carrel slot add <name> --dest=<deploy-path>
carrel slot add-entry <source-alias>:<type>:<name> <slot-name>
carrel slot rm-entry <source-alias>:<type>:<name> <slot-name>
```

## Local overrides

Per-user or per-host overrides layered on top of universal config:

```bash
carrel local init          # Initialize local config source
carrel local scan          # Scan and link entries to slots
carrel local path          # Print local source directory
```
