---
name: audit-document
description: >
  Audit any doc-definition-backed document for compliance, correctness,
  comprehensiveness, and readability. Reads the doc definition for the
  document's type and verifies structure, claims, and diagrams against
  source code, BUILD files, and visual vocabulary standards. Produces a
  structured findings report with recommendations.
  Usage: /audit-document <path-to-document>
---

# Audit Document

Audit a document against its doc definition, the codebase, and the
visual vocabulary standards. The auditor reads everything from scratch
and produces a findings report grouped into four categories: compliance,
correctness, comprehensiveness, and readability.

The skill uses progressive disclosure -- each phase reads only the
inputs it needs. A leaf-package README with no diagrams never triggers
reads of the visual vocabulary standards or palette.yaml. A design
study with no code references never reads BUILD files or headers.

## Trigger

User says "audit", "audit this document", "check for compliance",
"review for correctness", or invokes `/audit-document`.

## Arguments

- **path** (required): Path to the document to audit.

## Phase 0: Bootstrap

Read exactly two things. Do not read anything else yet.

### Step 1: Determine document type

Infer the type from the filename or directory context:

| Pattern | Type |
|---|---|
| `README.md` | readme |
| `*_arch-design.md` | arch-design |
| `*_arch-assessment.md` | arch-assessment |
| `*_design-study.md` | design-study |
| `*_spec.md` | spec-doc |
| `*_brief.md` | brief |
| `*_plan.md` | impl-plan |
| `glossary.md` | glossary |
| Inside a `cookbooks/` directory | cookbook |

If the type cannot be determined, ask the user.

For `impl-plan` documents, stop and direct the user to `/audit-plan`
instead -- it has plan-specific checks (companion alignment, codebase
grounding, status gating) that this skill does not replicate.

### Step 2: Read the doc definition

Read `/workspace/folio/doc-definitions/<type>_definition.md`. This is
the normative reference for what the document must contain. Do not
proceed from memory.

### Step 3: Read the document under audit

Read the full document.

## Phase 1: Compliance

Verify structural requirements from the doc definition. Each failure
is categorized by severity.

### Step 1: Section structure

Check that every required section is present and correctly ordered per
the doc definition. For each section, verify:

- Correct heading level and name.
- Required subsections present.
- Table column headers match the definition (e.g., `Document | Purpose`,
  not `Document | Description`).
- Code fence languages are correct (e.g., `starlark` for BUILD deps,
  not `python`).

### Step 2: Metadata and naming

If a naming or placement finding is suspected (wrong filename pattern,
wrong directory, missing metadata), read
`/workspace/folio/standards/document-structure_standards.md` now.
Verify:

- Filename follows the `[YYYY-MM-DD_]<topic>_<doctype>.md` pattern.
- Frontmatter fields are complete (if the doc definition requires them).
- Document placement matches scope rules.

Only read the document-structure standards if this step needs them.
Skip the read if the filename and placement are obviously correct.

### Step 3: Table of contents

Check whether a TOC is required (varies by doc type -- for READMEs,
required when more than 5 sections). If required, verify it exists
and links to all sections. Check for non-standard formatting (e.g.,
numbered prefixes when the definition shows plain names).

### Step 4: Diagrams

If the document contains any mermaid diagram, or if the doc definition
requires one (e.g., README definition requires a dependency diagram
when 4+ internal deps exist):

1. Read `/workspace/folio/standards/visual-vocabulary_standards.md`
   now.
2. Find and read `palette.yaml` from the nearest ancestor `docs/`
   directory. If no palette exists, note it as info-level (colors
   cannot be verified).
