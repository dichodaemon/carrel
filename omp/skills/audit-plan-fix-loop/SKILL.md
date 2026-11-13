---
name: audit-plan-fix-loop
description: >
  Run audit-fix-loop on a draft implementation plan with audit-plan as
  the auditor: plan impact categories, implementation decisions recorded
  in the plan's Design Decisions, design gaps sent back to the
  companion, and task renumbering and generated-file fix rules.
  Usage: /audit-plan-fix-loop <path> [--max-audits=N] [--exclude=<topic>...]
---

# Audit Plan Fix Loop

Bring a draft implementation plan to a state where an independent
`audit-plan` audit finds no errors or warnings. This skill runs the
`audit-fix-loop` skill as written, except for the substitutions below.
Each substitution names the `audit-fix-loop` phase and step it
replaces; everything else applies unchanged: one fresh auditor per
round, confirming every lead in the sources, the fan-out and consumer
maps, fixes as leads, the fix ledger, the growth check and the diff
review. A plan says how to build what its companion decided. When an
audit shows that the decision itself is incomplete, the fix belongs in
the companion, not in the plan.

## Trigger

User says "audit and fix the plan until clean", "audit-fix loop on the
plan", "keep auditing and fixing the plan", or invokes
`/audit-plan-fix-loop`.

## Arguments

- **path** (required): an implementation plan with `status: draft`.
- `--max-audits=N` (optional, default 5): the most audits to run.
- `--exclude=<topic>` (optional, repeatable): as in `audit-fix-loop`.

## Phase 0: Setup

### Step 1: Read the governing material

Replaces `audit-fix-loop` Phase 0, Step 1. Read `skill://audit-fix-loop`
in full, then `skill://audit-plan`,
`/workspace/folio/doc-definitions/impl-plan_definition.md`, the plan,
and every companion its metadata names (`design-note`, `arch-design`,
`brief`, `issue`). Stop if the plan's status is not `draft`.

Record the plan's word count as the baseline for `audit-fix-loop`
Phase 3, Step 8. The plan has no decision-change appendix, so the
whole document counts.

### Step 2: Write the auditor brief

Replaces `audit-fix-loop` Phase 0, Step 2, with these changes:

- The auditor reads `skill://audit-plan` and follows all its phases,
  passing the brief as its `brief` argument.
- The doc definition is `impl-plan_definition.md`.
- The yardstick is the companion's decisions and acceptance criteria.
  The plan implements them; the auditor does not reopen them.
- Drop the note about a decision-change appendix.

The evidence requirement, read-only rule, exclusions and reuse rule
apply as in `audit-fix-loop`.

### Step 3: Start the fix ledger

As `audit-fix-loop` Phase 0, Step 3.

## Phase 1: Audit

### Step 1: Spawn one fresh auditor

