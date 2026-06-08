---
name: update-skill
description: >
  Update an existing carrel skill's content and resync the registry.
  Ensures the registry hash matches the edited source file so
  deployment stays consistent. Usage: /update-skill <name>
---

# Update Skill

Update an existing carrel skill's SKILL.md and resync the registry
so the next `carrel run` deploys the updated content.

## Trigger

User says "update skill", "edit skill", "modify skill", or invokes
`/update-skill`.

## Arguments

- Required: skill name (kebab-case, e.g., `create-plan`).

## Step 1: Identify the Skill

Verify the skill exists and locate its source file:

```bash
carrel config view skill <name> --meta-only
```

Expected output includes `Name`, `Type: skill`, `Path`, and `Hash`.
If the skill does not exist, suggest `/create-skill` instead.

Read the current content of the source file to understand what is
being changed.

## Step 2: Edit SKILL.md

Modify the source file directly. The source path follows the pattern:

- Universal: `/workspace/carrel/omp/skills/<name>/SKILL.md`
- Target-specific: `<target>/.carrel/skills/<name>/SKILL.md`

The `Path` field from Step 1 gives the relative path within the
source. Combine with the source's root directory to get the absolute
path.

If renaming the skill, use `carrel config rename` instead of manual
file moves:

```bash
carrel config rename skill <old-name> <new-name>
```

This updates the registry entry, renames the file on disk, and
updates slot references atomically.

## Step 3: Resync Registry

After editing the source file, update the registry hash:

```bash
carrel config edit skill <name> --file=<absolute-path-to-SKILL.md>
```

**This step is mandatory.** `carrel config scan` skips
already-registered entries — it only picks up new files. Direct edits
to existing source files require `carrel config edit` to resync the
registry hash. Without this, `carrel verify` will report drift between
the source and registry.

## Step 4: Verify

Run two checks. Both must pass before committing.

### Check 1: Registry entry

```bash
carrel config view skill <name> --meta-only
```

Confirm the `Hash` value has changed from the value observed in Step 1.
If the hash is unchanged, the `carrel config edit` in Step 3 did not
take effect — re-run it with the correct file path.

### Check 2: Deployment plan

```bash
carrel plan | grep <name>
```

Should show `~ modified` (content changed) or remain as `DEPLOYED`
(already in sync). Either is acceptable. If the skill does not appear
at all, the slot wiring is broken — run `/create-skill` Step 4 to
repair it.

## Step 5: Commit and Push

```bash
cd /workspace/carrel
git add omp/skills/<name>/SKILL.md
git commit -m "fix(skills): update <name> skill

<one-line description of what changed>"
git push
```

After pushing, remind the user:

> Skill `<name>` is updated and pushed. Run `carrel run` from a
> terminal to deploy the changes. The updated skill will be available
> after the next OMP session restart.

## Rules

- **Never edit `.omp/` directly.** Edit the source file. The `.omp/`
  directory is deployed output and is overwritten on every `carrel run`.
- **Always resync the registry after editing.** `carrel config scan`
  does not update existing entries. Use `carrel config edit` to resync
  the hash. Skipping this leaves the registry stale.
- **Use `carrel config rename` for renames.** Do not manually rename
  the directory and re-register. The rename command handles registry,
  file, and slot references atomically.
- **Verify before committing.** Both checks in Step 4 must pass. A
  committed edit with a stale registry hash creates drift that
  `carrel verify` will flag.