3. For each diagram, verify:
   - Uses `flowchart TD`, not `graph TD`.
   - Fill colors match `palette.yaml` entries for each subsystem
     (`node_fill` and `stroke` from the owning subsystem's entry).
   - Container uses `container_fill` (lighter tint), not `node_fill`.
   - Solid borders for internal nodes, dashed for external references.
   - External references use their own subsystem's palette colors,
     not the scoped subsystem's colors.
   - A `> [!NOTE]` legend is present immediately after the diagram.
   - Legend explains scope, border styles, and arrow semantics.
   - Legend does not reference internal tooling (palette.yaml, arcane,
     generation scripts).
   - At L2: container is the owning subsystem, not a higher-level
     grouping.
   - At L1: container is the package.
   - At L3: no container, all nodes solid.
   - Cycle edges highlighted in red (if cycles exist).

Only read the visual vocabulary standards and palette if this step
triggers. Skip entirely if the document has no diagrams and the doc
definition does not require one.

### Step 5: Glossary check

Look for `docs/glossary.md` at the document's level or above. If
found, read it. Check the document for:

- Use of avoided aliases when a canonical term exists (info-level).
- Introduction of new terms that already have glossary definitions
  (info-level).

Only read the glossary if one exists. Skip if not found.

## Phase 2: Correctness

Verify claims against source code. The specific checks depend on the
document type. Read source files per-finding -- do not bulk-read all
headers upfront.

### For READMEs

1. Read the BUILD file in the package directory. Verify:
   - Every target in the Build Targets table exists in BUILD.
   - Every target in BUILD appears in the Build Targets table
     (completeness is Phase 3, but ghost targets are correctness).
   - Dependencies Internal table matches BUILD `deps` (at the
     package level).
   - Dependencies External table matches BUILD external deps.

2. For each API signature shown in the document, read the
   corresponding header file and verify:
   - Function signatures match (parameter names, types, const
     qualifiers, return types).
   - Struct/class fields match (no missing fields that are
     functionally important, no listed fields that don't exist).
   - Type locations match (file is where the document says it is).

3. For each file listed in the Contents table, verify it exists
   in the directory. Flag ghost files.

4. Verify test coverage claims against actual test files:
   - Read test files only when the document claims specific test
     case IDs or counts that need verification.
   - Count `TEST` / `TEST_F` macros and compare.

5. Check `docker exec` commands for required flags (e.g.,
   `-u "$(id -u)"` per project conventions).

6. Verify code reference links resolve to existing files.

### For arch-designs

1. For each type definition or contract, read the corresponding
   header and verify consistency.
2. For each diagram, verify that node names match real symbols in
   the codebase.
3. Check that acceptance criteria reference real test targets.

### For references and cookbooks

1. Verify that code examples reference real files and functions.
2. Check that step-by-step instructions reference existing paths.
3. Verify include paths and build targets are valid.

### For all types

- Verify internal document links (cross-references to other docs)
  resolve to existing files.
- Verify relative links to source files resolve.

## Phase 3: Comprehensiveness

Check for gaps -- things that should be documented but aren't.

### For READMEs

1. Read the directory listing. Compare against the Contents table:
   - Files in the directory not listed in Contents.
   - Sub-packages (directories with BUILD files) not mentioned.

2. Reuse the BUILD file from Phase 2. Compare against Build Targets:
   - BUILD targets not in the table.

3. Check for undocumented public API:
   - Read headers listed in Contents. For each exported function or
     type, check whether the API section mentions it (by name or
     by reference to the owning package).

4. Check the Testing section:
   - Are all test targets mentioned in the run command?
   - Does the coverage summary account for all test cases? (Read
     test files only if the summary looks incomplete.)

5. Check for missing source file links:
   - Are code references (file names, type names, function names)
     linked to their source files?
   - Are dependency paths linked to their package READMEs?

### For arch-designs

1. Check that every component mentioned in prose has a
   corresponding diagram node.
2. Check that every acceptance criterion has a verification method.

### For all types

- Check that the Reference Documents section (if present) links to
  actual existing documents.
- Check for sections the doc definition marks as required that are
  empty or placeholder.

## Phase 4: Readability

Assess presentation quality. These are advisory findings (info or
warning level). No additional reads should be needed -- everything
is already in context from Phases 1-3.

Check for:

- **Lead paragraph density**: Is it a single run-on sentence listing
  many concrete type names? Should it be 2-3 focused sentences?
- **API section bloat**: Does the API section duplicate type
  definitions from other packages? This creates staleness risk --
  recommend referencing with links instead.
- **Diagram granularity**: Too many nodes or edges for the diagram's
  orientation purpose? Recommend simplifying or splitting.
- **Empty-information sections**: External dependencies table listing
  only STL, or other universally-implied dependencies.
- **Structural confusion**: Test targets listed as packages,
  orientation text disguised as a package README, or other
  organizational mismatches.
- **Prose vs. code mismatch**: Design section describing an old
  version of the architecture that doesn't match the code verified
  in Phase 2.

## Phase 5: Report

### Step 1: Compile findings

Each finding has:

- **ID**: Category prefix + number (e.g., C-1, P-3, H-2, R-1).
  - `C` = Correctness
  - `P` = Compliance
  - `H` = Comprehensiveness
  - `R` = Readability
- **Location**: Section name and/or line number.
- **Description**: What is wrong.
- **Recommendation**: Specific action to fix it.

### Step 2: Present report

Format:

```
# Audit: <document path>

**Scope:** Compliance with <doc-definition>, correctness against
codebase, comprehensiveness, readability.

**Files examined:** <list of source files read during the audit>

---

## Correctness (N findings)

### C-1: <title>

**Line N.** <description>

**Recommendation:** <specific fix>

---

## Compliance (N findings)

### P-1: <title>

...

---

## Comprehensiveness (N findings)

### H-1: <title>

...

---

## Readability (N findings)

### R-1: <title>

...

---

## Diagrams

<diagram-specific findings if any, or "No diagrams present">

---

## Summary

| Category | Count |
|---|---|
| **Correctness** | N |
| **Compliance** | N |
| **Comprehensiveness** | N |
| **Readability** | N |
| **Total** | **N** |

<one paragraph identifying the root cause cluster if findings
share a common cause>
```

### Step 3: Root cause analysis

After compiling all findings, look for clusters -- multiple findings
that share a single root cause. State the root cause and which
findings it would resolve. This helps the author prioritize: fixing
one structural issue may eliminate several findings at once.

## Rules

- **Read-only.** Never modify the document. Report findings with
  recommendations.
- **Progressive disclosure.** Read each input only when a phase
  needs it. Do not front-load reads of standards, palette, BUILD
  files, or headers.
- **Per-finding verification.** When verifying an API signature,
  read only the specific header file -- not all headers in the
  package.
- **Doc-definition-driven.** The doc definition is the normative
  reference. If the definition doesn't require something, don't
  flag its absence.
- **Source-grounded.** Every correctness finding must cite the
  source file and the actual content that contradicts the document.
  Do not flag correctness issues based on memory or assumption.
- **No false positives from elision.** If the document uses
  `// ...` to elide fields, do not flag the elided fields as
  "missing" -- flag them only if the elision hides functionally
  important information without acknowledgment.
- **Diagram audit is mandatory when diagrams exist.** If the
  document contains any mermaid block, the visual vocabulary
  standards and palette must be checked. This is the most
  error-prone area.
- **Do not fabricate the "orientation page" concept.** Every
  directory with a BUILD file is a package. Its README follows
  the standard README structure. There is no exemption for
  directories that contain sub-packages.
