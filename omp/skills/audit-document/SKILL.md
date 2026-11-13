---
name: audit-document
description: >
  Audit a doc-definition-backed document for soundness, correctness,
  readability, comprehensiveness and compliance, content first. Produces
  a findings report with directive recommendations.
  Usage: /audit-document <path-to-document> [brief]
---

# Audit Document

Audit a document for content quality first and format second. The
auditor reads everything from scratch. It judges whether the technical
content is sound and grounded in the codebase, whether the solution is
as simple as the requirements allow, and whether the document is clear
and succinct. Only then does it check comprehensiveness and compliance
with the doc definition and the visual vocabulary standards. Findings
fall into five categories: soundness, correctness, readability,
comprehensiveness and compliance.

## Trigger

User says "audit this document", "review this document", "check for
compliance", "review for correctness", or invokes `/audit-document`.

## Arguments

- **path** (required): path to the document to audit.
- **brief** (optional): the doc definition path, the companions, the
  yardstick, exclusions and sections the author requested. The brief's
  definition path and companions override Phase 0, Steps 1–3. Topics
  the brief excludes are never reported. Sections the brief marks as
  requested are not flagged for existing.

## Phase 0: Bootstrap

### Step 1: Determine document type

Unless the brief names it, infer the type from the filename or
directory context:

| Pattern | Type |
|---|---|
| `README.md` | readme |
| `*_arch-design.md` | arch-design |
| `*_arch-assessment.md` | arch-assessment |
| `*_design-note.md` | design-note |
| `*_design-study.md` | design-study |
| `*_report.md` | report |
| `*_spec.md` | spec-doc (deprecated) |
| `*_brief.md` | brief |
| `*_plan.md` | impl-plan |
| `glossary.md` | glossary |
| Inside a `cookbooks/` directory | cookbook |

If the type cannot be determined, ask the user. For `impl-plan`
documents, stop and direct the user to `/audit-plan`: it has
plan-specific checks (companion alignment, codebase grounding, status
gating) that this skill does not replicate.

### Step 2: Read the doc definition

Read `/workspace/folio/doc-definitions/<type>_definition.md`, or the
path the brief gives. It defines what the document must contain and
what each section is for. If no definition file exists, stop and ask
for its path. Do not proceed from memory.

### Step 3: Read the document under audit

Read the full document, plus the companion documents its metadata or
the brief names (spec, design study, arch-design, brief). Companions
define the requirements and decisions the document must honour.

### Step 4: Build the inventory

Before judging anything, list:

- **Objectives and requirements.** What the document says the design
  must achieve, including requirements inherited from companions, the
  brief's yardstick or cited external sources. This is the yardstick
  for every soundness judgement.
- **Mechanisms.** Every component, type, state, config option,
  interface change, process, phase or rule the document proposes or
  explains.
- **Decisions.** Every design decision, with its stated rationale and
  alternatives.
- **Factual claims and predictions.** Every statement about existing
  code (symbols, signatures, file locations, line numbers, behaviour,
  BUILD targets) and every predicted outcome (requirement status,
  performance).
- **Dependencies.** For each factual claim and prediction, the
  decision, interface or acceptance criterion that relies on it, or
  "none".

The inventory drives Phases 1 and 2. Keep it in working memory; it
does not appear in the report.

## Phase 1: Soundness

Judge whether the technical content is right and as simple as the
requirements allow. Depth by document type:

- **design-note, arch-design, design-study, spec-doc:** every step.
- **brief, report, arch-assessment, readme, cookbook:** Steps 1, 2 and
  5. Judge whether the explanations are technically right and whether
  the usage shown is the simplest correct one.
- **glossary:** Step 2, Explanations only. Definitions must match the
  concepts as the code implements them.

### Step 1: Map the design onto the code

For each mechanism in the inventory, read the code it touches: the
call sites, the producers of its inputs, the consumers of its outputs,
and its lifecycle (initialization, per-cycle execution, teardown,
threading, ownership). Use `lsp references` on every existing symbol
the design changes or depends on.

### Step 2: Grounding

For each mechanism, verify:

