---
name: create-plan
description: >
  Create a comprehensive implementation plan from companion documents
  (spec, arch-design, brief, issue) and codebase investigation. Produces
  a draft plan conforming to the impl-plan doc definition, compatible
  with the execute-plan skill. Usage: /create-plan <companion-path> [output-dir]
---

# Create Plan

Create an implementation plan document from companion documents and
codebase context. The plan is produced in `draft` status for user review.

## Trigger

User says "create plan", "write a plan", "prepare a plan", or invokes
`/create-plan`.

## Arguments

- Required: path to a companion document (spec, arch-design, brief, or
  issue document). Multiple companions may be provided.
- Optional: output directory for the plan file. Defaults to a `plans/`
  sibling of the companion's directory.

## Phase 0: Read the Definition

### Step 1: Load the doc definition

Read `/workspace/folio/doc-definitions/impl-plan_definition.md` in full.
This is the normative reference. Do not rely on memory of the format.

### Step 2: Load the document structure standards

Read `/workspace/folio/standards/document-structure_standards.md` for
file naming conventions, date rules, and directory placement.

### Step 3: Check for a glossary

Look for `docs/glossary.md` at the companion document's level or above.
If found, read it and use canonical terms throughout the plan.

## Phase 1: Gather Inputs

### Step 1: Read companion documents

Read every companion document provided by the user. Classify each:

- `spec` -- what to build and why
- `arch-design` -- types, contracts, decomposition
- `brief` -- investigation findings, root cause analysis, benchmarks
- `issue` -- tracking ticket or issue document

At least one companion is required. If none is provided, ask the user.

### Step 2: Extract the contract

From the companion documents, extract:

- **Types and structs** that will be created, modified, or removed.
- **Functions and methods** that will be created, modified, or removed.
- **Signatures** with exact parameter lists.
- **Invariants** and constraints the implementation must satisfy.
- **Acceptance criteria** from the spec or brief.

### Step 3: Determine output path

Compute the plan filename: `YYYY-MM-DD_<topic>_plan.md` using today's
date and the topic slug from the companion. Determine the output
directory (user-provided or `plans/` sibling of the companion).

## Phase 2: Codebase Investigation

This phase prevents the most common plan failure: writing from the
companion document alone without verifying the code matches.

### Step 1: Read every source file named in the companions

For every file path mentioned in the companion documents, read the
relevant sections. Verify that the types, functions, and signatures
described in the companion actually exist in the codebase and match.

If they diverge, note the divergence -- it may affect the plan's
interface changes section or require a discovered-from bead.

### Step 2: Map callers and consumers

For every symbol (type, function, field) that will be modified or
removed, run `lsp references` to find all callers. Record:

- Production callers (these drive the solution breakdown)
- Test callers (these drive the test update tasks)
- Dead callers (these may be cleanup candidates)

### Step 3: Map test coverage

For every production file that will be modified, find its corresponding
test file. Record whether tests exist and what they cover. This drives
the test-writing vs test-updating distinction in the status table.

### Step 4: Map BUILD files

For every production file that will be modified, find its BUILD file.
Record which deps will need adding or removing.

### Step 5: Identify existing patterns

Read the files surrounding the change to identify conventions:

- Observer/instrumentation patterns (if adding instrumentation)
- Test helper patterns (if updating tests)
- Error handling patterns (if adding error paths)
- Naming conventions for timing labels, test names, etc.

## Phase 3: Draft the Plan

Write each section per the doc definition. Follow this order:

### Step 1: Metadata

```yaml
title: <Topic> -- Implementation Plan
status: draft
date: <today>
author: <git config user.name>
spec: <relative path if provided>
arch-design: <relative path if provided>
brief: <relative path if provided>
issue: <reference if provided>
```

### Step 2: Implementation Status

1. Define phases by analyzing the dependency graph from Phase 2:
   - Data model / vocabulary changes first (no deps)
   - Logic changes that consume new types (depends on data model)
   - Deletion / cleanup (depends on logic changes removing callers)
   - Test updates (depends on logic changes and deletions)
   - Build and verification (depends on everything)

2. For each phase, list tasks with two-level numbering. Each task row
   must name the target file(s) and the change.

3. For each implementation task that changes production logic, add a
   "Write test:" task.

4. End each phase with a "Verify:" task naming the exact build or test
   command.

5. Add end-to-end verification tasks in the final phase (full build,
   integration tests, scenario parity, profiling checks).

### Step 3: Architecture

- Directory layout table: one row per file created, modified, or deleted.
- Dependency graph: only when inter-package edges change.

### Step 4: Interface Changes

One subsection per interface element, grouped by file:

