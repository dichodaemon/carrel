# Glossary

## Terms

### Core Architecture

**Carrel**:
The configuration deployment engine — a Go binary that resolves sources, composes entries, and provisions configuration into target workspaces.
_Avoid_: config-manager, provisioner

**Consumer**:
A target workspace (repository) that receives deployed configuration from carrel.
_Avoid_: target, target-repo, target workspace

**Entry**:
A single configuration artifact — the registry representation of a source file — tracked with content hash, metadata, and origin.
_Avoid_: config entry, configuration entry

**Registry**:
The Dolt-backed configuration store tracking consumers, sources, entries, slots, and deployments.

**Slot**:
A named grouping of entries mapped to a deployment path for a specific consumer.
_Avoid_: output target, destination

**Source**:
A directory or repository providing configuration entries, registered in the carrel registry with a scope.
_Avoid_: config source, source directory

**Source alias**:
A short unique name identifying a source in the registry and CLI commands.
_Avoid_: --source flag, source name

**Source file**:
An individual file within a source that backs exactly one entry.

### Composition and Deployment

**Claim**:
A recorded path-hash pair from a successful deployment, used on the next deployment to detect drift and clean up stale files.
_Avoid_: DeploymentEntry, previousClaim

**ComposeMode**:
The operator governing how entries from multiple sources combine — Override (deepest scope wins) or Concatenate (append in source order).
_Avoid_: Primitive

**Composition**:
The process of resolving a consumer's slots and entries into a Deployment Plan.

**Conflict Policy**:
The rule governing how the deployer handles destination files not owned by carrel — error, backup, or skip.

**Deployment Plan**:
The set of files with resolved content produced by composing a consumer's slots and entries.
_Avoid_: Output Plan, plan

### Source Organization

**Compound ID**:
A `<source-alias>:<type>:<name>` string identifying an entry across all sources.

**Config type**:
A classification of configuration entry types — rule, skill, command, extension, agent, tool, hook, prompt, instruction, context-file, append-system, and OS tool types (zsh, nvim, wezterm, p10k).
_Avoid_: Capability, CapabilityType, type, configuration type

**Convention**:
The on-disk layout rules defining how entries of a given config type are named and organized within a source directory.

**Scope**:
The set of consumers a source applies to — universal, target-specific, user-personal, or host-local.
_Avoid_: SourceScope

## Relationships

- A **Carrel** deploys configuration into **Consumers**
- A **Consumer** contains one or more **Slots**
- A **Slot** maps zero or more **Entries** to a deployment path
- An **Entry** is backed by exactly one **Source file**
- A **Source file** belongs to exactly one **Source**
- A **Source** is identified by exactly one **Source alias**
- A **Source** provides zero or more **Entries**
- A **Registry** tracks **Consumers**, **Sources**, **Entries**, **Slots**, and **Claims**
- **Composition** produces a **Deployment Plan** from a **Consumer**'s **Slots** and **Entries**
- A **Deployment Plan** produces zero or more **Claims** on successful deployment
- Each **Entry** has a **ComposeMode** and belongs to a **Source** with a **Scope**
- Each **Entry** is classified by exactly one **Config type**
- Each **Config type** has exactly one **Convention**
- A **Compound ID** references an **Entry** by its **Source alias**, **Config type**, and name

## Example Dialogue

> **Dev:** "When I add a new skill file to `omp/skills/`, is that a new **Source**, a new **Source file**, or a new **Entry**?"
> **Domain expert:** "It's a new **Source file** — the skill's `SKILL.md` inside the existing **Source**. After scanning, it becomes an **Entry** in the **Registry**."
>
> **Dev:** "And where does the **Slot** come in?"
> **Domain expert:** "The **Entry** gets mapped into one or more **Slots**. Each **Slot** belongs to a **Consumer** and has a deployment path. When carrel composes, it collects all the **Entries** in each **Slot** and writes the resolved content to that path."
>
> **Dev:** "So the deployment path is a property of the **Slot**, not the **Entry**?"
> **Domain expert:** "Right. The **Entry** doesn't know where it ends up. The **Slot** decides that — it's the bridge between what gets deployed and where it lands."

## Flagged Ambiguities

- README claimed four **Composition** primitives — Override, Concatenation, Reference, and Inheritance — but only Override and Concatenation are implemented — resolved 2026-05-08: Reference and Inheritance do not exist in code or the registry schema. Docs need updating.
- README and carrel-configuration rule referenced a "final flag" preventing override by deeper scopes — resolved 2026-05-08: the `final` column was dropped in registry schema migration v6. The flag is not in the **Entry** struct and is not read by the composer. Docs need updating.