- **Data availability.** The inputs the design consumes exist at the
  point where it consumes them, with the frequency, frame, units and
  validity it assumes.
- **Lifecycle and timing.** Initialization order, per-cycle ordering,
  latency and state persistence match what the design assumes.
- **Ownership and interfaces.** The design changes the components that
  own the behaviour, not a neighbour that happens to see the data.
  Every caller of a changed interface is accounted for.
- **Explanations.** Descriptions of existing behaviour are technically
  right, not just plausible.

### Step 3: Parsimony

For each unit of complexity, ask three questions. Units of complexity
include new types, fields, state, config options, interface changes,
phases, processes, indirection layers and special cases.

1. Which objective or requirement needs it?
2. What is the simplest design that meets that requirement? Look first
   at what the codebase already does: if an existing mechanism covers
   the need, or would with a small extension, the design must use it or
   state why it cannot.
3. Does the document state why it chose more than the simplest design?

There is a finding when a simpler design meets the requirements and the
document states no reason, when the stated reason does not hold (the
claimed cost of the simpler design is not real, or the claimed benefit
is not needed by any requirement), or when the mechanism serves no
stated objective or requirement. A stated reason that holds is no
finding: do not override a justified decision with a preference.

Every parsimony finding names the simpler alternative concretely, cites
the code it would build on, and says what it gives up. If you cannot
name a concrete alternative, there is no finding.

### Step 4: Efficiency

Check cost where it matters:

- Work per cycle in hot paths: algorithmic complexity, allocations,
  copies, redundant recomputation.
- CPU and GPU budget: new kernels, new sample buckets, larger buffers,
  extra memory traffic.
- Communication: new topics, larger messages, serialization in hot
  paths.

A design that adds hot-path cost without acknowledging it, or whose
cost threatens a stated budget or deadline, is a finding.

### Step 5: Coherence

- **Contradictions.** No two sections, diagrams, types or contracts
  disagree.
- **Invariants.** Every stated invariant is enforced by a mechanism
  the document describes.
- **State machines.** Every state has an entry, an exit and a defined
  response to every input that can occur in it, including faults and
  mode changes.
- **Edge cases.** Boundaries, empty inputs, missing or stale data,
  first cycle, and restart are handled or explicitly excluded.

When a mechanism has one gap, enumerate all its cases (each state ×
input, each caller, each consumer of a changed config) and report every
open case in one finding. Report every finding you find; do not sample.

### Step 6: Requirement fit and alternatives

- Every objective and requirement maps to a mechanism, or the document
  explicitly scopes it out.
- No mechanism goes beyond the stated scope.
- Every design decision carries the rationale and alternatives its doc
  definition requires (design note: Rationale, plus Alternatives with
  the deciding factor). Do not ask for a Risk or trade-off the
  definition makes optional.
- **Design studies.** The options are compared fairly on the same
  criteria. The simplest viable option is among them. The
  recommendation follows from the evaluation. Rejected options record
  why they were rejected.

## Phase 2: Correctness

Verify every factual claim in the inventory against the source, one
claim at a time. Every finding names what depends on the statement
(Phase 0, Step 4). If nothing does, recommend deleting the statement.
A prediction the document cannot guarantee is a finding only if a
decision relies on it; recommend "to be measured". Never predict
requirement outcomes yourself.

### Step 1: README checks

1. Read the BUILD file in the package directory. Verify:
   - If a Build Targets table is present, every target in it exists in
     BUILD.
   - The Dependencies Internal table matches BUILD `deps` at the
     package level.
   - The Dependencies External table matches BUILD external deps.
2. For each API signature shown in the document, read the header and
   verify:
   - Function signatures match: parameter names, types, const
     qualifiers, return types.
   - Struct and class fields match: no missing fields that are
     functionally important, no listed fields that don't exist.
   - Type locations match: the file is where the document says it is.
3. For each file listed in the Contents table, verify it exists in the
   directory. Flag ghost files.
4. Verify test coverage claims against the test files. Read test files
   only when the document claims specific test case IDs or counts.
   Count `TEST` / `TEST_F` macros and compare.
