---
name: audit-plan
description: >
  Independently audit a draft implementation plan for structural
  compliance, internal consistency, companion alignment, and codebase
  grounding. Severity by impact; every factual finding carries verbatim
  evidence. Usage: /audit-plan <path-to-plan> [brief]
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

- **path** (required): path to an implementation plan document.
- **brief** (optional): companions, the yardstick, exclusions and
  sections the author requested. The brief's companions add to those
  in the plan's metadata. Topics the brief excludes are never reported.
  Sections the brief marks as requested are not flagged for existing.

## Phase 0: Load Inputs

### Step 1: Read the doc definition

Read `/workspace/folio/doc-definitions/impl-plan_definition.md` in
full. This is the normative reference for what the plan must contain.

### Step 2: Read the plan

Read the plan document. Reject if `status` is not `draft` — there is
nothing to audit on `approved`, `issued`, or `archived` plans.

### Step 3: Read companion documents

Read every companion referenced in the plan's metadata (`design-note`,
`arch-design`, `brief`, `issue`) or named by the brief. If a
referenced file does not exist, record an error finding immediately.

### Step 4: Check for glossary

Look for `docs/glossary.md` at the plan's level or above. If found,
read it — term misuse is an info-level finding.

## Phase 1: Structural Compliance

Verify the plan has every required section with correct format per the
doc definition.

### Metadata

- [ ] Metadata has `title`, `status`, `date`, `author`.
- [ ] Title ends with "-- Implementation Plan".
- [ ] Date matches `YYYY-MM-DD` format and the filename date prefix.
- [ ] At least one companion field is set (`design-note`,
      `arch-design`, `brief`, or `issue`).

### Status table

- [ ] Status table has a phase summary with dependency declarations.
- [ ] Tasks use two-level numbering (N.M) where the first digit is
      the phase number.
- [ ] Every task row names target file(s) and the change.
- [ ] Every production logic task has a corresponding test task
      (write or update).
- [ ] Every phase ends with a "Verify:" task.
- [ ] Every "Verify:" task names an exact command or observable
      pass/fail condition.

### Sections

- [ ] Architecture section exists with a directory layout table.
- [ ] Interface changes section covers types AND functions (not just
      data model).
- [ ] Every solution breakdown subsection has a done condition naming
      a Verify: task.
- [ ] Document staleness audit is present.

## Phase 2: Internal Consistency

Cross-reference sections against each other.

### Status table ↔ solution breakdown

- [ ] Every task in the status table has a corresponding solution
      breakdown subsection (or is grouped with one that names it).
- [ ] Phase dependencies in the summary match the dependency
      declarations in the solution breakdown ("Requires:" / "Produces:"
      annotations).

### Status table ↔ directory layout

- [ ] Every file in the directory layout is touched by at least one
      task in the status table.
- [ ] Every task's target file appears in the directory layout.

### Interface changes ↔ tasks

- [ ] Every type and function in the interface changes section is
      referenced by at least one task.
- [ ] Every removed function/field has a task that performs the
      removal.

### Cross-reference integrity

- [ ] The staleness audit covers all companion documents listed in
      the metadata.
- [ ] Design decisions reference the correct task numbers.
- [ ] Every solution breakdown done condition and every success
      criterion names a Verify: task that exists, by its current
      number.

## Phase 3: Companion Alignment

For each companion type present in the plan's metadata or the brief,
run the appropriate checks.

### Contract completeness (all companion types)

For every normative contract file (schema, interface definition,
config format) referenced by any companion document:

- [ ] Read the contract file in full.
- [ ] For every field, type, and constraint in the contract, verify
      one of:
      (a) A plan task produces conforming output for this field, or
      (b) The current code already conforms (spot-check in code), or
      (c) The field is not applicable to this producer and the plan
          states why.
- [ ] Any field with no coverage and no justification is a finding:
      "Schema X field Y not addressed by any task."
- [ ] If the plan cites a companion's prose summary of a contract
      but the contract file has additional fields beyond those
      examples, the uncovered fields are findings unless explicitly
      scoped out in the plan's design decisions.

### Unresolved companion choices

Scan every companion document for passages that offer multiple
approaches or alternatives:

- [ ] For each such passage, verify the plan's Design Decisions
      section records which alternative was chosen and why.
- [ ] If no Design Decision addresses the choice, that is a finding:
      "Companion C §N offers alternatives [X, Y]; plan silently chose
      X without recording the decision."
- [ ] If the plan chose the companion's non-preferred alternative,
      verify the Design Decision explains why.

### If `design-note` is set

- [ ] Every acceptance criterion in the design note maps to at least
      one task and at least one Verify: gate in the plan.
- [ ] Non-goals stated in the design note are not implemented by any
      task in the plan.
- [ ] Every decision in the design note is implemented as decided, or
      the deviation is recorded in the plan's design decisions with a
      rationale and the staleness audit has a task that updates the
      note.
- [ ] Every type, field, config key and interface the design note
      outlines is consistent with the plan's interface changes, or the
      difference is recorded as a deviation as above.
- [ ] The plan does not introduce scope beyond the design note without
      flagging it in the design decisions section.
- [ ] The plan references the design note's objective, decisions and
      criteria instead of copying them.

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

### If `issue` is set

- [ ] The plan's scope matches the issue's description (not
      materially over- or under-scoped).
- [ ] Any sub-tasks or acceptance criteria listed in the issue are
      covered by plan tasks.

### Artifact provenance

For every task in the status table that creates or modifies a data
file (JSON, JSONL, proto text, config, test fixture):

- [ ] Classify the file's provenance using the categories from
      create-plan Phase 2 Step 7 (authored, generated, or derived).
