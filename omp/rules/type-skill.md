---
description: >
  Format, structure, naming, and best practices for skill entries.
  Skills are specialized agent workflows injected at session startup.
  Use `carrel config add skill <name> --file=omp/skills/<name>/SKILL.md` to register.
  Validate with `validate-skill` skill before registering.
globs:
  - 'omp/skills/*/SKILL.md'
---

# Skill Entry Type

Skills are specialized knowledge packages that teach agents how to perform specific workflows. They are loaded at OMP session startup and registered in the agent's skill inventory. When a user request matches a skill's trigger phrases or slash-command, the agent reads the skill and follows its instructions.

## Format

- **File extension:** `.md` (Markdown with YAML frontmatter)
- **Location:** `omp/skills/<name>/SKILL.md` — one directory per skill, containing exactly one `SKILL.md` plus optional supplementary files
- **Frontmatter fields:**
  - `name` (required) — kebab-case skill identifier, must match the directory name exactly
  - `description` (required) — YAML folded scalar summarizing the skill's purpose. For non-trivial skills, end with `Usage: /<name> [args]`

## Naming Conventions

- **kebab-case:** `create-skill`, `audit-plan`, `execute-plan`, `rebase-pr-chain`
- **Verb-first:** name describes the action the skill performs
- **Must match directory name exactly:** `omp/skills/create-skill/SKILL.md` → `name: create-skill`
- **Common prefixes:**
  - `create-` for generation workflows: `create-plan`, `create-skill`
  - `audit-` for validation/review: `audit-plan`, `audit-document`
  - `execute-` for execution workflows: `execute-plan`
  - `generate-` for output generation: `generate-glossary`
  - `find-` for discovery: `find-session`

## Content Best Practices

### Complexity Tiers

Skills fall into three tiers; choose based on workflow complexity:

- **Minimal:** body IS the instruction prose, no H2 sections. Use for simple behavioral directives (e.g., `grill-me` — "interview me relentlessly").
- **Procedural:** flat `## Step N:` sections, no phases. Use for linear multi-step workflows (e.g., `rebase-pr-chain`).
- **Complex:** `## Phase N:` with `### Step N:` subsections, plus `## Trigger`, `## Arguments`, and `## Rules` sections. Use for multi-phase workflows with validation and branching (e.g., `execute-plan`, `create-skill`).

### Required Sections by Tier

**Complex skills** MUST include:
- `# <Title>` — human-readable title (H1)
- One-paragraph purpose statement
- `## Trigger` — natural-language variants and slash-command
- `## Arguments` — required/optional parameters
- `## Phase N: <Name>` sections with `### Step N:` subsections
- `## Rules` section with flat bullet list of invariants

**Procedural skills** MUST include:
- `# <Title>` (H1)
- One-paragraph purpose statement
- `## Step N: <Name>` sections (flat, no phases)

**Minimal skills** have no required structure beyond the frontmatter.

### General Guidelines

- **Self-contained.** A skill is the agent's only reference when activated. Include all necessary context, commands, and decision points.
- **Actionable.** Every step must be a concrete instruction the agent can execute — not background reading.
- **Verify, don't assume.** Include explicit verification steps (e.g., "run `carrel config view skill <name> --meta-only` and confirm `Hash` is non-empty").
- **Reference related skills** for delegation: "Use the `create-plan` skill to scaffold the implementation plan."

## Deployment Behavior

1. **Source registration:** `carrel config add skill <name> --file=omp/skills/<name>/SKILL.md`
2. **Slot wiring:** `carrel slot sync-all` links the skill to the skill inventory slot
3. **Deployment:** `carrel run` deploys to `.omp/skills/<name>/SKILL.md`
4. **Loading:** OMP discovers skills at session startup and injects them into the agent's system prompt under `<skills>`

A skill that is registered but not slotted will NOT deploy and will NOT appear in the agent's session. Slot wiring is mandatory.

## Validation

Use the `validate-skill` skill (or `carrel config validate-skill <name>`) to check:
- Frontmatter is valid YAML with required `name` and `description` fields
- `name` matches the directory name exactly
- Directory contains exactly one `SKILL.md`
- Body structure matches the declared (or detected) complexity tier
- No missing required sections for the tier
- All section headings are valid markdown
- Command snippets are syntactically valid bash

## Example

```markdown
---
name: grill-me
description: Interview the user relentlessly about a plan or design until reaching
  shared understanding, resolving each branch of the decision tree. Use when user
  wants to stress-test a plan, get grilled on their design, or mentions "grill me".
---

Interview me relentlessly about every aspect of this plan until we reach a shared
understanding. Walk down each branch of the design tree, resolving dependencies
between decisions one-by-one. For each question, provide your recommended answer.

Ask the questions one at a time.

If a question can be answered by exploring the codebase, explore the codebase instead.
```
