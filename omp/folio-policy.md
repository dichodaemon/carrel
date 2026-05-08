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

A `docs/` directory may contain a `glossary.md` — a living lexicon of project-specific vocabulary. It defines terms, concepts, and names that have precise meaning within the scope of the `docs/` directory's parent and all its descendants. Format is defined in `/workspace/folio/doc-definitions/glossary_definition.md`.

- **Use glossary terms when possible.** When writing documentation, commit messages, code comments, or agent output about anything within the glossary's scope, prefer the canonical terms defined in it. Do not use avoided aliases.
- **Check the glossary before coining a term.** If you're about to introduce a new term within a scope that has a glossary, check whether it already defines one for that concept. If it does, use it.
- **Check whether a glossary exists for the scope you're working in.** When starting work in a directory, look for a `docs/glossary.md` at that level or above. If one exists, read it before generating documentation.
---

