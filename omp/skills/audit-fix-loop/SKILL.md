---
name: audit-fix-loop
description: >
  Repeatedly audit a doc-definition-backed document with independent
  audit-document subagents. Treat their findings as leads, confirm each
  one against the sources yourself, fix every confirmed finding
  everywhere it occurs, and verify every claim a fix introduces before
  the next audit. Stop when an audit leaves no confirmed error or
  warning, when decision changes keep coming, or at the audit cap.
  Usage: /audit-fix-loop <path> [--max-audits=N] [--exclude=<topic>...]
---

# Audit Fix Loop

Bring a document to a state where an independent audit finds no errors
or warnings. Each round, a fresh subagent audits the document with the
`audit-document` skill. The orchestrating agent confirms every lead
against the sources itself, fixes each confirmed finding completely,
verifies the fix, and starts the next round. Subagents only produce
leads. The orchestrator alone is responsible for whether findings are
real and whether fixes are correct. This skill prevents four failures:

- **The incomplete fix**: a fact corrected in one place while its other
  statements stay stale, so that each later audit flags another copy.
- **The unverified fix**: a fix writes new claims (a symbol, a rule, a
  criterion, a config field) that nobody checks against the code, so a
  later audit flags an error the loop itself introduced.
- **The growing document**: every finding is answered by adding text,
  and auditors only ask for more, so the document grows round after
  round while its decisions stay the same.
- **The unsettled design**: audits keep changing decisions, one or two
  per round, because the design still has cases nobody worked out.
  More audits find those cases one at a time; they do not settle the
  design.

## Trigger

User says "audit and fix until clean", "audit-fix loop", "keep
auditing and fixing", "loop audits on this document", or invokes
`/audit-fix-loop`.

## Arguments

- **path** (required): the document to audit and fix. For an
  implementation plan, use `audit-plan-fix-loop` instead.
- `--max-audits=N` (optional, default 10): the most audits to run.
- `--exclude=<topic>` (optional, repeatable): finding topics the user
  has put out of scope, for example document length. Every auditor is
  told about them and must not report them. The orchestrator discards
  any finding on an excluded topic. A topic the user explicitly puts out
  of scope during the loop becomes an exclusion from the next round on.
  Declining a single proposed change does not.

## Phase 0: Setup

### Step 1: Read the governing material

Read the `audit-document` skill, the doc definition for the document's
type, and the full document. Note the document's companions, the code
it describes and the external sources it cites, such as requirement
exports. The fixes in Phase 3 are judged against these.

Record the body's word count: the document without its decision-change
appendix. This is the baseline for the growth checks in Phase 3, Step 8.

### Step 2: Write the auditor brief

Write one auditor brief. It contains:

- The document path and the instruction to read `skill://audit-document`
  and follow all its phases.
- The doc definition path. Name it explicitly when the document type is
  missing from the skill's type table.
- The codebase root, the companions, the cited external sources and the
  glossary.
- The yardstick: what the document is meant to achieve. Without it, an
  auditor flags deliberate omissions that the document justifies.
- The exclusions (`--exclude`, plus topics the user put out of scope
  during the loop), and a note that the decision-change appendix, when
  present, is requested and must not be flagged.
- The evidence requirement: every soundness and correctness finding
  cites the source path, line numbers and a short quote, plus the
  document's line numbers. Findings without evidence are discarded.
- Read-only: the auditor modifies nothing and runs no builds or tests.

Reuse the brief in every round. Its only allowed change is a new
exclusion. It never contains earlier findings, earlier fixes or the
orchestrator's opinions, which would bias the audit.

### Step 3: Start the fix ledger

Create a fix ledger outside the document, for example
`local://<document-stem>_fix-ledger.md`. Phase 2 and Phase 3 add one
entry per finding and per fix; Phase 4 reports from it. The ledger
never goes into the document.

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

### Step 1: Confirm or reject each lead

Nothing an auditor says is a fact until you confirm it yourself. Do not
infer, and do not trust the auditor's quote or line numbers. For each
finding, read the cited source yourself: the code at the given file and
lines, the requirement text, the doc definition or the glossary. Then
classify it:

- **Confirmed**: the source shows the document is wrong, contradictory,
  incomplete or non-compliant as described.
