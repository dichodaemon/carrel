# Folio Policy

## Universal Dev Framework

`/workspace/folio` is the absolute path to the **folio** repository on this machine (`dichodaemon/folio` on GitHub). Folio manages documentation framework and workflow artifacts.

This workspace uses a documentation framework with strict format definitions and authoring standards. The authoritative source is:

```
/workspace/folio/
  doc-definitions/    Format and structure requirements for each document type
  standards/          Conventions governing documentation practices
  references/         Factual reference material
```

### Rules

- **Before authoring any document** (spec, plan, design study, cookbook, README, or reference), read the corresponding doc-definition. Do not guess at format -- the definitions are normative.
- **Before making structural decisions** about documentation (naming, placement, directory layout), read `standards/document-structure_standards.md`.
- **File naming follows a strict pattern**: `[YYYY-MM-DD_]<topic>_<doctype>.md`. The separator rule is: underscores between structural elements, hyphens within elements. The doctype suffix inventory is closed -- do not invent new suffixes without checking the standard.
- **Document placement is scope-driven**: component docs stay local, cross-cutting docs go in `docs/`, universal framework docs live in the folio repo root. See section 3 of the document structure standards.
- **Cookbooks are append-only**: add numbered entries at the end. Never reorder, renumber, or restructure existing entries.


### Glossary

A project may have a `docs/glossary.md` — a living lexicon of project-specific vocabulary. It defines the terms, concepts, and names that have precise meaning within the codebase and cannot be looked up in a general reference. Format is defined in `/workspace/folio/doc-definitions/glossary_definition.md`.

- **Use glossary terms when possible.** When writing documentation, commit messages, code comments, or agent output about the project, prefer the canonical terms defined in the glossary. Do not use avoided aliases.
- **Check the glossary before coining a term.** If you're about to introduce a new term, check whether the glossary already has one for that concept. If it does, use it.
- **When a glossary exists and you're generating documentation**, read it first. The glossary is the vocabulary contract for the project — documentation that ignores it creates confusion.
---

