---
name: execute-plan
description: >
  Execute an approved implementation plan: validate prerequisites, scaffold
  beads (master epic, phase epics, per-task beads with full descriptions),
  then execute tasks sequentially by claiming, implementing, verifying done
  conditions, and closing each bead. Companion documents (spec, arch-design,
  brief) are loaded for context. Usage: /execute-plan <path-to-plan>
---

# Execute Plan

Execute an approved implementation plan end-to-end, tracking every task
through beads.

## Trigger

User says "execute plan", "run the plan", "implement the plan", or invokes
`/execute-plan`.

## Arguments

- Required: path to an implementation plan document.
- Optional: `--dry-run` -- validate plan, find companion, scaffold beads,
  but do not start execution.

## Phase 0: Validate and Prepare

### Plan validation

Read the plan document. Verify it conforms to the impl-plan doc definition
(`/workspace/folio/doc-definitions/impl-plan_definition.md`):

- Metadata block with `title`, `status`, `date`, `author`.
- Status table with numbered phases and tasks.
- Architecture section with directory layout.

Reject if `status` is `draft` (not reviewed) or `archived` (stale). Accept
`approved` or `issued`.

This is a minimum gate check, not a full audit. For comprehensive
validation (structural compliance, internal consistency, companion
alignment, codebase grounding), run `/audit-plan` before approving.

### Companion documents

Read the plan's `spec`, `arch-design`, and `brief` metadata fields. For
each that points to a file that exists, load it silently as context.

If no companion field is set, or none of the referenced files exist, search
the plan's sibling directories for a design study, spec, arch-design, or
brief with a matching topic slug. If found, confirm with the user:

> Found companion document: `<path>`. Is this the right one?

If nothing found:

> No companion document found. Proceed without one?

Wait for user confirmation before continuing.

### Clean working tree

Run `git status`. If there are uncommitted changes, ask the user to commit
or stash before proceeding. Execution must start from a clean state so that
every task's changes are traceable.

## Phase 1: Scaffold Beads

### Existing beads check

Search for a master epic matching the plan's title. If found, proceed to
Bead structure validation. If not found, proceed to Bead scaffolding.

### Bead structure validation

Verify the existing bead structure matches the plan:

- One master epic.
- One sub-epic per phase, each a `parent-child` of the master epic.
- One task per plan task, each a `parent-child` of its phase epic.
- Sequential `blocks` edges between phase epics (P2 blocked by P1, etc.).
- Intra-phase `blocks` edges per the plan's dependency declarations.
- Every task description includes **target files**, **what to implement**,
  and **done conditions**.

If compliant, proceed to Phase 2. If not, report discrepancies and fix them
(add missing beads, wire missing edges, enrich thin descriptions).

### Bead scaffolding

Generate a `bd create --graph` JSON with:

- **1 master epic**: title = plan title, description links to plan file path.
- **1 sub-epic per phase**: `parent-child` to master epic. Sequential
  `blocks` edges between phases per the plan's phase dependency declarations.
- **1 task per plan task**: `parent-child` to its phase epic. `blocks` edges
  per the plan's solution breakdown dependencies.
- **Verification gates**: Tasks whose title starts with "Verify:" are
  **phase gates**. They are `parent-child` of their phase epic like any
  other task. Do NOT add `blocks` edges between phase epics and
  verification tasks — beads forbids `blocks` edges between epics and
  tasks. Phase gating is enforced procedurally: the execution loop (Phase
  2 § Phase gates) halts at each phase boundary until every "Verify:" task
  in the phase passes. Because the next phase's epic is blocked by the
  current phase's epic (via the sequential epic-to-epic `blocks` edges),
  non-passing verification transitively blocks all subsequent phases.

Every task description **MUST** include:

- **Target files**: exact paths from the plan's Architecture section
  (directory layout) and Solution Breakdown section.
- **What to implement**: the change described in the plan's task row and
  solution breakdown.
- **Done conditions**: observable pass/fail conditions from the plan's
  Success Criteria section and task-specific checks.

After creation, transition the plan's `status` field from `approved` to
`issued`. If `--dry-run`, present the scaffolded bead structure and stop.
Do not proceed to Phase 2.

### Bead audit

After beads are created (whether freshly scaffolded or pre-existing),
audit every bead against the plan before proceeding to execution:

1. **Title–plan alignment.** Every bead's title must match the
   corresponding plan task's numbering and summary. A bead titled
   "6.3 Verify: Live gate" must correspond to plan task 6.3 and
   describe the same gate.

2. **Description completeness.** Every task bead's description must
   contain the three required sections (target files, what to implement,
   done conditions). Verify each section is substantive — not a
   one-liner or copy of the title.

3. **Done conditions match plan.** The done conditions in each bead
   must match the plan's success criteria for that task. If the plan
   says "run the live visualizer and confirm rendering", the bead
   must say the same — not a weaker proxy like "run unit tests".

4. **Manual vs automated.** If a done condition requires manual
   verification (e.g., visual inspection, live system test), the bead
   description must explicitly state this. Do not describe manual
   gates with language that implies automated verification.

Report any mismatches and fix them before proceeding to Phase 2.
This audit is mandatory — not a best-effort check.

## Phase 2: Execute

### Precondition

Before executing any task, verify the bead audit (Phase 1 § Bead audit)
is complete. If beads were scaffolded or validated in this session and
no audit was performed, STOP and run it now.

### Execution loop

1. **Find work**: `bd ready` filtered to the master epic's descendants.
   Pick the lowest-numbered unblocked task.

2. **Claim**: `bd update <id> --claim`.

3. **Load context**: Read the task's description (target files, what, done
   conditions). Read the relevant sections of the implementation plan AND
   the companion document (if available) for the current task's phase and
   solution breakdown.

4. **Implement**: Edit files, update BUILD targets, etc.

5. **Verify done conditions**: Execute the checks stated in the task's
   description (build, test, search). Every condition must pass.

6. **Close**: `bd close <id>`. Only after all done conditions are verified.

7. **Commit**: Commit the changes. Commit message references the task
   (e.g., "refactor(fallback): 1.2 move stopping_sampler_types to
   vocabulary").

8. **Loop**: Return to step 1.

### Ordering rules

- Tasks within a phase execute in increasing number order (1.1, 1.2, ...).
- Parallel tasks within the same phase epic are allowed only when both have
  no mutual `blocks` edges and touch disjoint files.
- Cross-phase parallelism is forbidden unless the plan's phase summary
  explicitly declares it.

### Blocked tasks

- If a task cannot be completed, mark it blocked: `bd update <id> --blocked`
  with a note explaining why.
- **Do not close blocked tasks.** Being blocked is acceptable. Marking
  unfinished work as complete is inadmissible.
- Skip the blocked task and continue to the next unblocked task.
- Execution continues until either:
  - (a) All beads including the master epic are closed, or
  - (b) Every remaining task is blocked.

### Phase gates

At each phase boundary (all non-gate tasks in a phase closed), run the
phase's verification tasks ("Verify:" tasks). Every verification task
**MUST** pass before the phase epic can be closed. If a verification
task fails:

1. **Halt.** Do not proceed to the next phase. Do not work around the
   failure by closing the phase epic without a passing gate.
2. **Diagnose** the failure within the current phase's scope.
3. **Fix** the issue — re-open a completed task if needed, or create a
   `discovered-from` bead for unplanned remediation work.
4. **Re-run** the verification task until it passes.
5. **Close** the verification task, then close the phase epic.

Close the master epic only after all phase epics are closed.

## Rules

**Process:**

- **Claim before working.** No code changes without a claimed bead.
- **Clean tree between tasks.** Commit before claiming the next task.
- **Read before edit.** Read the plan section and companion section relevant
  to the current task before touching any file.
- **Build after every epic.** When all tasks in a phase epic are closed,
  verify the build passes before closing the epic. Individual tasks require
  a build only when their done conditions explicitly call for it.

**Invariants:**

- **Never close incomplete work.** A bead is closed only when every done
  condition in its description has been verified. If work cannot be
  completed, leave the bead open and mark it blocked.
- **Verification tasks are hard gates.** A "Verify:" task that fails
  blocks its phase epic from closing. Do not skip it, do not defer it, do
  not close the phase without it passing.

**Conflict resolution:**

- **Companion document is context, not authority.** The plan is the
  execution authority. The companion provides rationale and type definitions.
  When they conflict, create a `discovered-from` issue to track the
  discrepancy.
- **`discovered-from` protocol.** If execution surfaces unplanned work,
  create a new bead immediately, link it `discovered-from` the current task,
  and continue working on the current task.