- **Types/fields added:** exact struct definition with annotations.
- **Types/fields removed:** table with field, reason, and zero-caller
  confirmation.
- **Functions added:** signature, purpose, callers.
- **Functions modified:** before/after signature, what changed, callers
  to update.
- **Functions removed:** replacement (if any), zero-caller confirmation.

### Step 5: Solution Breakdown

One subsection per logical unit of work. Each must include:

- Function signature or location
- Step-by-step logic description
- Edge cases
- Dependencies (produces/requires)
- Done condition cross-referencing a Verify: task

### Step 6: Design Decisions

One subsection per non-obvious implementation choice:

- The decision
- What was considered
- Why this choice was made

### Step 7: Success Criteria

Per-component checklist. Every criterion must trace to a Verify: task
in the status table. State this cross-reference explicitly.

### Step 8: Document Staleness Audit

1. List every companion and related document.
2. For each, state whether the plan's changes invalidate content.
3. If invalidated, add an update task to the status table.

### Step 9: Cleanup

If the work adds diagnostic instrumentation, list it with file, line,
and action (remove / downgrade). Include cleanup tasks in the status
table.

## Phase 4: Self-Audit

Before presenting the plan, verify it against these checklists. Fix
any failures before proceeding.

### Checklist 1: Doc-definition compliance

- [ ] Metadata has `title`, `status: draft`, `date`, `author`.
- [ ] At least one companion field is set (`spec`, `arch-design`,
      `brief`, or `issue`).
- [ ] Status table has phase summary with dependency declarations.
- [ ] Every task row names target file(s).
- [ ] Every production logic task has a corresponding test task
      (write or update).
- [ ] Every phase ends with a Verify: task.
- [ ] Every Verify: task names an exact command or observable condition.
- [ ] Architecture section has a directory layout table.
- [ ] Interface changes section covers types AND functions.
- [ ] Every solution breakdown subsection has a done condition.
- [ ] Every success criterion traces to a Verify: task.
- [ ] Document staleness audit is complete.

### Checklist 2: Arch-design alignment

For every interface change (new type, modified signature, new field):

- [ ] The change is consistent with the arch-design's type definitions
      and contracts.
- [ ] If the change deviates from the arch-design, the deviation is
      flagged in the design decisions section with a rationale.
- [ ] If the arch-design needs updating, the staleness audit includes
      it and the status table has an update task (or the update was
      done pre-plan).

### Checklist 3: Execute-plan compatibility

The execute-plan skill requires every bead description to include
target files, what to implement, and done conditions. Verify:

- [ ] Every task in the status table has enough detail to generate a
      bead description without going back to the solution breakdown.
- [ ] Verify: tasks are distinguishable by their "Verify:" prefix.
- [ ] Phase dependencies are explicit in the phase summary (execute-plan
      wires `blocks` edges from these declarations).
- [ ] Intra-phase dependencies are explicit in the solution breakdown
      (execute-plan wires `blocks` edges from these).

### Checklist 4: Codebase grounding

- [ ] Every file path in the directory layout exists in the codebase
      (or is marked "New file").
- [ ] Every function signature in the interface changes section matches
      the current codebase (not the companion document's description).
- [ ] Every caller listed for modified functions was found via
      `lsp references`, not assumed.
- [ ] Every test file listed exists and was read.

## Phase 5: Present

### Step 1: Write the plan file

Write the plan to the computed output path. Status is `draft`.

### Step 2: Summarize

Present a summary to the user:

- Plan path
- Number of phases, tasks, verify gates, test tasks
- Key design decisions that need review
- Any arch-design deviations flagged
- Any divergences between companion documents and codebase

### Step 3: Wait for approval

The plan is a `draft` deliverable. Do not change status. The user
reviews and approves (or requests changes). Only after approval does
the status transition to `approved` and the plan becomes eligible for
execute-plan.

## Rules

- **Read the doc-definition first, every time.** Do not rely on memory
  of the format. The definition is the authority.
- **Read the codebase before writing.** Every type, function, and file
  path in the plan must be verified against the actual code. Hope is
  not a strategy.
- **Every design decision must be verified against the arch-design.**
  If misaligned, flag it for discussion before writing into the plan.
  Do not silently deviate.
- **The plan is a draft.** The user approves it. Do not change status
  from `draft` or begin execution.
- **Companion documents are context, not the plan.** Do not duplicate
  investigation findings, acceptance criteria, or design rationale from
  the companion. Reference them.
- **Maximize execute-plan compatibility.** The plan's primary consumer
  is the execute-plan skill. Every structural choice should make
  execute-plan's job easier: clear task boundaries, explicit done
  conditions, exact file paths, Verify: gates.
