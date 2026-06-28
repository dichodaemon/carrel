---
description: >
  Format, structure, naming, and best practices for slot-defaults entries.
  Slot-defaults define default entry-to-slot wiring for consumers that haven't
  declared their own `.carrel/slots.yml`. They ensure universal entries are
  deployed correctly without requiring per-consumer slot configuration.
  Use `carrel config add slot-defaults <name> --file=omp/slot-defaults.yml` to register.
  Validate with `validate-slot-defaults` skill before registering.
globs:
  - 'omp/slot-defaults.yml'
---

# Slot-Defaults Entry Type

Slot-defaults provide fallback slot wiring for consumers that lack their own `.carrel/slots.yml`. They ensure universal configuration entries are deployed to the correct paths without requiring every consumer to explicitly declare slots.

Slot-defaults are the safety net of the slot system: they guarantee that newly registered entries will deploy somewhere, even to consumers that haven't been customized.

## Format

- **File extension:** `.yml` (YAML)
- **Location:** `omp/slot-defaults.yml` — one file at the source root
- **Frontmatter:** None. Slot-defaults are pure YAML declarations.

### File Structure

```yaml
# Default slot contributions for the <source> source.
# These declarations ensure that consumers which haven't declared their own
# .carrel/slots.yml still get the correct entry-to-slot wiring.
#
# A consumer with its own .carrel/slots.yml takes priority — this file is
# skipped entirely for those consumers.

contributions:
  - slot: <slot-name>
    entries:
      - <type>:<entry-name>
      - <type>:<entry-name>
```

| YAML Path | Type | Description |
|---|---|---|
| `contributions` | list | Array of slot contribution blocks |
| `contributions[].slot` | string | Slot name — the deployment destination identifier |
| `contributions[].entries` | list | Entries wired to this slot, in format `<type>:<name>` |

### Entry Reference Format

Entry references use the format `<type>:<name>`:
- `rule:carrel-configuration` — the `carrel-configuration` rule entry
- `skill:create-plan` — the `create-plan` skill entry
- `append-system:APPEND_SYSTEM.md` — an append-system entry
- `zsh:zshrc.zsh` — a zsh config entry

## Naming Conventions

- **File name:** `slot-defaults.yml` — fixed name, one per source
- **Slot names:** kebab-case, descriptive of the deployment path or purpose:
  - `carrel-configuration` — for rule entries
  - `APPEND_SYSTEM.md` — for append-system entries
  - `grill-me` — for skill entries
- **Avoid generic slot names:** `default`, `misc`, `other`

## Content Best Practices

- **One contribution block per slot.** Group related entries under the same slot. A slot typically maps to a deployment path or a logical category.
- **Alphabetical order within entries.** Keep `entries` lists alphabetically sorted for readability and to minimize merge conflicts.
- **Comment the purpose.** Each file SHOULD start with a comment block explaining that this file is skipped when the consumer has its own `.carrel/slots.yml`.
- **Don't duplicate consumer slots.** Slot-defaults exist ONLY because some consumers lack `.carrel/slots.yml`. If a slot should be available to all consumers regardless, it belongs in slot-defaults. If a slot is consumer-specific, it belongs in the consumer's `.carrel/slots.yml`.
- **Every registered entry SHOULD have a slot-default.** A registered entry without a slot will not deploy anywhere. After creating a new entry, add it to the appropriate slot-defaults contribution block.
- **Keep in sync with registration.** Whenever you run `carrel config add`, follow up with `carrel slot sync-all` to ensure the new entry is wired. If `slot sync-all` can't find a slot, add one to `slot-defaults.yml`.
- **Slot name matches expected deployment.** Slot names are symbolic, but they SHOULD suggest the deployment path. The mapping from slot name to path is defined in carrel's slot registry.
- **Validate with `carrel slot sync-all --dry-run`.** Before committing slot-defaults changes, dry-run to see which entries will be newly wired and which slots will be affected.

## Deployment Behavior

1. **Registration:** `carrel config add slot-defaults <name> --file=omp/slot-defaults.yml`
2. **Composition:** During `carrel run` or `carrel slot sync-all`, carrel checks whether the consumer has its own `.carrel/slots.yml`:
   - **Consumer has `.carrel/slots.yml`:** `slot-defaults.yml` is **completely ignored** for that consumer. The consumer's own slots take full control.
   - **Consumer lacks `.carrel/slots.yml`:** `slot-defaults.yml` from each active source is used to wire entries to slots.
3. **Slot resolution:** Entries listed in `slot-defaults.yml` contributions are assigned to their declared slots. Entries not listed in any contribution block are left unslotted (and will not deploy).
4. **No deployment path.** Slot-defaults themselves don't deploy to a user-visible path. They are metadata consumed by carrel's composition engine.

### Priority Model

```
Consumer has .carrel/slots.yml?
├── YES → Use consumer's slots.yml; ignore all slot-defaults.yml
└── NO  → Use slot-defaults.yml from all active sources
```

This is an all-or-nothing override. You cannot partially override slot-defaults — if the consumer defines any slots, it must define all of them.

## Validation

Use the `validate-slot-defaults` skill to check:
- File is valid YAML
- `contributions` is a list of objects with `slot` (string) and `entries` (list of strings)
- Entry references use the format `<type>:<name>` with recognized types
- Referenced `<type>:<name>` entries exist in the registry
- No duplicate slot names across contribution blocks
- No duplicate entry references within the same slot
- File is at the correct path: `omp/slot-defaults.yml`

## Example

```yaml
# Default slot contributions for the carrel-omp universal source.
# These declarations ensure that consumers which haven't declared their own
# .carrel/slots.yml still get the correct entry-to-slot wiring.
#
# A consumer with its own .carrel/slots.yml takes priority — this file is
# skipped entirely for those consumers.

contributions:
  - slot: APPEND_SYSTEM.md
    entries:
      - append-system:APPEND_SYSTEM.md
      - append-system:beads-policy.md
      - append-system:carrel-policy.md
      - append-system:folio-policy.md

  - slot: carrel-configuration
    entries:
      - rule:carrel-configuration

  - slot: grill-me
    entries:
      - skill:grill-me

  - slot: no-push-oh-my-pi
    entries:
      - rule:no-push-oh-my-pi

  - slot: no-todo-write
    entries:
      - rule:no-todo-write

  - slot: rebase-pr-chain
    entries:
      - skill:rebase-pr-chain
```