5. Check `docker exec` commands for required flags, such as
   `-u "$(id -u)"` per project conventions.

### Step 2: Arch-design checks

1. For each type definition or contract, read the corresponding header
   and verify consistency with the current code, or that the document
   marks it as new or modified.
2. For each diagram, verify that node names match real symbols.
3. Check that acceptance criteria reference real test targets or name
   the targets to be created.
4. Verify every file and line anchor.

### Step 3: Cookbook checks

1. Verify that code examples reference real files and functions.
2. Check that step-by-step instructions reference existing paths.
3. Verify include paths and build targets.

### Step 4: All types

- Internal document links resolve to existing files.
- Relative links to source files resolve.

## Phase 3: Readability

Read the whole document again, in order, as its target reader would:
an engineer competent in the field but new to this area. When
succinctness and completeness pull apart, cut repetition and filler,
never information a decision, interface or criterion needs.

### Step 1: Clarity

Look for `docs/glossary.md` at or above the document's level and read
it. Then check:

- The lead and the opening of each section tell the reader what the
  section establishes before the detail.
- Every term is defined before it is used, or is a glossary term. One
  concept has one name throughout.
- The reasoning is explicit. The reader learns why, not only what.
- Sentences say one thing each. Long sentences that chain several
  facts get split.
- Diagrams, tables and prose each carry what they are best at. A
  diagram does not repeat a table; prose does not narrate a diagram.

### Step 2: Succinctness

- **Repetition.** The same fact, rule or decision stated in several
  sections. Keep it in the section that owns it and reference it
  elsewhere.
- **Filler.** Throat-clearing, restated headings, summaries of what
  was just said, hedging.
- **Verbosity.** Passages that can say the same in fewer words.
- **Duplicated external content.** Type definitions or APIs copied
  from other packages. They go stale; reference them with links.
- **Empty-information content.** Content any reader already assumes,
  such as dependency tables listing only the standard library.

For every succinctness finding, locate each redundant passage and
state what to cut or merge.

### Step 3: Reasoning gaps

- A conclusion that does not follow from what the document states.
- Missing context a reader needs to evaluate a decision.
- Behaviour the document relies on but never explains.

A missing-content finding names the decision or doc-definition section
that needs the content. Otherwise there is no finding.

### Step 4: Structure

- The section order follows the reader's path, from what and why to
  how.
- Multi-paragraph content is not squeezed into table cells. Tables
  that truncate content or flatten hierarchy go back to prose with
  headings.
- No structural confusion: test targets listed as packages,
  orientation text disguised as a package README (every directory with
  a BUILD file is a package, with no orientation-page exemption), or
  similar organizational mismatches.
- Diagram granularity suits the diagram's purpose. Too many nodes or
  edges for orientation means simplify or split.

## Phase 4: Comprehensiveness

Check for required content the doc definition or a decision needs and
the document lacks. A finding names the doc-definition section or the
decision that needs the content; otherwise there is no finding.

### Step 1: README checks

1. Read the directory listing and compare it with the Contents table:
   - Files in the directory not listed in Contents.
   - Sub-packages (directories with BUILD files) not mentioned.
2. Reuse the BUILD file from Phase 2. If the definition requires a
   Build Targets table, list BUILD targets missing from it.
3. Check for undocumented public API. Read the headers listed in
   Contents. For each exported function or type, check whether the
   API section mentions it, by name or by reference to the owning
   package.
4. Check the Testing section:
   - All test targets appear in the run command.
   - The coverage summary accounts for all test cases. Read test files
     only if the summary looks incomplete.
5. Check for missing source links:
   - Code references (file, type, function names) link to their
     source files.
   - Dependency paths link to their package READMEs.

### Step 2: Arch-design checks

1. Every component mentioned in prose has a corresponding diagram
   node.
2. Every acceptance criterion has a verification method.
3. Every external dependency the design relies on is named, with its
   status.

### Step 3: All types

Sections the doc definition marks as required are not empty or
placeholders.

## Phase 5: Compliance

Verify the structure and presentation rules from the doc definition
and the standards. Standards exist so that readers can rely on a
shared vocabulary.

### Step 1: Section structure