- **Partially confirmed**: the problem is real but the auditor's
  description or recommendation is wrong. Record the corrected version.
- **Rejected**: the source does not support the finding, the finding is
  on an excluded topic, or it asks to undo a decision whose stated
  trade-off holds. Record the reason.

Severity is your own judgement under the `audit-document` severity
table, not the auditor's.

Give each confirmed or partially confirmed finding one impact category,
by what its fix changes in the document. If several fit, use the first
in the list:

- **Decision**: a rule, behaviour, scope, threshold or config
  constraint the document decides.
- **Interface**: a type, field, signature, dependency or ownership,
  with no change in behaviour.
- **Description**: a wrong statement about existing code or a cited
  source.
- **Prediction**: an expected outcome the document cannot guarantee,
  such as a requirement status.
- **Risk or rationale**: a missing or wrong downside or argument for a
  decision that stands.
- **Criteria or inventory**: acceptance criteria, change lists, file
  lists.
- **Readability or compliance**: wording, structure, repetition, doc
  definition rules.

Record each finding's impact category in the ledger.

### Step 2: Decide whether to stop

- **Unsettled design**: this audit and the two before it each had at
  least one confirmed or partially confirmed decision-category finding
  of severity error or warning, from the auditor's report or your own
  Phase 1 checks. Diff-review findings do not count. Apply this round's
  other fixes (Phase 3) but hold its decision findings, then report
  (Phase 4) and ask the user how to proceed, with the decision count
  per audit and the mechanisms affected. If the user says to continue,
  count again from the next audit.
- **Clean**: otherwise, if no confirmed or partially confirmed finding
  of severity error or warning remains, stop (Phase 4). Info findings
  do not block, but fix the confirmed ones anyway.

Otherwise continue with Phase 3.

## Phase 3: Fix

### Step 1: Snapshot the document

Before the first edit of the round, copy the document outside the
repository, for example `local://<document-stem>_round-<N>.md`. Steps 6
and 8 diff against this snapshot.

### Step 2: Map the fan-out of each finding in the document

Before editing, search the whole document for every statement of the
affected fact, rule, name or value: synonyms, paraphrases, tables,
diagrams, type and config outlines, risks, acceptance criteria,
approach, open questions and appendices. List every location. A fix
that misses a location is incomplete.

### Step 3: Map the code consumers of each changed contract

A fix that changes a contract also changes what the document says
about everything that consumes it. A contract is an interface, a rule,
a config field, a data flow or an acceptance criterion. For each one
the fix changes, list its consumers in the code: callers, the test
harness and scene format that will run each criterion, loaders and
validators, serializers and converters, and other configs that hold the
same value. Every document statement about a consumer joins the
Step 2 list. Read each consumer in the source; do not infer it.

### Step 4: Decide and write each fix

A **decision change** is a fix in the decision impact category (Phase
2, Step 1). Before acting on one, state a confidence from 0 to 100 that
the change is correct, based only on facts you confirmed yourself in
the sources:

- **Above 85**: apply it, and record it in the appendix (Step 7).
- **85 or below**: stop and ask the user. Give the finding, the
  evidence, the proposed change, the alternatives and your confidence
  with its basis. Continue with the other fixes while waiting, but do
  not start the next audit until the answer is in.

Apply every other fix without asking. Write each fix as short as the
doc definition allows:

- **Delete before correcting.** If no decision, rationale or
  doc-definition requirement needs a wrong statement, delete it instead
  of correcting it.
- **Measure, don't re-predict.** Replace a wrong prediction with "to be
  measured", not with another prediction, unless a decision forces the
  outcome.
- **Replace before adding.** Otherwise rewrite the wrong statement in
  place. Add a sentence only when the confirmed finding shows something
  is missing, and add only that.
- **No Risk for an error.** Never add a Risk to cover a wrong
  statement; correct or delete it. Add a Risk only when a decision
  stands despite a downside, in one or two sentences.
- **Evidence stays out of the body.** Source paths, symbol names that
  serve only as proof, data identifiers and measurements go into the
  appendix entry (Step 7) or the ledger, unless the doc definition
  requires them in the body.
- **State each fact once.** Refer to the section that states a fact
  instead of restating it.
