---
name: audit-plan
description: >
  Independently audit an implementation plan against the doc definition,
  companion documents, and codebase. Produces a structured findings
  report with errors, warnings, and info. The gate between draft and
  approved. Usage: /audit-plan <path-to-plan>
---

# Audit Plan

Independently review an implementation plan for structural compliance,
internal consistency, companion alignment, and codebase grounding.
The auditor reads everything from scratch — no context from the
authoring session.

## Trigger

User says "audit plan", "review the plan", "check the plan", or
invokes `/audit-plan`.

## Arguments

- Required: path to an implementation plan document.

## Phase 0: Load Inputs

### Step 1: Read the doc definition

Read `/workspace/folio/doc-definitions/impl-plan_definition.md` in
full. This is the normative reference for what the plan must contain.

### Step 2: Read the plan

Read the plan document. Reject if `status` is not `draft` — there is
nothing to audit on `approved`, `issued`, or `archived` plans.

### Step 3: Read companion documents

Read every companion referenced in the plan's metadata (`spec`,
`arch-design`, `brief`, `issue`). If a referenced file does not exist,
record an error finding immediately.

### Step 4: Check for glossary

Look for `docs/glossary.md` at the plan's level or above. If found,
read it — term misuse is an info-level finding.

## Phase 1: Structural Compliance

Verify the plan has every required section with correct format per the
doc definition. Each failure is an error-level finding.

- [ ] Metadata has `title`, `status`, `date`, `author`.
- [ ] Title ends with "-- Implementation Plan".
- [ ] Date matches `YYYY-MM-DD` format and the filename date prefix.
- [ ] At least one companion field is set (`spec`, `arch-design`,
      `brief`, or `issue`).
- [ ] Status table has a phase summary with dependency declarations.
- [ ] Tasks use two-level numbering (N.M) where the first digit is
      the phase number.
- [ ] Every task row names target file(s) and the change.
- [ ] Every production logic task has a corresponding test task
      (write or update).
- [ ] Every phase ends with a "Verify:" task.
- [ ] Every "Verify:" task names an exact command or observable
      pass/fail condition.
- [ ] Architecture section exists with a directory layout table.
- [ ] Interface changes section covers types AND functions (not just
      data model).
- [ ] Every solution breakdown subsection has a done condition
      cross-referencing a Verify: task.
- [ ] Every success criterion traces to a specific Verify: task.
- [ ] Document staleness audit is present.

## Phase 2: Internal Consistency

Cross-reference sections against each other. Each mismatch is a
warning-level finding.

- [ ] Every task in the status table has a corresponding solution
      breakdown subsection (or is grouped with one that names it).
- [ ] Every file in the directory layout is touched by at least one
      task in the status table.
- [ ] Every task's target file appears in the directory layout.
- [ ] Every type and function in the interface changes section is
      referenced by at least one task.
- [ ] Every removed function/field has a task that performs the
      removal.
- [ ] Phase dependencies in the summary match the dependency
      declarations in the solution breakdown ("Requires:" / "Produces:"
      annotations).
- [ ] The staleness audit covers all companion documents listed in
      the metadata.
- [ ] Design decisions reference the correct task numbers.
- [ ] Success criteria table references match actual Verify: task
      numbers (no stale cross-references).

## Phase 3: Companion Alignment

For each companion type present in the plan's metadata, run the
appropriate checks. Misalignment is an error-level finding.

### If `arch-design` is set

- [ ] Every type or struct in the interface changes section is
      consistent with the arch-design's type definitions.
- [ ] Every new or modified function signature is consistent with the
      arch-design's contracts (section numbers, parameter lists).
- [ ] Every design decision in the plan aligns with the arch-design,
      or the deviation is explicitly flagged in the design decisions
      section with a rationale.
- [ ] If the arch-design needs updating to match the plan, the
      staleness audit includes it with an action (update task or
      "already updated").

### If `brief` is set

- [ ] Every bug or finding described in the brief is addressed by at
      least one task in the status table. No dropped findings.
- [ ] Fixes the brief explicitly rejected are not re-introduced by
      the plan.
- [ ] The plan's solution breakdown approach matches the brief's
      proposed fixes (the plan is solving the problem the brief
      identified, not a different one).