- [ ] For generated files, identify the generator. Verify the plan
      runs the generator rather than hand-editing the output. A task
      that hand-edits a generated file is a finding: "Task X.Y
      hand-edits generated file F; must run generator G instead."
- [ ] For derived files, verify the plan includes a task to create
      or run the transform tool. A task that hand-writes a derived
      artifact without a tool is a finding.

## Phase 4: Codebase Grounding

Verify the plan's claims against the actual codebase. Check every
path, signature, caller, test target and fixture; do not sample. When
one item fails, check all its siblings (every caller of the symbol,
every fixture of the format) and report every failing case in one
finding.

### File paths

For every file path in the directory layout:

- [ ] The file exists in the codebase, or the row says "New file".
- [ ] The described change is plausible given the file's current
      content (e.g., "remove field X" — does field X exist?).

### Function signatures

For every function in the interface changes section:

- [ ] **Added:** the function does not already exist (or the plan
      says "rewrite" not "add").
- [ ] **Modified:** the "before" signature matches the current code.
- [ ] **Removed:** the function currently exists.

### Caller verification

For every modified or removed function:

- Run `lsp references` on the symbol.
- [ ] Every caller found by LSP is accounted for in the plan (either
      updated by a task, or confirmed unaffected).
- [ ] The plan does not list callers that LSP does not find (phantom
      callers).

### Test files

For every test file mentioned in the plan:

- [ ] The test file exists, or a task creates it.
- [ ] The test target mentioned in Verify: tasks is a valid build
      target, or a task creates it.

### Test fixture conformance

If the plan modifies output formats and Verify: tasks use test data
fixtures (e.g., JSON files, JSONL recordings, proto text fixtures)
for validation:

- [ ] Identify every test fixture file used by Verify: tasks or
      referenced in solution breakdown done conditions.
- [ ] For each fixture, determine the target schema it should
      conform to after the plan's changes.
- [ ] Read the fixture and verify it conforms to the target schema.
      Stale field names, wrong structure, or wrong formats for
      polylines and trajectories are findings.
- [ ] If a fixture is non-conforming, verify the plan includes a
      task to update it. If not, that is a finding: "Fixture X used by
      Verify: task Y.Z does not conform to schema S and no task
      updates it."

## Phase 5: Report

### Step 1: Compile findings

Group findings by phase. Each finding has:

- **Severity**, by impact, whatever the phase:

  | Severity | Meaning | Examples |
  |---|---|---|
  | `error` | Executing the plan as written fails, builds or tests the wrong thing, produces incorrect beads, or implements something the companion did not decide. | A wrong path or "before" signature; an unlisted caller that breaks the build; a missing or non-runnable Verify: task; a production task without a test task; a dropped acceptance criterion or brief finding; a non-goal implemented; an unrecorded deviation from a companion decision or contract; a silently chosen companion alternative; a hand-edited generated file; a phase that depends on a later phase's output. |
  | `warning` | The plan executes correctly but a reader or the bead handoff is misled. | A stale task or section cross-reference; a task missing from the directory layout or the solution breakdown; a non-preferred companion alternative chosen without its reason; a staleness-audit entry without an action; a phantom caller. |
  | `info` | A concrete edit nothing depends on. | Wording; an avoided glossary alias. |

- **Location**: section name and task number (e.g., "Status table,
  task 2.3" or "Interface changes, §3.4").
- **Description**: what is wrong. Every finding that states a fact
  about code, a companion or the doc definition cites the path, line
  numbers and a verbatim quote.
- **Suggested fix**: the fix, stated as a directive.

### Step 2: Root cause analysis

Look for clusters: several findings that share one cause, such as one
missing interface change behind several missing tasks and stale
cross-references. State the cause and the findings it would resolve.

### Step 3: Present report

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
1. [Interface changes, §3.2] Caller `foo_test.cc` not listed for
   modified function `Bar()`. Found via lsp references.
   Fix: Add test update task for `foo_test.cc`.

### Info
1. [Metadata] Brief companion exists but `brief` field not set.
   Fix: Add `brief: <path>` to metadata.

### Root causes
1. <cause>: resolves errors 1, 3 and warning 2.
```

### Step 4: Verdict

- **PASS**: zero errors. The plan is ready for the user to approve
  (transition to `approved` status).
- **FAIL**: one or more errors. The plan must be fixed before approval.
  List the errors and suggested fixes.

Do not modify the plan. Do not change its status. The author fixes
the errors (or `audit-plan-fix-loop` does) and re-invokes the audit.

## Rules

- **Read everything from scratch.** The auditor has no context from
  the authoring session. Every claim in the plan is verified
  independently.
- **The auditor does not modify the plan.** It reports findings. The
  author (or create-plan) fixes them.
- **Severity by impact.** Phase 5, Step 1 sets every finding's
  severity, whichever phase found it.
- **Source-grounded, never inferred.** Every finding that states a fact
  cites path, line numbers and a verbatim quote. Call relations,
  ordering and ownership are shown by the call chain read in the
  source, never inferred from names, comments or file position. A
  claim you did not verify is not a finding.
- **Every case, no sampling.** Check every path, signature, caller,
  test target and fixture; report all failing cases of one gap in one
  finding.
- **Run `lsp references` for every modified or removed symbol.** Do
  not trust the plan's caller lists. Verify them.
- **Check companion alignment for every companion type present.** Do
  not skip a companion because it's "just a brief" or "just an issue."
  Each type has specific checks.
- **Honour the brief.** Never report an excluded topic or flag a
  requested section.
- **The companion's decisions are the yardstick.** Judge whether the
  plan implements them; do not reopen them.
