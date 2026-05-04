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

When a task requires modifying OMP configuration, use `carrel <type> add|rm|edit` commands — not direct file edits in deployed `.omp/` directories.

## Source hierarchy

Configuration comes from sources registered in the carrel registry. Sources are ordered by scope:

| Scope | Applies to | Example |
|---|---|---|
| Universal | All consumers | `/workspace/carrel/omp/` |
| Target-specific | One consumer | `<target-repo>/.carrel/` or a sibling `<target>-config/` repo |
| User-personal | OS tool customization | User dotfiles repo |
| Host-local | Host-level overrides | `config/local/` |

Universal sources are inherited by all consumers. Deeper scopes override shallower scopes (unless an entry has the `final` flag).

## Deployment (carrel run)

`carrel run` resolves the consumer from the current working directory, composes configuration from all applicable sources, deploys to the locations OMP discovers, then execs `omp`.

- `.omp/` in the target repo is the deployed output — **never edit it directly**.
- `AGENTS.md` at the repo root is deployed by carrel for non-opt-in repos and listed in `.git/info/exclude`.
- For opt-in repos (those with `.carrel/`), carrel deploys to `.omp/` only; `AGENTS.md` is user-managed.

## Key rules

- **Never edit `.omp/` directly.** It is deployed by `carrel run`. Changes are overwritten.
- **Use carrel commands to manage configuration.** `carrel <type> add|rm|edit|list|view|update|rename`.
- **Source location determines scope:**
  - Universal: `/workspace/carrel/omp/<type>/` — inherited by all consumers.
  - Target-specific: `<target>/.carrel/<type>/` or `<target>-config/omp/<type>/` — only that consumer.
- **Restart the OMP session** after modifying configuration (most config is loaded at init).

## Common operations

### Adding a rule, skill, or command

```bash
carrel rule add no-push-master --source=carrel-omp --content='...'
carrel skill add grill-me --source=carrel-omp
carrel rule list
carrel rule view no-push-master
```

### Registering a new target repo

1. `carrel discover` — lists unregistered repos.
2. `carrel bootstrap` — initializes the registry if not done.
3. For opt-in repos: add `.carrel/` with configuration. For opt-out repos: create a sibling `<target>-config/` repo.
4. Run `carrel run` from the target repo to deploy and start an OMP session.

### Querying state

```bash
carrel status       # registered consumers and sources
carrel verify       # deployed state vs. registry claims
carrel discover     # workspace repos and registration status
```
