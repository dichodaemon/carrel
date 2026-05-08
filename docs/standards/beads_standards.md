# Beads Standards

Standards for using beads (`bd`) as the issue tracker and execution planner. Beads is the single system for task tracking, dependency management, and persistent memory. Do not use TodoWrite, markdown checklists, or ad-hoc files.

---

## 1. Workflow

### 1.1. From Companion Document to Issues

When a companion document (arch-design, spec, or design study) exists, this is the procedure:

1. **Read** the companion document's acceptance criteria.
2. **Create one issue per criterion.** Title names the criterion. `--description` references the companion section. `--acceptance` restates the criterion in verifiable form.
3. **Identify shared prerequisites.** When multiple criteria need the same type, function, or build target, create a separate issue for the prerequisite. Its acceptance: "type compiles and is importable" or equivalent.
4. **Wire dependencies.** `bd dep add <criterion-id> <prerequisite-id>` for each criterion that consumes the prerequisite. `bd dep add` between criteria that have ordering constraints.
5. **Start implementing.** `bd ready` shows the first unblocked issue. Claim it, implement it, close it, repeat.

Create all issues upfront before writing any code. This gives the full dependency graph for parallelization and visibility. If an issue needs revision during implementation, update it or create a `discovered-from` child.

**Batch creation.** When creating more than a handful of issues, use `bd create --graph <plan.json>` to create the full issue graph (issues + dependencies) in a single operation. Do not run individual `bd create` and `bd dep add` commands in a loop -- the per-call overhead compounds and the inline arguments bloat context. Write the graph to a JSON file first, then create in one call.

### 1.2. When Beads Replaces a Plan

| Companion exists? | Task complexity | Use |
|---|---|---|
| Arch-design exists | Any | Beads issues. No implementation plan. |
| Spec or design study exists | Multi-step | Beads issues. No implementation plan. |
| None | Trivial (1-3 steps) | Beads issues only. |
| None | Non-trivial | Write an arch-design or spec first. Then beads issues. |

Bug-fix plans are the exception. Root cause analysis, diagnostic evidence, and fix narrative have value that beads issues do not replicate. Use a bug-fix plan for investigations; a beads issue to track the work.

---

## 2. Companion Documents

Issues reference their companion document (arch-design or spec) by section number. The companion says *what*; the issue says *how*.

| Document | Beads' role |
|---|---|
| **Arch-design** | Issues decompose the arch-design into executable steps. Each issue references specific sections (types, contracts, acceptance criteria). |
| **Spec** | Same as arch-design. Issues implement the spec's solution section. |
| **Design study** | Issues may be created as deliverables of a resolved study. Link with `discovered-from` if a study issue exists. |
| **Bug-fix plan** | Beads tracks the work; the plan documents the investigation. |

---

## 3. Issue Creation

### 3.1. Required Fields

- `--title` -- 5-10 words describing what is being done, not how.
- `--description` -- implementation context (see section 3.3).
- `--type` -- `task`, `bug`, `feature`, `epic`, or `chore`.
- `--priority` -- 0-4 (0=critical, 2=medium, 4=backlog). Integers, not words.
- `--acceptance` -- testable done conditions (required for `task` and `feature`; enforced by `bd lint`).

### 3.2. Decomposition

Every acceptance criterion in the companion document gets its own beads issue. The title names the criterion; the description references the companion section; `--acceptance` restates the criterion in verifiable form.

When multiple criteria share a prerequisite (a type, a function, a build target), create a separate issue for the prerequisite and wire it as a `blocks` dependency for each criterion that needs it. This keeps criterion issues focused on verifying observable behavior, not on building infrastructure.

If a single criterion requires multiple implementation steps (defining types, writing the function, writing the test), those steps are child issues linked with `parent-child`.

### 3.3. Description Quality

An issue description must contain enough context for an agent to execute the task without reading the full companion document.

**Always required:**

- **Target files** -- which files to create or modify. Exact paths.
- **What to implement** -- the specific contract, type, or behavior. Reference the companion section.
- **Done conditions** -- what the agent verifies before closing. Use `--acceptance` for the primary conditions.

**When applicable:**

- **Edge cases** -- non-obvious failure modes or boundary conditions.
- **Dependencies on other issues' outputs** -- what this issue consumes that another issue produces. Name the artifacts and formalize with `bd dep add`.

### 3.4. Supplementary Fields

- `--design` -- implementation-level decisions (build system quirks, test fixture choices, migration ordering).
- `--deps` -- dependencies in `type:id` format. Add at creation when known.

```bash
bd create \
  --title="Implement PlanBehavior orchestration" \
  --description="Implement contract 8.1 from the arch-design. Types from sections 7.4-7.11. See section 6.1 for data flow. Files: fallback/behavior/plan_behavior.h, plan_behavior.cc." \
  --acceptance="PlanBehavior returns BehaviorOutput on success path. FakeCoordinator unit test passes." \
  --type=task --priority=1
```

---

## 4. Dependencies

### 4.1. Dependency Types

| Type | Meaning | When to use |
|---|---|---|
| `blocks` | B must complete before A can start. | A consumes types, interfaces, or files that B produces. |
| `discovered-from` | B was discovered while working on A. | New work surfaces during implementation. |
| `related` | Soft connection, no blocking. | Issues touch the same area but neither blocks the other. |
| `parent-child` | Hierarchical. | Multi-step criteria, epics grouping sub-tasks. |

### 4.2. Proactive `discovered-from`

When working on an issue and you discover unplanned work:

1. Create a new issue immediately.
2. Link it: `bd link <new-id> <current-id> --type discovered-from`.
3. Continue working on the current issue.

Do not defer. The new issue captures context while it's fresh. The link preserves provenance.

Examples:

- A test reveals a gap in the acceptance criteria.
- An existing function needs refactoring before the new code can use it.
- A build target dependency is missing.

### 4.3. Arch-Design Deviations

If the implementation needs to differ from the arch-design:

1. **Create a deviation issue** for the code change. Link: `bd link <deviation-id> <current-id> --type discovered-from`.
2. **Create an arch-design update issue** for updating the document. Link: `bd link <update-id> <current-id> --type discovered-from`.
3. **Wire the update as a blocker** for any existing issue that depends on the changed contract: `bd dep add <existing-id> <update-id>`.

Step 3 ensures downstream work waits for the arch-design to reflect the actual contract before proceeding.

### 4.4. Phasing

Use `blocks` dependencies to express ordering. `bd ready` shows only unblocked work.

```bash
bd create \
  --title="Define lane topology types" \
  --description="Define LaneIndex, LaneInfo, DrivingConvention per arch-design section 7.2. File: fallback/behavior/lane_topology.h." \
  --acceptance="Types compile. Unit test for ego-relative indexing passes." \
  --type=task --priority=1
# -> secondary-configs-abc

# (description and acceptance omitted for brevity -- see section 3.1)
bd create --title="Implement lane-follow maneuver" --type=task --priority=1
# -> secondary-configs-def
bd dep add secondary-configs-def secondary-configs-abc

bd create --title="Implement PlanBehavior orchestration" --type=task --priority=1
# -> secondary-configs-ghi
bd dep add secondary-configs-ghi secondary-configs-def
```

---

## 5. Memory

Use `bd remember` to store knowledge that must survive across sessions. Memories are injected at `bd prime` time.

```bash
bd remember "pre-commit hooks fail in core-stack; always use --no-verify" --key core-stack-hooks
bd recall core-stack-hooks       # retrieve by key
bd memories "pre-commit"         # search by keyword
```

**When to remember:**

- Environment quirks that cause repeated failures.
- Implementation decisions too granular for the arch-design but too important to forget.
- Corrections to wrong assumptions.

**When NOT to remember:**

- Information already in the companion document. Don't duplicate.
- Temporary state. Use `bd update --claim` instead.

---

## 6. Lifecycle

### 6.1. Issue Lifecycle

1. **Create before starting.** Every unit of work has an issue.
2. **Claim when working.** `bd update <id> --claim`.
3. **Close when done.** `bd close <id>`. Close immediately.
4. **Do not reopen.** Create a new issue and link with `discovered-from`.

### 6.2. Database Routing

Set `BEADS_DB` explicitly on every `bd` call. The routing table is in the session system prompt (APPEND_SYSTEM.md). Use the database for the workspace where the code change lands.

### 6.3. Hygiene

| Command | Purpose | When |
|---|---|---|
| `bd lint` | Check for missing description sections. | Before starting work on an issue. |
| `bd stale` | Find issues with no recent activity. | At session start. |
| `bd preflight` | Pre-PR checks (lint, stale, orphans). | Before opening a PR. |
| `bd orphans` | Find broken dependency references. | After closing a batch of issues. |

---

## 7. Anti-Patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| **TodoWrite or markdown checklists** | Fragmented. No dependencies or memory. Not queryable. | Use beads exclusively. |
| **Single mega-issue** | Cannot parallelize. Verification bundled. Dependencies invisible. | One issue per acceptance criterion (section 3.2). |
| **One-line description** | Agent cannot execute without reading the full companion document. | Target files, contract reference, done conditions (section 3.3). |
| **Duplicating the arch-design** | Two sources of truth. Description goes stale. | Reference by section number. Add implementation-specific detail only. |
| **Done conditions only in description** | `bd lint` cannot check them. Invisible to `bd ready`. | Use `--acceptance`. |
| **Forgetting `discovered-from`** | New work has no provenance. | Always link discovered work to the issue that surfaced it. |
| **Deferring issue creation** | Context lost. "I'll create it later" -- you won't. | Create immediately. |
| **Closing without verifying** | Acceptance criteria unchecked. | Verify before `bd close`. |
| **Memories in MEMORY.md** | Fragments across accounts. Not injected at session start. | Use `bd remember`. |


---

## 8. Orchestrator Responsibilities

When an orchestrator agent dispatches subagents (via the `task` tool or equivalent), beads protocol still applies. The subagent cannot follow protocol it does not know about.

### 8.1. Required Context for Subagents

The orchestrator **MUST** include the following in every subagent dispatch's shared `context` field:

1. **`discovered-from` protocol** (sections 4.2--4.3, verbatim or summarized). Subagents must know to create issues immediately when they encounter unplanned work, missing prerequisites, or deviations -- not leave TODO comments or skip silently.
2. **BEADS_DB path** for the workspace the subagent is modifying, so it can run `bd create` and `bd link`.
3. **Current issue ID** the subagent is implementing, so `discovered-from` links are wired to the correct parent.
4. **Blocking relationships** the subagent should wire if the discovered work blocks downstream issues. The orchestrator knows the dependency graph; the subagent does not.

### 8.2. Anti-Patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| **TODO instead of issue** | Discovered work is invisible to `bd ready`. No dependency tracking. Lost across sessions. | Subagent creates issue immediately with `discovered-from` link. |
| **Omitting beads context from subagents** | Subagent has no way to follow protocol it was never told about. | Include sections 4.2--4.3 in every dispatch context. |
| **Assuming subagent will read standards** | Subagents do not read folio standards unless explicitly instructed. | Provide the rules inline or by reference with a read directive. |

### 8.3. Verification

After subagent tasks complete, the orchestrator **SHOULD** check whether any subagent left TODO comments indicating discovered work. If found, the orchestrator creates the missing beads issue retroactively and wires dependencies.