---
name: audit-fix-loop
description: >
  Repeatedly audit a doc-definition-backed document with independent
  audit-document subagents. Treat their findings as leads, confirm each
  one against the sources yourself, and fix every confirmed finding
  everywhere it occurs. Stop when an audit finds no errors or warnings,
  or when the audit cap is reached.
  Usage: /audit-fix-loop <path> [--max-audits=N] [--exclude=<topic>...]
---

# Audit Fix Loop

Bring a document to a state where an independent audit finds no errors
or warnings. Each round, a fresh subagent audits the document with the
`audit-document` skill. The orchestrating agent confirms every lead
against the sources itself, fixes each confirmed finding completely,
and starts the next round. Subagents only produce leads. The
orchestrator alone is responsible for whether findings are real and
whether fixes are correct. The failure this skill prevents is the
incomplete fix: a fact corrected in one place while its other
statements stay stale, so that each later audit flags another copy.

## Trigger

User says "audit and fix until clean", "audit-fix loop", "keep
auditing and fixing", "loop audits on this document", or invokes
`/audit-fix-loop`.

## Arguments

- **path** (required): the document to audit and fix.
- `--max-audits=N` (optional, default 10): the most audits to run. A
  subagent that fails before producing a report does not count as an
  audit.
- `--exclude=<topic>` (optional, repeatable): finding topics the user
  has put out of scope, for example document length. Every auditor is
  told about them and must not report them. The orchestrator discards
  any finding on an excluded topic.

## Phase 0: Setup

### Step 1: Read the governing material

Read the `audit-document` skill, the doc definition for the document's
type, and the full document. Note the document's companions, the code
it describes and the external sources it cites, such as requirement
exports. The fixes in Phase 3 are judged against these.

### Step 2: Fix the auditor brief

Write one auditor brief and reuse it unchanged in every round. It
contains:

- The document path and the instruction to read `skill://audit-document`
  and follow all its phases.
- The doc definition path. Name it explicitly when the document type is
  missing from the skill's type table.
- The codebase root, the companions, the cited external sources and the
  glossary.
- The yardstick: what the document is meant to achieve. Without it, an
  auditor flags deliberate omissions that the document justifies.
- The exclusions from `--exclude`. Also, if the decision-change
  appendix (Phase 3, Step 4) exists, a note that it is requested and
  must not be flagged.
- The evidence requirement: every soundness and correctness finding
  cites the source path, line numbers and a short quote, plus the
  document's line numbers. Findings without evidence are discarded.
- Read-only: the auditor modifies nothing and runs no builds or tests.

The brief never contains earlier findings, earlier fixes or the
orchestrator's opinions. Those would bias the audit.

## Phase 1: Audit

### Step 1: Spawn one fresh auditor

Spawn a new subagent (the `reviewer` agent, or `task` if `reviewer` is
unavailable) with the brief. Use a new agent every round. Never message
an earlier auditor or reuse its context: an auditor that has seen
earlier rounds is not independent.

If the subagent fails before reporting (a model or permission error,
for example), fix the cause and spawn again. A failed spawn is not an
audit and does not count toward `--max-audits`.

### Step 2: Use the waiting time

While the auditor runs, check the document's factual claims yourself:
existing symbols, file locations, behaviour and cited requirements.
Record each claim you confirm as false. These checks are your own
leads, and they go through Phase 2 and Phase 3 like the auditor's.

## Phase 2: Verify

### Step 1: Treat every finding as a lead

Nothing an auditor says is a fact until you confirm it yourself. Do not
infer, and do not trust the agent's quote or line numbers.

### Step 2: Confirm or reject each lead

For each finding, read the cited source yourself: the code at the given
file and lines, the requirement text, the doc definition or the
glossary. Then classify it:

- **Confirmed**: the source shows the document is wrong, contradictory,
  incomplete or non-compliant as described.
- **Partially confirmed**: the problem is real but the auditor's
  description or recommendation is wrong. Record the corrected version.
- **Rejected**: the source does not support the finding, the finding is
  on an excluded topic, or it asks to undo a decision whose stated
  trade-off holds. Record the reason.

Severity is your own judgement under the `audit-document` severity
table, not the auditor's.

### Step 3: Decide the stop condition

The round is clean when no confirmed or partially confirmed finding of
severity error or warning remains. A clean round ends the loop (Phase
4). Info findings do not block, but fix the confirmed ones anyway.

## Phase 3: Fix

### Step 1: Map the fan-out of each finding

Before editing, search the whole document for every statement of the
affected fact, rule, name or value: synonyms, paraphrases, tables,
diagrams, type and config outlines, risks, acceptance criteria,
approach, open questions and appendices. List every location. A fix
that misses a location is incomplete.

### Step 2: Classify each fix

- **Fact or wording fix**: corrects a claim, a name, a reference or the
  presentation without changing what the document decides. Apply it
  without asking.
- **Decision change**: changes a settled decision, its mechanism, its
  scope or its acceptance criteria. Before acting, state a confidence
  from 0 to 100 that the change is correct. Base it only on facts you
  confirmed yourself in the sources, never on inference or subagent
  output.
  - **Above 85**: apply it, and record it in the appendix (Step 4).
  - **85 or below**: stop and ask the user. Give the finding, the
    evidence, the proposed change, the alternatives and your
    confidence with its basis. Continue with the other fixes while
    waiting, but do not start the next audit until the answer is in.

### Step 3: Apply all locations in one pass

Edit every location from Step 1 together. Follow the doc definition's
rules: section structure, citations by section number and name rather
than line number, glossary terms. Then re-read each edited section in
full, and every section that refers to it, and confirm they agree with
each other.

### Step 4: Record decision changes in an appendix

Keep a final appendix in the document, for example "Appendix A.
Decision changes during audit". Add one entry per decision change:

- the audit round;
- the affected decision, by section number and name;
- the change, before and after;
- the evidence: source paths and the facts confirmed;
- the confidence, and whether the user approved the change.

Create the appendix with the first decision change. Never remove
entries.

## Phase 4: Report

### Step 1: Loop or stop

If the round was not clean and the audit count is below
`--max-audits`, return to Phase 1. Otherwise stop.

### Step 2: Analyze and summarize

Report to the user:

- One row per round: the audit number, the findings reported, the
  findings confirmed (with severity), the findings rejected (with the
  reason), and the fixes applied.
- The decision changes, with their confidence and approval status.
- Recurring clusters: findings that came back across rounds, and their
  root cause.
- The stop reason: a clean audit, or the cap reached with the
  findings still open.
- Anything the user must still decide.

## Rules

- **Subagents produce leads, never facts.** Confirm every claim against
  the source yourself before acting on it. Agent output is not
  evidence.
- **You alone own correctness.** You are accountable for every
  confirmed finding and every fix, whatever an auditor said.
- **One fresh auditor per round.** Use the same brief every round. Never
  pass earlier findings or fixes to an auditor.
- **Auditors are read-only.** Only the orchestrator edits the document.
- **Fix every location.** Map each finding's fan-out across the whole
  document before editing, edit all locations in one pass, and re-read
  to confirm consistency.
- **Decision changes need confidence above 85, from confirmed facts.**
  At 85 or below, ask the user. Record every applied decision change in
  the appendix.
- **Honour exclusions.** Excluded topics are never reported by auditors
  and never fixed.
- **Failed spawns do not count.** Only an audit that produces a report
  counts toward the cap.
- **No git without approval.** Do not commit or push during or after
  the loop unless the user explicitly asks.
- **Report, don't just stop.** End with the per-round analysis and
  summary in Phase 4, Step 2.