As `audit-fix-loop` Phase 1. Your own checks while the auditor runs
cover the plan's paths, "before" signatures, callers (`lsp
references`), test targets and fixtures.

## Phase 2: Verify

### Step 1: Confirm, rate and categorize each lead

Replaces the severity source and the impact categories of
`audit-fix-loop` Phase 2, Step 1. Severity is your own judgement under
the `audit-plan` severity table (Phase 5, Step 1). Give each confirmed
or partially confirmed finding one impact category, the first that
fits:

- **Design**: the fix changes something the companion decides: a
  behaviour, rule, threshold, scope or acceptance criterion.
- **Implementation decision**: the fix changes a choice recorded in
  the plan's Design Decisions, such as a parameter position, a phase
  order or where a field lives.
- **Interface**: a type, field, signature, BUILD target, dependency or
  caller list in Interface Changes or a task.
- **Task**: a task that is missing, wrong, in the wrong phase or
  ordered before what it consumes; a phase dependency.
- **Verify**: a Verify: task that is missing, not runnable or does not
  exercise what its phase changed; a missing test task.
- **Cross-reference**: task numbers, section references, done
  conditions, success-criteria mappings, directory layout rows.
- **Companion alignment**: a staleness-audit entry, an unrecorded
  companion choice, a deviation from the companion without its record.
- **Readability or compliance**: wording, structure, doc definition
  rules.

Design and Implementation decision findings are both decision changes
under `audit-fix-loop` Phase 3, Step 4: above 85 confidence apply, 85
or below ask the user.

### Step 2: Decide whether to stop

Replaces the unsettled-design rule of `audit-fix-loop` Phase 2, Step 2.
Count only Design findings of severity error or warning. If this audit
and the two before it each had at least one, apply this round's other
fixes, hold its Design findings, and report (Phase 4). Tell the user
the companion has unsettled cases, list the mechanisms affected, and
recommend running `audit-fix-loop` on the companion before the next
plan audit. The clean rule applies as in `audit-fix-loop`.

## Phase 3: Fix

### Step 1: Snapshot the documents

As `audit-fix-loop` Phase 3, Step 1. When a fix in the round edits the
companion, snapshot the companion too.

### Step 2: Map the fan-out in the plan

Extends `audit-fix-loop` Phase 3, Step 2. Search every section that
can restate a task, file or interface: the phase summary, the status
table, the directory layout, Interface Changes, the solution
breakdown's task lists and done conditions, Design Decisions, Success
Criteria, the staleness audit and Cleanup. When a fix adds, removes,
splits, merges or moves a task, every reference to that task number,
and to every task number after it in the same phase, joins the list.

### Step 3: Map the code consumers

As `audit-fix-loop` Phase 3, Step 3. Run `lsp references` on every
signature the fix changes; each caller gets a task or is confirmed
unaffected in the source.

### Step 4: Decide and write each fix

As `audit-fix-loop` Phase 3, Step 4, plus these rules:

- **Design fixes go to the companion first.** Edit the companion under
  `audit-fix-loop` Phase 3, Steps 4–7, recording the change in the
  companion's decision-change appendix. Then update the plan to match
  and add the companion to the plan's staleness audit as "already
  updated".
- **Implementation decisions stay in the plan.** Rewrite the entry in
  the plan's Design Decisions with its rationale. Record the round,
  before and after, evidence and confidence in the ledger; the plan
  gets no appendix.
- **Run generators, never edit their output.** A task that changes a
  generated file runs its generator; a derived file gets a task that
  creates or runs the transform tool.
- **Keep the plan's invariants.** Every new task names its target files
  and the change; every production logic task has a test task; every
  phase still ends with a Verify: task that names an exact command.
- **Reference, don't copy.** Cite the companion by section number and
  name; never copy its objective, decisions or criteria into the plan.

### Step 5: Apply all locations in one pass

As `audit-fix-loop` Phase 3, Step 5.

### Step 6: Verify the fix against the sources

As `audit-fix-loop` Phase 3, Step 6, plus: every target in a new or
changed Verify: command exists in its BUILD file or is created by an
earlier task, and every renumbered reference resolves to the task it
meant.

### Step 7: Record decision changes

Replaces `audit-fix-loop` Phase 3, Step 7: Design changes are recorded
in the companion's appendix and Implementation decisions in the plan's
Design Decisions and the ledger (Step 4).

### Step 8: Check growth and review the diff

As `audit-fix-loop` Phase 3, Step 8. A task, test task or Verify: task
that a confirmed finding shows missing is a required addition. When
the round edited the companion, the diff review covers both documents.

## Phase 4: Report

### Step 1: Loop or stop

As `audit-fix-loop` Phase 4, Step 1.

### Step 2: Analyze and summarize

As `audit-fix-loop` Phase 4, Step 2, plus a separate list of every
companion change. The plan's status stays `draft`; approval is the
user's.

## Rules

- **`audit-fix-loop` applies except where substituted.** Each
  substitution names the step it replaces.
- **The auditor is `audit-plan`.** Severity comes from its table.
- **Design gaps belong to the companion.** Fix them there first; three
  audits in a row with a Design finding stop the loop.
- **The plan has no appendix.** Implementation decisions live in its
  Design Decisions.
- **Never change the plan's status.** Approval is the user's.
- **No git without approval.** Do not commit or push unless the user
  explicitly asks.