Check that every required section is present and in the order the doc
definition gives. For each section, verify:

- Heading level and name.
- Required subsections.
- Table column headers match the definition (for example
  `Document | Purpose`, not `Document | Description`).
- Code fence languages are correct (for example `starlark` for BUILD
  deps, not `python`).

### Step 2: Metadata and naming

Read `/workspace/folio/standards/document-structure_standards.md`
unless the filename matches the type table pattern and the directory
matches the type's directory in the standard's Quick Reference. Verify:

- The filename follows the `[YYYY-MM-DD_]<topic>_<doctype>.md`
  pattern (READMEs and `glossary.md` are exempt).
- Frontmatter fields are complete, if the doc definition requires
  them.
- Placement matches the scope rules.

### Step 3: Table of contents

Check whether a TOC is required (varies by doc type; for READMEs,
required when there are more than 5 sections). If required, verify it
exists and links to all sections. Check for non-standard formatting,
such as numbered prefixes when the definition shows plain names.

### Step 4: Diagrams

Run this step if the document contains any mermaid diagram, or if the
doc definition requires one (for example, the README definition
requires a dependency diagram when there are 4 or more internal deps).

1. Read `/workspace/folio/standards/visual-vocabulary_standards.md`.
2. Find and read `palette.yaml` in the nearest ancestor `docs/`
   directory. If none exists, record an info-level finding (colours
   cannot be verified).
