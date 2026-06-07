---
name: execute-plan
description: >
  Execute an approved implementation plan: validate prerequisites, scaffold
  beads (master epic, phase epics, per-task beads with full descriptions),
  then execute tasks sequentially by claiming, implementing, verifying done
  conditions, and closing each bead. Companion documents (spec, arch-design,
  design study) are loaded for context. Usage: /execute-plan <path-to-plan>
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

### Step 1: Read and validate the plan

Read the plan document. Verify it conforms to the impl-plan doc definition
(`/workspace/folio/doc-definitions/impl-plan_definition.md`):

- Metadata block with `title`, `status`, `date`, `author`.
- Status table with numbered phases and tasks.
- Architecture section with directory layout.

Reject if `status` is `draft` (not reviewed) or `archived` (stale). Accept
`approved` or `issued`.

### Step 2: Find companion document

Read the plan's `spec` and `arch-design` metadata fields. If either points
to a file that exists, present it to the user:

> Found companion document: `<path>`. Is this the right one?

If neither field is set, or the referenced file does not exist, search the
plan's sibling directories for a design study, spec, or arch-design with a
matching topic slug. If found, confirm. If nothing found:

> No companion document found. Proceed without one?

Wait for user confirmation before continuing.

### Step 3: Verify clean working tree

Run `git status`. If there are uncommitted changes, ask the user to commit
or stash before proceeding. Execution must start from a clean state so that
every task's changes are traceable.

## Phase 1: Scaffold Beads

### Step 1: Check for existing beads

Search for a master epic matching the plan's title. If found, proceed to
compliance validation (step 2). If not found, proceed to creation (step 3).

### Step 2: Validate existing beads (if found)

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

### Step 3: Create beads (if not found)

Generate a `bd create --graph` JSON with:

- **1 master epic**: title = plan title, description links to plan file path.
- **1 sub-epic per phase**: `parent-child` to master epic. Sequential
  `blocks` edges between phases per the plan's phase dependency declarations.
- **1 task per plan task**: `parent-child` to its phase epic. `blocks` edges
  per the plan's solution breakdown dependencies.
- **Verification gates**: Tasks whose title starts with "Verify:" are
  **phase gates**. Add a `blocks` edge from the phase epic to each of its
  verification tasks (the phase epic is blocked by the verification task).
  The phase epic cannot be closed until every verification task in it
  passes. Because the next phase's epic is blocked by this phase's epic,
  non-passing verification transitively blocks all subsequent phases.

Every task description **MUST** include:

- **Target files**: exact paths from the plan's directory layout (section
  3.3) and solution breakdown (section 3.5).
- **What to implement**: the change described in the plan's task row and
  solution breakdown.
- **Done conditions**: observable pass/fail conditions from the plan's
  success criteria (section 3.7) and task-specific checks.

After creation, transition the plan's `status` field from `approved` to
`issued`.

## Phase 2: Execute

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

**Non-passing verification blocks the next phase.** Each phase epic has
a `blocks` edge to the next phase's epic. A non-passing verification
task keeps the phase epic open, which prevents `bd ready` from surfacing
any task in subsequent phases. This is intentional — broken phases must
not propagate to downstream work.

Close the master epic only after all phase epics are closed.

## Rules

- **Claim before working.** No code changes without a claimed bead.
- **Never close incomplete work.** A bead is closed only when every done
  condition in its description has been verified. If work cannot be
  completed, leave the bead open and mark it blocked.
- **Companion document is context, not authority.** The plan is the
  execution authority. The companion provides rationale and type definitions.
  When they conflict, create a `discovered-from` issue to track the
  discrepancy.
- **`discovered-from` protocol.** If execution surfaces unplanned work,
  create a new bead immediately, link it `discovered-from` the current task,
  and continue working on the current task.
- **Clean tree between tasks.** Commit before claiming the next task.
- **Read before edit.** Read the plan section and companion section relevant
  to the current task before touching any file.
- **Build after every epic.** When all tasks in a phase epic are closed,
  verify the build passes before closing the epic. Individual tasks require
  a build only when their done conditions explicitly call for it (e.g.,
  unit tests, verification gates).
- **Verification tasks are hard gates.** A "Verify:" task that fails
  blocks its phase epic from closing. Do not skip it, do not defer it, do
  not close the phase without it passing. This is the mechanism that
  prevents broken phases from propagating to downstream work.