---
name: generate-glossary
description: >
  Generate a docs/glossary.md for a project directory by exploring its code
  and documentation, extracting project-specific vocabulary, then grilling the
  user to sharpen definitions and resolve ambiguities. Accepts a path argument
  (defaults to cwd). Usage: /generate-glossary [path]
---

# Generate Glossary

Generate a `docs/glossary.md` for a target directory by exploring its code and
documentation, extracting project-specific vocabulary, then grilling the user
to sharpen definitions, screen for anti-patterns, and resolve ambiguities.

## Trigger

User says "generate glossary", "create glossary", "build glossary", "add a
glossary", or invokes `/generate-glossary`.

## Arguments

- **path** (optional): Target directory. Defaults to the current working
  directory. Must contain a `docs/` subdirectory.

## Step 1: Read the glossary definition

Read `/workspace/folio/doc-definitions/glossary_definition.md` in full. Do not
proceed from memory — the definition is the authoritative format specification
and contains rules, anti-patterns, and examples that govern every subsequent
step.

## Step 2: Resolve target

If the user provided a path argument, use it. Otherwise, use the current
working directory.

Check that `<target>/docs/` exists. If it does not, stop and report: "No
`docs/` directory found at `<target>`. A glossary requires a `docs/` directory
— it lives at `docs/glossary.md` and its scope is the parent of `docs/` and
all descendants."

If `<target>/docs/glossary.md` already exists, report it and ask whether to
update the existing glossary or stop.

The scope for the glossary is `<target>/` — the parent of the `docs/` directory
— and all files and subdirectories under it.

## Step 3: Explore code

Read source files under `<target>/` (excluding `docs/`). Identify:

- Package names, module names, directory names that carry domain meaning
- Types, interfaces, enums, and their field names
- Function names that encode domain concepts
- Configuration keys, CLI flags, environment variables
- Terms used inconsistently (the same concept called different things in
  different files)

For each candidate term, note where it appears and how it's used. Skip general
programming concepts — if a competent developer unfamiliar with this project
would know the term, it does not belong.

## Step 4: Explore documentation

Read everything under `<target>/docs/`. Extract:

- Defined terms in existing reference docs, specs, arch-designs, READMEs
- Implicit vocabulary — words used consistently with specific meaning
- Past naming decisions recorded in design studies or flagged as resolved
- Relationships between concepts documented in architecture docs

Look for contradictions: a term defined one way in a spec but used differently
in a README, or a concept called by two names across documents.

## Step 5: Draft candidate terms

Compile a candidate glossary from the terms found in code and documentation.
For each:

- Propose a canonical name (pick the best one when multiple names exist)
- Draft a one-sentence IS-definition
- List aliases to avoid
- Map relationships to other candidate terms

Group terms under subheadings if natural clusters emerge. Sort alphabetically
within each group.

## Step 6: Grill the user

Present the candidate terms one cluster at a time. For each term, ask:

1. Is this the right canonical name? (propose one)
2. Is the definition accurate? (share the one-sentence draft)
3. What names should be avoided? (list alternatives found in code/docs)
4. How does it relate to other terms? (share proposed relationships)

Ask one question at a time. Wait for the user's answer before continuing.

During the grilling:

- **Challenge against existing language.** When the user uses a term that
  conflicts with what code or docs show, call it out. "The arch-design calls
  this a 'Slot,' but you just said 'target' — which is it?"
- **Sharpen fuzzy language.** When the user uses vague or overloaded terms,
  propose a canonical term. "You said 'config' — do you mean the Source file
  or the deployed Slot?"
- **Cross-reference with code.** When the user describes how something works,
  check whether the code agrees. Surface contradictions immediately.
- **Stress-test relationships.** Invent concrete scenarios that probe
  boundaries between related terms. "If a Slot has two Sources and both define
  the same key, which wins? Is that a Slot concern or a Source concern?"

### Anti-pattern screening

After the term-by-term grilling, run every candidate through the seven
anti-patterns from the glossary definition. For each anti-pattern, check
whether any candidate violates it:

| Anti-pattern | Check |
|---|---|
| Glossary as general dictionary | "Would a competent developer already know this term without this project?" If yes, remove it. |
| Multi-sentence definitions | Does any definition spill past one sentence? Cut or move to a reference doc. |
| Missing _Avoid_ aliases | Did exploration find alternative names that aren't listed? Add them. |
| Stale terms | Is the term still used in the current codebase? If removed, drop it. |
| Relationships without terms | Does any relationship reference a term not in the glossary? Define it or remove the relationship. |
| Dialogue restates definitions | Will the planned example dialogue test a boundary, or just repeat definitions? |
| Terms without boundaries | Does any term sit in complete isolation — no relationships, no ambiguity flags? Ask whether it's too general. |

Surface every hit to the user and resolve it before proceeding.

### Scope discipline probe

For each term, explicitly ask: is this specific to *this* project's scope, or
is it a concept from an external dependency, upstream service, or sibling
component? Terms outside the `docs/` parent scope do not belong — document them
in a reference doc instead.

### Boundary identification

Before moving to the writing step, ask the user: "Which pair of terms is most
often confused or has the blurriest boundary?" Build the example dialogue
around that boundary. The dialogue must clarify something the definitions
alone leave ambiguous — if every question has an obvious answer from the Terms
section, the dialogue is not pulling its weight.

## Step 7: Write the glossary

Produce `<target>/docs/glossary.md` following the format defined in the
glossary doc-definition. The file must contain all five required sections in
order:

1. `# Glossary`
2. `## Terms` — canonical names with one-sentence definitions and _Avoid_
   aliases, grouped under subheadings if clusters exist, alphabetically sorted
3. `## Relationships` — bold term names with cardinality where clear
4. `## Example Dialogue` — 2–4 exchanges drawn from the grilling session,
   testing the boundary identified in Step 6
5. `## Flagged Ambiguities` — any term conflicts surfaced during exploration
   or grilling, with the resolution and date

## Step 8: Pre-commit checklist

Before committing, verify every item. A "no" on any item means going back to
the relevant step.

- [ ] Every term is project-specific — no general programming concepts (Step 3)
- [ ] Every definition is exactly one sentence starting with what the thing IS
      (Step 5)
- [ ] _Avoid_ aliases are listed for every term that has known alternatives
      (Step 6)
- [ ] Terms are alphabetically sorted within each group (Step 5)
- [ ] Every term in the Relationships section is defined in Terms (Step 7)
- [ ] Relationships use **bold term names** and express cardinality where
      clear (Step 7)
- [ ] Example dialogue tests a boundary the definitions leave ambiguous, not
      restating definitions (Step 6)
- [ ] Flagged ambiguities include the resolution and a date (Step 7)
- [ ] No term references a concept outside the glossary's scope (Step 6)
- [ ] The file is named `glossary.md` and placed at `<target>/docs/glossary.md`
      (Step 2)
