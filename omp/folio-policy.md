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

---

