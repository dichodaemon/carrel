---
name: create-skill
description: >
  Create a new carrel skill end-to-end: write SKILL.md, register in the
  source, create a deployment slot, wire the entry, verify, commit, and
  remind the user to deploy. Ensures no wiring step is skipped.
  Usage: /create-skill <name> [--source=<source-alias>]
---

# Create Skill

Create a new carrel skill with correct file structure, registry
registration, slot wiring, and verification. The skill is committed
and ready for deployment after this workflow completes.

## Trigger

User says "create skill", "add a skill", "new skill", or invokes
`/create-skill`.

## Arguments

- Required: skill name (kebab-case, e.g., `my-workflow`).
- Optional: `--source=<source-alias>` (default: `carrel-omp` for
  universal scope).

## Step 1: Gather Requirements

Determine from the user's request:

- **Name**: kebab-case slug (e.g., `create-plan`). This becomes the
  directory name and the `name` field in frontmatter.
- **Scope**: universal (`carrel-omp`) or target-specific. Default to
  universal unless the user specifies otherwise.
- **Description**: one-sentence summary for the frontmatter. For
  non-trivial skills, end with `Usage: /<name> [args]`.
- **Trigger phrases**: natural-language variants the user might say to
  invoke this skill (e.g., "create plan", "write a plan").
- **Complexity tier**: determines the section structure.
  - **Minimal**: body IS the instruction prose, no H2 sections. Use for
    simple behavioral directives (e.g., grill-me).
  - **Procedural**: flat `## Step N:` sections, no phases. Use for
    linear multi-step workflows (e.g., rebase-pr-chain).
  - **Complex**: `## Phase N:` with `### Step N:` subsections, plus
    `## Trigger`, `## Arguments`, and `## Rules` sections. Use for
    multi-phase workflows with validation and branching (e.g.,
    execute-plan).

If the user has not provided enough detail, ask. Do not guess at the
skill's behavior.

## Step 2: Write SKILL.md

Create the file at the source's skill directory:

- Universal: `/workspace/carrel/omp/skills/<name>/SKILL.md`
- Target-specific: `<target>/.carrel/skills/<name>/SKILL.md`

### Frontmatter

```yaml
---
name: <name>
description: >
  <description ending with Usage: /<name> [args]>
---
```

The `name` field must match the containing directory name exactly.

### Body

Structure depends on the complexity tier determined in Step 1.

**Complex skills** must include:

- `# <Title>` (H1, human-readable form of the name)
- One-paragraph purpose statement
- `## Trigger` with natural-language variants and slash-command
- `## Arguments` with required/optional parameters
- `## Phase N: <Name>` sections with `### Step N:` subsections
- `## Rules` section with flat bullet list of invariants

**Procedural skills** must include:

- `# <Title>` (H1)
- One-paragraph purpose statement
- `## Step N: <Name>` sections (flat, no phases)

**Minimal skills** have no required structure beyond the frontmatter.

## Step 3: Register

Register the new file in the config:

```bash
carrel config add --file=omp/skills/<name>/SKILL.md
```

Verify registration:

```bash
carrel config view skill <name> --meta-only
```

Expected output includes `Name: <name>`, `Type: skill`, and a `Hash`
value. If the skill does not appear, check that the file path matches
the source's skill directory and that the frontmatter is valid YAML.

## Step 4: Wire Slot

Sync all deployment slots (this picks up the newly registered entry):

```bash
carrel slot sync-all
```

**This step is mandatory.** A skill that is registered but not slotted
will not be deployed by `carrel run` and will not appear in the OMP
session. This is the most commonly forgotten step.

## Step 5: Verify

Run three checks. All must pass before committing.

### Check 1: Registry listing

```bash
carrel config list skill
```

The new skill must appear in the output with the correct source alias.

### Check 2: Deployment trace

```bash
carrel trace <source-alias>:skill:<name>
```

Must show the target consumer and deployment path. If it says "Not
deployed anywhere", the slot wiring (Step 4) failed.

### Check 3: Deployment plan

```bash
carrel plan | grep <name>
```

Must show `+ added` for the skill's deployment path. This confirms
the next `carrel run` will deploy it.

## Step 6: Commit and Push

```bash
cd /workspace/carrel
git add omp/skills/<name>/SKILL.md
git commit -m "feat(skills): add <name> skill

<one-line description of what the skill does>"
git push
```

After pushing, remind the user:

> Skill `<name>` is registered, slotted, and pushed. Run `carrel run`
> from a terminal to deploy it to your workspace. The skill will be
> available after the next OMP session restart.

## Rules

- **Name must be kebab-case** and match the directory name exactly.
  `create-plan` not `createPlan` or `create_plan`.
- **Never skip slot wiring.** Registration without a slot means the
  skill exists in the registry but is invisible to deployment. Always
  complete Step 4.
- **Agent sessions cannot run `carrel run`.** It is interactive and
  acquires a database lock. Always remind the user to run it from a
  terminal.
- **Never edit `.omp/` directly.** Skill files go in the source
  directory (`/workspace/carrel/omp/skills/` for universal). The
  `.omp/` directory is deployed output and is overwritten on every
  `carrel run`.
- **Verify before committing.** All three checks in Step 5 must pass.
  A committed-but-unwired skill creates confusion — it exists in git
  but doesn't deploy.
- **One skill per directory.** Each skill gets its own directory
  containing `SKILL.md` and optional supplementary files.