- [ ] Constraints stated in the brief (sign conventions, data flow
      invariants, performance requirements) are respected in the
      interface changes and solution breakdown.
- [ ] Diagnostic evidence from the brief (expected values, cycle
      numbers, metric names) is reflected in the plan's Verify: tasks
      and success criteria where applicable.

### If `spec` is set

- [ ] Every acceptance criterion in the spec maps to at least one
      task and at least one Verify: gate in the plan.
- [ ] Non-goals stated in the spec are not implemented by any task
      in the plan.
- [ ] The plan does not introduce scope beyond the spec without
      flagging it in the design decisions section.

### If `issue` is set

- [ ] The plan's scope matches the issue's description (not
      materially over- or under-scoped).
- [ ] Any sub-tasks or acceptance criteria listed in the issue are
      covered by plan tasks.

## Phase 4: Codebase Grounding

Verify the plan's claims against the actual codebase. Each false claim
is an error-level finding.

### Step 1: File paths

For every file path in the directory layout:

- [ ] The file exists in the codebase, or the row says "New file".
- [ ] The described change is plausible given the file's current
      content (e.g., "remove field X" — does field X exist?).

### Step 2: Function signatures

For every function in the interface changes section:

- [ ] **Added:** the function does not already exist (or the plan
      says "rewrite" not "add").
- [ ] **Modified:** the "before" signature matches the current code.
- [ ] **Removed:** the function currently exists.

### Step 3: Caller verification

For every modified or removed function:

- Run `lsp references` on the symbol.
- [ ] Every caller found by LSP is accounted for in the plan (either
      updated by a task, or confirmed unaffected).
- [ ] The plan does not list callers that LSP does not find (phantom
      callers).

### Step 4: Test files

For every test file mentioned in the plan:

- [ ] The test file exists.
- [ ] The test target mentioned in Verify: tasks is a valid build
      target.

## Phase 5: Report

### Step 1: Compile findings

Group findings by phase. Each finding has:

- **Severity**: `error` (blocks approval), `warning` (needs
  acknowledgment), `info` (advisory).
- **Location**: section name and task number (e.g., "Status table,
  task 2.3" or "Interface changes, SS3.4").
- **Description**: what is wrong.
- **Suggested fix**: how to resolve it.

### Step 2: Present report

Format:

```
## Audit Report: <plan title>

### Summary
- Errors: N
- Warnings: N
- Info: N
- Verdict: PASS / FAIL

### Errors
1. [Status table, task 2.3] No Verify: task after task 2.3.
   Fix: Add "Verify: bazel test //pkg:target passes" as task 2.4.

### Warnings
1. [Interface changes, SS3.2] Caller `foo_test.cc` not listed for
   modified function `Bar()`. Found via lsp references.
   Fix: Add test update task for `foo_test.cc`.

### Info
1. [Metadata] Brief companion exists but `brief` field not set.
   Fix: Add `brief: <path>` to metadata.
```

### Step 3: Verdict

- **PASS**: zero errors. The plan is ready for the user to approve
  (transition to `approved` status).
- **FAIL**: one or more errors. The plan must be fixed before approval.
  List the errors and suggested fixes.

Do not modify the plan. Do not change its status. The author fixes
the errors and re-invokes the audit.

## Rules

- **Read everything from scratch.** The auditor has no context from
  the authoring session. Every claim in the plan is verified
  independently.
- **The auditor does not modify the plan.** It reports findings. The
  author (or create-plan) fixes them.
- **Error means the plan will fail during execute-plan** or produce
  incorrect beads. A missing Verify: task, a wrong file path, or a
  dropped brief finding are errors.
- **Warning means the plan could work but is risky.** A missing
  cross-reference, an unlisted caller, or a stale task number are
  warnings.
- **Info is advisory.** Term misuse, style suggestions, or minor
  improvements that don't affect correctness.
- **Run `lsp references` for every modified or removed symbol.** Do
  not trust the plan's caller lists. Verify them.
- **Check companion alignment for every companion type present.** Do
  not skip a companion because it's "just a brief" or "just an issue."
  Each type has specific checks.
