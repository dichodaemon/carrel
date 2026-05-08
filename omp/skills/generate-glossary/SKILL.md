---
name: generate-glossary
description: Generate a docs/glossary.md for the current project by exploring code and documentation, extracting project-specific vocabulary, then grilling the user to sharpen definitions and resolve ambiguities. Use when no glossary exists and the project has domain-specific terms that need precise definitions.
---

Generate a `docs/glossary.md` for this project. If `docs/glossary.md` already exists, report it and ask whether to update the existing glossary or stop.

## 1. Explore code

Read source files under the repo root. Identify:

- Package names, module names, directory names that carry domain meaning
- Types, interfaces, enums, and their field names
- Function names that encode domain concepts
- Configuration keys, CLI flags, environment variables
- Terms used inconsistently (the same concept called different things in different files)

For each candidate term, note where it appears and how it's used. Skip general programming concepts — if a competent developer unfamiliar with this project would know the term, it does not belong.

## 2. Explore documentation

Read everything under `docs/`. Extract:

- Defined terms in existing reference docs, specs, arch-designs, READMEs
- Implicit vocabulary — words used consistently with specific meaning
- Past naming decisions recorded in design studies or flagged as resolved
- Relationships between concepts documented in architecture docs

Look for contradictions: a term defined one way in a spec but used differently in a README, or a concept called by two names across documents.

## 3. Draft candidate terms

Compile a candidate glossary from the terms found in code and documentation. For each:

- Propose a canonical name (pick the best one when multiple names exist)
- Draft a one-sentence IS-definition
- List aliases to avoid
- Map relationships to other candidate terms

Group terms under subheadings if natural clusters emerge. Sort alphabetically within each group.

## 4. Grill the user

Present the candidate terms one cluster at a time. For each term, ask:

1. Is this the right canonical name? (propose one)
2. Is the definition accurate? (share the one-sentence draft)
3. What names should be avoided? (list alternatives found in code/docs)
4. How does it relate to other terms? (share proposed relationships)

Ask one question at a time. Wait for the user's answer before continuing.

During the grilling:

- **Challenge against existing language.** When the user uses a term that conflicts with what code or docs show, call it out. "The arch-design calls this a 'Slot,' but you just said 'target' — which is it?"
- **Sharpen fuzzy language.** When the user uses vague or overloaded terms, propose a canonical term. "You said 'config' — do you mean the Source file or the deployed Slot?"
- **Cross-reference with code.** When the user describes how something works, check whether the code agrees. Surface contradictions immediately.
- **Stress-test relationships.** Invent concrete scenarios that probe boundaries between related terms.

## 5. Write the glossary

Produce `docs/glossary.md` following the format defined in the glossary doc-definition. The file must contain all five required sections in order:

1. `# Glossary`
2. `## Terms` — canonical names with one-sentence definitions and _Avoid_ aliases, grouped under subheadings if clusters exist, alphabetically sorted
3. `## Relationships` — bold term names with cardinality where clear
4. `## Example Dialogue` — 2–4 exchanges drawn from the grilling session, testing a boundary the definitions leave ambiguous
5. `## Flagged Ambiguities` — any term conflicts surfaced during exploration or grilling, with the resolution and date

After writing, review the file against every rule in the glossary definition: one sentence max per term, project-specific only, no undefined terms in relationships, bold term names, resolution dates on flagged ambiguities.
