# Carrel

Carrel configures any repository for AI-assisted development without modifying
tracked files. It is the configuration manager for the Scriptorium ecosystem:
a single Go binary that assembles OMP and OS tool configuration from multiple
sources via a Dolt-backed registry, deploys to the paths OMP discovers, and
provides a queryable interface for humans and agents.

---

## Contents

| File | Description |
|---|---|
| `cmd/carrel/` | CLI entry point (cobra command tree). |
| `internal/registry/` | Dolt-backed configuration store with in-memory test backend. |
| `internal/composer/` | Pure composition pipeline: sources → output plan. |
| `internal/deployer/` | Writes output plan to disk, handles collisions, manages claims. |
| `internal/scanner/` | Workspace discovery and carula migration. |
| `internal/authoring/` | CRUD operations on configuration entries. |
| `internal/tui/` | Bubbletea dashboard for interactive status and verification. |
| `omp/` | Carrel's own universal OMP configuration source. |
| `Dockerfile` | Container image with Go 1.26, CGO, and all tooling. |
| `integration_test.go` | End-to-end CLI tests against real workspace. |

---

## API

Carrel is consumed via CLI subcommands, not as a Go library. The `internal/`
packages are compiler-enforced private — only `cmd/carrel` imports them.

```bash
carrel bootstrap          # Initialize registry
carrel run [--dry-run]    # Compose, deploy, exec OMP
carrel discover           # Scan workspace for repos
carrel migrate            # Import carula configuration
carrel status             # Show registry state
carrel verify             # Check deployed state
carrel os-setup           # Deploy OS tool config in container
carrel host-setup         # Deploy OS tool config on host
carrel dashboard          # Launch TUI
carrel <type> add|rm|list|view|edit|update|rename   # Per-type CRUD
```

---

## Usage

### Build

```bash
cd /workspace/carrel
CGO_ENABLED=1 go build ./cmd/carrel
```

### Bootstrap

```bash
./carrel bootstrap
```

Creates the registry at `/workspace/.carrel/`, registers carrel and folio
as consumers, registers carrel's `omp/` as a universal source. Idempotent.

### Deploy a session

```bash
cd /workspace/<target-repo>
carrel run
```

Resolves the consumer from cwd, composes configuration from universal and
target-specific sources, deploys to `.omp/` and `AGENTS.md`, then execs `omp`.

---

## Dependencies

### Internal

| Package | Path | Purpose |
|---|---|---|
| (none) | | Carrel has no internal workspace dependencies. |

### External

| Package | Version |
|---|---|
| dolthub/driver | v1.86 |
| spf13/cobra | v1.10 |
| charm.land/bubbletea/v2 | v2.0 |
| cespare/xxhash/v2 | v2.3 |
| google/uuid | v1.6 |
| charm.land/lipgloss/v2 | v2.0 |

---

## Design

Carrel follows a wrapper pattern: it assembles configuration per-session,
writes to OMP's discovery paths, then hands off via Unix exec. No persistent
assembled state exists. The registry tracks metadata (consumers, sources,
entries, deployments); the filesystem owns content. Neither is master —
drift is detected and surfaced, never silently resolved.

Composition is a pure function. Four primitives govern how entries combine:
override (deepest scope wins, final flag prevents override), concatenation
(append in source order), reference, and inheritance. Sources are layered:
universal → target-specific → user-personal → host-local.

Deployment follows a claim model: every successful deployment records paths
and content hashes. On the next deployment, stale files are verified by hash
before removal. Foreign files at destination paths trigger a configurable
conflict policy (error, backup, or skip).

---

## Testing

Tests cover:

- **Registry** — Consumer/source/entry/deployment CRUD on Dolt and in-memory backends.
- **Composer** — Override (deepest scope, final flag), concatenation ordering, determinism.
- **Deployer** — Collision policies (error/backup/skip), stale cleanup, dry-run, claim recording.
- **Scanner** — Workspace discovery, carula symlink migration.
- **Authoring** — Add, remove, edit, update metadata, rename for all capability types.
- **Integration** — Bootstrap idempotency, consumer registration, discover, run flags, CRUD, status, verify.

Run from the dev container:

```bash
cd /workspace/carrel
go test ./... -count=1
```

Or with CGO:

```bash
CGO_ENABLED=1 go test ./... -count=1
```

---

## Reference Documents

| Document | Purpose |
|---|---|
| [`carrel_arch-design.md`](https://github.com/dichodaemon/carula/blob/master/docs/workflow/arch-designs/carrel_arch-design.md) | Settled architecture. |
| [`carrel-build_brief.md`](https://github.com/dichodaemon/carula/blob/master/docs/workflow/briefs/2026-05-04_carrel-build_brief.md) | Build plan framing 10 epics. |
| [`scriptorium-v1-configuration_design-study.md`](https://github.com/dichodaemon/carula/blob/master/docs/workflow/design-studies/2026-05-02_scriptorium-v1-configuration_design-study.md) | Configuration architecture decisions. |
| [`carrel-registry-storage_design-study.md`](https://github.com/dichodaemon/carula/blob/master/docs/workflow/design-studies/2026-05-03_carrel-registry-storage_design-study.md) | Registry schema and drift response. |
| [`carrel-composition-and-collision_design-study.md`](https://github.com/dichodaemon/carula/blob/master/docs/workflow/design-studies/2026-05-03_carrel-composition-and-collision_design-study.md) | Composition primitives and collision policy. |