- **Work out the whole mechanism.** When a fix changes a mechanism that
  an earlier round already changed, list all of its cases once in the
  ledger (a table, a worked trace, or every pairing of its inputs),
  check each against the sources, and rewrite the mechanism from that
  list instead of adding another clause.

### Step 5: Apply all locations in one pass

Edit every location from Steps 2 and 3 together. Follow the doc
definition's rules: section structure, citations by section number and
name rather than line number, glossary terms. Then re-read each edited
section in full, and every section that refers to it, and confirm they
agree with each other.

### Step 6: Verify the fix against the sources

A fix is not done when the edit lands. It is done when every claim it
introduced or changed is confirmed. Diff the document against the Step
1 snapshot, and treat each added or changed claim as a new lead under
Phase 2, Step 1:

- each symbol, path, type and config field exists as stated, or is
  stated as new;
- each described behaviour matches the code that implements it;
- each interface or config rule matches its loaders, validators and
  other consumers (Step 3);
- each new or changed acceptance criterion can be run by the harness
  the document names, with the inputs the harness supports;
- each number agrees with the other numbers it constrains.

Read the source for each one yourself. A claim that fails goes back to
Step 4. Record in the ledger, per fix: the finding, each claim the fix
introduced, the source that confirms it, and the body word count added
or removed. A claim without evidence in the ledger is unverified, and
the next audit cannot start while one remains.

### Step 7: Record decision changes in an appendix

Keep a final appendix in the document, for example "Appendix A.
Decision changes during audit". Add one entry per decision change:

- the audit round;
- the affected decision, by section number and name;
- the change, before and after;
- the evidence: source paths and the facts confirmed;
- the confidence, and whether the user approved the change.

Create the appendix with the first decision change. Never remove
entries.

### Step 8: Check growth and review the diff

Count the body's words again. If the body grew by more than 5% in the
round, re-read every addition in the diff and cut each one that a
confirmed finding does not require. Then count again. If the body is
more than 10% larger than the baseline, ask the user whether to cut,
accept the size or stop, and do not start the next audit until the
answer is in. If the user accepts the size, the current count becomes
the baseline.

When the round changed a decision, an interface or an acceptance
criterion, or reaches the cap, spawn a fresh `reviewer` (or `task`) on
the diff alone, with the snapshot, the document and the codebase root.
Ask it to check only the changed text against the sources and the doc
definition. Confirm its findings under Phase 2, Step 1 and fix them
under Phase 3, Steps 2–7. Do not spawn another review for those fixes.
This review does not count toward `--max-audits` and cannot end the
loop.

## Phase 4: Report

### Step 1: Loop or stop

If Phase 2, Step 2 did not stop the loop and the audit count is below
`--max-audits`, return to Phase 1. Otherwise stop.

### Step 2: Analyze and summarize

Report to the user:

- One row per round: the audit number, the findings reported, the
  findings confirmed (with severity and impact category), the
  decision-category count, the findings rejected (with the reason), and
  the body word count after the round.
- The body word count at baseline and at the end, and which fixes
  account for most of the growth.
- The decision changes, with their confidence and approval status.
- Recurring clusters: findings that came back across rounds, and their
  root cause. Name the findings caused by an earlier round's fix.
- The stop reason: a clean audit, an unsettled design, or the cap. If
  the cap was reached, say whether the last round's fixes were reviewed
  (Phase 3, Step 8).
- Anything the user must still decide.

## Rules

- **Subagents produce leads, never facts.** Agent output is not
  evidence; confirm every claim in the source yourself.
- **One fresh auditor per round, one brief.** The brief changes only by
  new exclusions and never carries earlier findings or fixes.
- **Auditors are read-only.** Only the orchestrator edits the document.
- **Your fixes are leads too.** No audit starts while a claim a fix
  introduced lacks evidence in the ledger.
- **Delete, measure or replace before adding.** Concision is yours even
  when length is excluded.
- **Stop on an unsettled design.** Three audits in a row with a
  confirmed decision-category finding: stop and ask the user (Phase 2,
  Step 2).
- **Fix every location.** Each fix covers its fan-out in the document
  and in the code that consumes a changed contract, in one pass.
- **Decision changes at 85 or below need the user.** Record every
  applied decision change in the appendix.
- **No git without approval.** Do not commit or push unless the user
  explicitly asks.