3. For each diagram, verify:

   **Type and content**
   - The diagram type matches the doc definition's directive. If the
     definition requires a detail-level data flow diagram for runtime
     data flows, a plain flowchart with unlabeled arrows and no
     stereotypes does not satisfy it. A diagram of the wrong type is
     missing required structure.
   - Detail-level data flow diagrams contain both entity kinds: at
     least one component (plain label, the entity with runtime
     identity that orchestrates work) and stereotyped operations it
     invokes. A diagram with only operations and no owning component
     violates visual vocabulary standards §3.2.1 (Entity Model) and
     §3.2.4 (Node Kinds). Operations connect to their owning component
     by an arrow or sit in the same container.
   - Operations carry the stereotype prefixes (`«kernel»`,
     `«function»`, `«method»`) that the doc definition or the visual
     vocabulary standards require for the diagram type.
   - Arrows are labeled with data types or buffer names. All data flow
     diagrams require labeled arrows.
   - Arrow routing follows caller-to-callee ownership (§3.2.5, Arrow
     Semantics). Every incoming arrow to an operation originates at the
     entity that calls it, not at a sibling operation that merely
     produces data it consumes. If a component orchestrates two
     operations, the data routes through the component
     (`A → Component → B`), never directly between siblings (`A → B`).
     Read the implementation files (`.cc`, `.cu`) to trace call sites;
     this check cannot be done from the diagram alone.

   **Visual vocabulary**
   - Uses `flowchart TD`, not `graph TD`, as the diagram skills
     require.
   - Fill colours match `palette.yaml` (`node_fill` and `stroke` from
     the owning subsystem's entry).
   - Containers use `container_fill` (the lighter tint), not
     `node_fill`.
   - Solid borders for internal nodes, dashed for external references.
   - External references use their own subsystem's palette colours,
     not the scoped subsystem's.
   - At L2 the container is the owning subsystem, not a higher-level
     grouping. At L1 the container is the package. At L3 there is no
     container and all nodes are solid.
   - Cycle edges, if highlighted, use one distinct colour (§3.1.3,
     optional).

   **Legend**
   - A `> [!NOTE]` legend follows the diagram immediately.
   - The legend declares fill colours, border styles, arrow weights and
     any stereotypes used (§2.5).
   - The legend does not mention internal tooling (palette.yaml,
     arcane, generation scripts).

4. When a finding requires replacing a diagram, the recommendation
   directs the implementer to follow `/create-dataflow-diagram`, not
   to draw a replacement from intuition.

### Step 5: Glossary

Using the glossary read in Phase 3, Step 1, check the document for:

- Avoided aliases used where a canonical term exists.
- New terms introduced for concepts the glossary already defines.

## Phase 6: Report

### Step 1: Compile findings

Each finding has:

- **ID**: category prefix plus number: `S` soundness, `C` correctness,
  `R` readability, `H` comprehensiveness, `P` compliance.
- **Severity**, by impact, whatever the category:

  | Severity | Meaning | Examples |
  |---|---|---|
  | `error` | Fixing it changes a decision, an interface or an acceptance criterion, or the mechanism cannot work as written. | A mechanism that cannot work against the code; a simpler design that meets the requirements with no stated reason; scope creep; an incomplete state machine; a contradiction between decisions; cost that threatens a stated budget. |
  | `warning` | A wrong or missing statement that a decision, interface or criterion relies on as a premise; the fix changes text but not the decision. Also every violation of required structure or the visual vocabulary. | A wrong description of code a rationale rests on; a missing required alternative; unacknowledged hot-path cost; a missing required section; a wrong diagram type, palette, border style, stereotype or arrow routing. |
  | `info` | A concrete edit to a statement nothing depends on. | A wrong statement no decision needs (recommend deleting it); repetition; a wording fix; an avoided glossary alias. |

- **Location**: section name and the document's line numbers.
- **Description**: what is wrong. Every finding that states a fact
  about code, a requirement or the doc definition cites the path, line
  numbers and a verbatim quote. It names what in the document depends
  on the statement.
- **Recommendation**: the fix, stated as a directive ("Replace X with
  Y", "Delete Z and reuse W"). Never hedge ("Consider X", "You might
  want to Y"). Parsimony findings name the simpler alternative and
  what it gives up.

### Step 2: Root cause analysis

Look for clusters: several findings that share one cause. State the
cause and the findings it would resolve. Fixing one structural or
design issue often eliminates several findings at once.

### Step 3: Present report

Format:

```
# Audit: <document path>

**Scope:** Soundness and correctness against the codebase,
readability, comprehensiveness, compliance with <doc-definition>.

**Files examined:** <source files read during the audit>

## Verdict

<Two to four sentences. Is the design sound and grounded? Is it as
simple as the requirements allow? Is the document clear and succinct?
Name the most important findings by ID.>

---

## Soundness (N findings)

### S-1: <title> [error]

**Lines N–M.** <description, evidence and what depends on it>

**Recommendation:** <directive fix>

---

<Repeat the block for Correctness (C), Readability (R),
Comprehensiveness (H) and Compliance (P).>

## Summary

| Category | Errors | Warnings | Info | Total |
|---|---|---|---|---|
| **Soundness** | N | N | N | N |
| **Correctness** | N | N | N | N |
| **Readability** | N | N | N | N |
| **Comprehensiveness** | N | N | N | N |
| **Compliance** | N | N | N | N |
| **Total** | **N** | **N** | **N** | **N** |

<root cause analysis from Step 2>
```

## Rules

- **Read-only.** Never modify the document. Report findings with
  recommendations.
- **Content before format.** Run the phases in order. Soundness and
  readability get full attention; a long compliance checklist never
  crowds them out.
- **Parsimony.** Phase 1, Step 3 governs complexity findings: judged
  against the document's own requirements, with a concrete alternative,
  never against taste.
- **Source-grounded, never inferred.** Every finding that states a fact
  cites path, line numbers and a verbatim quote. Ordering, timing,
  threading and call relations are shown by the call chain read in the
  source, never inferred from declaration order, file position, names
  or comments. A claim you did not verify is not a finding.
- **Honour the brief.** Never report an excluded topic or flag a
  requested section.
- **Progressive disclosure.** Read each input when a phase needs it.
  Read the code a mechanism touches when assessing that mechanism; do
  not front-load standards, palette, BUILD files or headers.
- **Doc-definition-driven structure.** The doc definition is the
  normative reference for structure. If it does not require a section,
  do not flag the section's absence.
- **No false positives from elision.** If the document uses `// ...`
  to elide fields, do not flag the elided fields as missing unless the
  elision hides functionally important information without
  acknowledgment.
- **Decisive recommendations.** Every recommendation is a directive.
  If a finding does not warrant action, it does not warrant a finding.
