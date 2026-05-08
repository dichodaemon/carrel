# Configuration Standards for AI Coding Assistants

Personal standards for writing and organizing configuration files (rules, skills, commands, context files, etc.) across AI coding assistant tools, with OMP as the primary authoring target. Currently followed personally; may be proposed as a team standard later.

---

## 1. Design Goals

### 1.1. Portability

Goal: capabilities should work in as many frameworks as possible, primarily OMP, Claude Code, and Cursor.

### 1.2. Maintainability (Write Once, Convert)

Goal: author everything in one canonical format (OMP native), and use tooling to convert or distribute to other frameworks as needed. No manual duplication across `.omp/`, `.claude/`, `.cursor/`.

### 1.3. Scope (Where Configuration Lives)

Goal: clear rules for what lives at the user-private, project-private, project, and folder levels, and why.

---

## 2. Standards

**Guiding principle: configure only what the model gets wrong.**

The agent already knows how to write code, structure tests, and produce documentation. Every rule, skill, and context file consumes context window tokens and competes for the model's attention. Over-configuring is worse than under-configuring: it adds noise, creates contradictions, and degrades the quality of guidance that actually matters.

Start with zero configuration. Observe where the agent consistently makes the wrong choice for your project. Add the minimum configuration to correct that specific failure. If you can't point to a repeated mistake the configuration would prevent, don't add it.

### 2.1. Capabilities Quick Reference

| Capability | What it is | Passive / Active | When the agent sees it |
|---|---|---|---|
| [**Rule**](../references/omp-configuration_reference.md#24-rules) | Markdown file with frontmatter defining a coding standard | Passive | On demand, when working on files matching its globs |
| [**Skill**](../references/omp-configuration_reference.md#25-skills) | Named directory with `SKILL.md` and supplementary files | Passive | On demand, via `skill://<name>` when context matches |
| [**Slash command**](../references/omp-configuration_reference.md#23-slash-commands) | Markdown file expanding into prompt text on `/name` | Passive | When user types `/name` |
| [**Context file**](../references/omp-configuration_reference.md#26-context-files) | `CLAUDE.md` or `AGENTS.md` merged into system prompt | Passive | Always (project root) or when working in matching directory |
| [**Custom tool**](../references/omp-configuration_reference.md#27-custom-tools) | TypeScript/JavaScript module the model can call | Active | During any turn, like built-in tools |
| [**Extension**](../references/omp-configuration_reference.md#21-extensions) | Runtime module with full lifecycle access | Active | Startup; reacts to events, registers tools/commands/shortcuts |
| [**Theme**](../references/omp-configuration_reference.md#22-themes) | JSON file controlling TUI appearance | N/A | Applied at startup and on live reload |

### 2.2. Portability Matrix

Which discovery paths are read by which tools:

| Capability | OMP | Claude Code | Cursor | Codex CLI |
|---|---|---|---|---|
| Rules | `.omp/rules/` | -- | `.cursor/rules/` | -- |
| Skills | `.omp/skills/`, `.claude/skills/` | `.claude/skills/` | -- | -- |
| Commands | `.omp/commands/`, `.claude/commands/`, `.codex/commands/` | `.claude/commands/` | -- | `.codex/commands/` |
| Context files | `.claude/CLAUDE.md`, `AGENTS.md` | `.claude/CLAUDE.md`, `AGENTS.md` | -- | `AGENTS.md` |
| Custom tools | `.omp/tools/`, `.claude/tools/`, `.codex/tools/` | `.claude/tools/` | -- | `.codex/tools/` |
| Hooks | `.omp/hooks/`, `.claude/hooks/` | `.claude/hooks/` | -- | -- |
| Extensions | `.omp/extensions/` | -- | -- | -- |
| Themes | `~/.omp/agent/themes/` | -- | -- | -- |
| SYSTEM.md | `.omp/SYSTEM.md` | -- | -- | -- |

See the [Capabilities and Discovery Paths](../references/omp-configuration_reference.md#2-capabilities-and-discovery-paths) reference for complete discovery path details.

**Key observations:**

- `.claude/` is the widest shared surface between OMP and Claude Code. If a capability has a claude provider, putting it in `.claude/` gets you both tools for free.
- Rules have no `.claude/` provider. OMP reads `.omp/rules/` and `.cursor/rules/`. Claude Code reads neither. Cursor reads `.cursor/rules/`. There is no single path that gives all three tools the same rules.
- `AGENTS.md` has the broadest reach (OMP + Claude Code + Codex CLI). `CLAUDE.md` is OMP + Claude Code.
- Extensions, themes, and SYSTEM.md are OMP-only. No cross-tool path exists.
- Cursor has almost no overlap with OMP beyond `.cursor/rules/`.

### S1. Canonical Format: OMP Native [Maintainability]

**All OMP-native capabilities are authored in `.omp/`.**

Applies to: rules, skills, commands, custom tools, hooks, extensions, themes, prompts, instructions, SYSTEM.md.

See [Capabilities and Discovery Paths](../references/omp-configuration_reference.md#2-capabilities-and-discovery-paths) for each capability's native directory.

See [Decision Flow](../references/omp-configuration_reference.md#4-decision-flow-what-to-create) for choosing the right capability type.

### S2. Cross-Tool Distribution via `.claude/` [Portability, Maintainability]

**For capabilities that have a claude provider, maintain a copy (or symlink) in `.claude/` for Claude Code compatibility.**

Applies to: skills, commands, custom tools, hooks.

Does NOT apply to: rules (no claude provider), extensions (OMP-only), themes (OMP-only), context files (authored in `.claude/` per S1 exception).

**Open question:** Symlinks vs copies vs build step? See Open Questions, item 1.

### S3. Rules Require Special Handling [Portability]

**Rules are the hardest capability to share across tools.**

- OMP reads `.omp/rules/` and `.cursor/rules/`.
- Claude Code does not read rules from any path (it has no rules provider).
- Cursor reads `.cursor/rules/`.

**Decision:** Author in `.omp/rules/` (canonical source). For Cursor users, convert to `.cursor/rules/`. For Claude Code users, embed critical rules as `@`-includes in `CLAUDE.md` -- this is a workaround (no glob matching, no on-demand loading) but the only available mechanism.

See [Rules](../references/omp-configuration_reference.md#24-rules) for the full rules configuration reference.

See [Time-Traveling Stream Rules (TTSR)](../references/omp-configuration_reference.md#241-time-traveling-stream-rules-ttsr) for TTSR-specific guidance.

### S4. Context Files Are the Universal Layer [Portability]

**`AGENTS.md` is the only file format read by OMP, Claude Code, AND Codex CLI.**

Use `AGENTS.md` for project-level and directory-level instructions that must be visible to all tools. Use `CLAUDE.md` when you only need OMP + Claude Code.

**Boundary between context files and rules:** A context file must be both **general** (not specific to a file type or pattern) AND **short** (< 10 lines of guidance per topic). If the guidance is longer, or if it applies to specific file types, it belongs in a rule. Context files are loaded on every turn -- every line competes for the model's attention. Keep them as a concise index of project facts, not a dump of all conventions.

Examples:
- "We use Bazel. Workspaces are split by top-level directory." --> Context file (general, short).
- "In `cc_library` targets, always specify `deps` explicitly, never use glob." --> Rule with `globs: ['**/BUILD.bazel']` (specific to file type, actionable directive).

### S5. Scope Hierarchy [Scope]

Configuration lives at four levels, each with different ownership and lifecycle:

| Level | Location | Committed | Shared | Use for |
|---|---|---|---|---|
| **User-private** | `~/.omp/agent/` | No | No | Personal tool preferences: theme, model roles, symbol preset, user-level extensions. |
| **Project-private** | `<cwd>/.omp/` (gitignored entries) | No | No | Per-project personal overrides: local extensions, experimental rules, debug tools. |
| **Project** | `<cwd>/.omp/`, `<cwd>/.claude/` | Yes | Yes | Shared team standards: rules, skills, commands, context files, custom tools. |
| **Folder** | `<cwd>/<dir>/AGENTS.md`, `<cwd>/<dir>/CLAUDE.md` | Yes | Yes | Directory-scoped overrides: workspace-specific instructions, subsystem conventions. |

**Relationship between `.omp/` and `.claude/` at Project scope:**

- `.omp/` is the **canonical source** for all OMP-native capabilities (rules, skills, commands, tools, etc.) per S1.
- `.claude/` holds two kinds of content:
  - **Context files** (`CLAUDE.md`) -- authored here directly (S1 exception).
  - **Derived copies** of skills, commands, and tools -- generated or symlinked from `.omp/` per S2.
- Do not author OMP-native capabilities directly in `.claude/`. The `.omp/` version is the source of truth.

**Principles:**

- **User-private is yours.** Never commit `~/.omp/agent/config.yml` or user-level extensions to a repo. These are your preferences, not the team's.
- **Project-private is local.** Per-project personal config (e.g., a debug extension, a local rule) goes in `.omp/` but must be in `.gitignore`. Do not pollute the shared config.
- **Project is shared.** Everything committed under `.omp/` or `.claude/` should work for any team member using any supported tool.
- **Folder overrides project.** Use `AGENTS.md` in subdirectories for workspace-specific instructions (e.g., `backend/AGENTS.md` for backend-specific instructions). Deeper files override higher ones.

### S6. Naming Conventions [Maintainability]

**Rules use one file per language-concern pair: `<domain>/<concern>.md`.**

Examples: `cpp/testing.md`, `cpp/ownership.md`, `cpp/errors.md`, `python/typing.md`.

See [Rules](../references/omp-configuration_reference.md#24-rules) for frontmatter schema and glob patterns.

**Skills use `<workflow-name>/SKILL.md`.**

Examples: `scientific-debugging/SKILL.md`, `release-process/SKILL.md`.

**Commands use `<verb>-<noun>.md` or `<verb>.md`.**

Examples: `review-pr.md`, `spec.md`, `plan.md`.

---

## 3. Use Cases

### UC1. Building a New Repo from Scratch

Starting a greenfield project. The goal is to configure OMP (and cross-tool equivalents) so that every aspect of the project -- structure, coding, documentation -- has the right guidance in the right place.

Remember the guiding principle: start with zero configuration and add only what you observe the agent getting wrong.

#### UC1.1. Elements and Recommended Capabilities

| Element | Sub-element | Best capability | Scope level | Rationale |
|---|---|---|---|---|
| **Directory structure** | Layout conventions | Context file (`CLAUDE.md`) | Project | Always needed; the agent must know the project layout before any task. |
| | Naming conventions | Rule | Project | Domain-specific (e.g., `src/` vs `lib/`); loaded on demand when creating files. |
| **Coding** | Coding standards | Rule (per language-concern) | Project | Glob-matched to file types. One rule per concern (naming, errors, ownership, etc.). |
| | Build files | Context file or Rule | Project | If every task needs build context (monorepo layout, workspace structure), use context file. If only needed when editing BUILD files, use a rule with `globs: ['**/BUILD.bazel']`. |
| | Unit testing | Rule | Project | Glob-matched to test files (`*test*.cc`, `*test*.py`). Covers test structure, naming, assertions. |
| | Validation processes | Skill | Project | Multi-step workflows (CI pipeline, pre-submit checks, coverage analysis). Too long for a rule; agent reads on demand. |
| | Release process support | Skill | Project | Step-by-step release procedures, checklists, environment setup. Read when the user asks to prepare a release. |
| **Documentation** | (see breakdown below) | Mixed | Project / Folder | Depends on scope, type, and lifespan. |

#### UC1.2. Documentation Breakdown

Documentation guidance splits across three axes: scope (what is being documented), type (what kind of document), and lifespan (how long it lives). Each combination maps to a different capability.

**By scope -- where does the guidance live?**

| Doc scope | Best capability | Scope level | Rationale |
|---|---|---|---|
| New subproject | Context file (`AGENTS.md`) | Folder | Placed in the subproject root. Defines structure, conventions, and dependencies for that subproject. |
| New component | Rule | Project | Glob-matched to component directories. Covers file layout, interface conventions, test expectations. |
| New capability/improvement | Skill | Project | Workflow guidance: how to spec, plan, implement, and verify a feature. Agent reads when starting feature work. |
| Bug fix | Skill | Project | Debugging workflow: how to investigate, reproduce, fix, and verify. Separate from feature work. |

**By type -- what capability enforces the format?**

| Doc type | Best capability | Rationale |
|---|---|---|
| Specification | Skill + Command | Skill (`spec-writing/`) is reference material defining the format, sections, and expectations (see [Spec Reference](../doc-definitions/spec-doc_definition.md)). Command (`/spec`) scaffolds a new file from a template embedded in the command body. |
| Implementation plan | Skill + Command | Skill (`feature-workflow/`) is reference material defining the plan format (see [Implementation Plan Reference](../doc-definitions/impl-plan_definition.md)). Command (`/plan`) scaffolds a new file from a template. |
| Cookbook | Reference doc | Append-only collection of standalone recipes per domain. Format defined in [Cookbook Reference](../doc-definitions/cookbook_definition.md). No skill or command needed -- entries are added incrementally, not scaffolded. |
| README | Rule | Glob-matched to `**/README.md`. Enforces structure, required sections, style. |
| TODO | Rule | Glob-matched to TODO files or inline TODO comments. Enforces format and expiration policy. |

**By lifespan -- how does maintenance affect the choice?**

| Lifespan | Best capability | Rationale |
|---|---|---|
| Long-lasting (maintained) | Rule, Context file, or Cookbook | Rules enforce structure on every edit. Context files keep high-level guidance always visible. Cookbooks accumulate findings over time -- append-only, never restructured. All three are maintained alongside the code they describe. |
| Single-shot (transient) | Skill + Command | The skill defines the format; the command scaffolds it. Once written, the document is not re-enforced by rules -- it lives as a static artifact. |

**Common combinations:**

| Scenario | Scope | Type | Lifespan | Recommendation |
|---|---|---|---|---|
| Project README | Project | README | Long-lasting | Rule (`readme-format.md`, glob: `README.md` at root) |
| Subproject README | Subproject | README | Long-lasting | Same rule (glob: `**/README.md`) + `AGENTS.md` for subproject conventions |
| Feature spec | Capability | Specification | Single-shot | Skill (`spec-writing/`) + Command (`/spec`) to scaffold |
| Feature plan | Capability | Implementation plan | Single-shot | Skill (`feature-workflow/`) + Command (`/plan`) to scaffold |
| Bug investigation | Bug fix | Implementation plan | Single-shot | Skill (`bug-fix-workflow/`) to guide the process |
| Domain cookbook | Project or Tool | Cookbook | Long-lasting | One `<domain>_cookbook.md` per domain. No skill or command -- entries appended as findings emerge. |
| Component docs | Component | README | Long-lasting | Rule (glob-matched to component dirs) for structure |
| Release notes | Project | Specification | Long-lasting | Skill (`release-process/`) for format + Rule for enforcing changelog structure |

#### UC1.3. Recommended Configuration Layout

```text
<repo>/
  .omp/
    rules/
      directory/
        naming.md                    # Glob: *, loaded when creating files
      cpp/
        testing.md                   # Glob: **/*test*.cc
        ownership.md                 # Glob: **/*.cc, **/*.h
        errors.md                    # Glob: **/*.cc, **/*.h
      python/
        typing.md                    # Glob: **/*.py
      build/
        bazel.md                     # Glob: **/BUILD.bazel
      docs/
        readme-format.md             # Glob: **/README.md
        todo-format.md               # Glob: **/TODO.md
    skills/
      validation-process/
        SKILL.md
      release-process/
        SKILL.md
      feature-workflow/
        SKILL.md
      bug-fix-workflow/
        SKILL.md
      spec-writing/
        SKILL.md
    commands/
      spec.md                        # Scaffolds a new spec document
      plan.md                        # Scaffolds a new implementation plan
  .claude/
    CLAUDE.md                        # Project-wide context: build system, repo layout, dev environment
    skills/                          # Derived from .omp/skills/ per S2 (symlinks or copies)
    commands/                        # Derived from .omp/commands/ per S2
  docs/
    doc-definitions/                  # Defines formats for document types
      spec-doc_definition.md
      impl-plan_definition.md
      cookbook_definition.md
      design-study_definition.md
    standards/                        # Conventions and standards for practices
      document-structure_standards.md
      omp-configuration_standards.md
    references/                       # Records facts, configurations, practical knowledge
      omp-configuration_reference.md
      dev-environment_setup.md
    cookbooks/                        # Domain-scoped append-only recipe collections
      docker_cookbook.md
      wezterm_cookbook.md
    workflow/                          # Specs, plans, and design studies
      specs/
      plans/
      design-studies/
    archive/                           # Retired documents (preserving relative path)
  <subproject>/
    AGENTS.md                        # Subproject-specific conventions (cross-tool)
```

Note: start with `CLAUDE.md` and a few rules for your most common mistakes. Add skills and commands as workflows stabilize. Populate `docs/` directories as documents are created -- do not pre-populate every slot on day one. See [Document Structure Standards](document-structure_standards.md) for the full directory structure and file naming conventions.

#### UC1.4. Mapping to Standards

| Standard | How it applies here |
|---|---|
| S1 | All rules, skills, and commands authored in `.omp/`. Context files authored in `.claude/`. |
| S2 | Skills and commands symlinked or copied to `.claude/` for Claude Code users. |
| S3 | Rules stay in `.omp/rules/`. Critical rules also referenced via `@`-includes in `CLAUDE.md` for Claude Code. Cursor gets `.cursor/rules/` via conversion when tooling exists. |
| S4 | `AGENTS.md` used for subproject-level context (broadest tool reach). `CLAUDE.md` for project root (OMP + Claude Code). |
| S5 | Rules and skills at Project scope. `AGENTS.md` at Folder scope. Personal experiments in Project-private (gitignored). |
| S6 | Rules: `<domain>/<concern>.md`. Skills: `<workflow>/SKILL.md`. Commands: `<verb>.md`. |

---

## 4. Decisions Log

Resolved questions and their rationale, for future reference.

| # | Decision | Rationale | Date |
|---|---|---|---|
| D1 | Naming granularity: one rule per language-concern pair (`cpp/testing.md`), not one per language | Matches existing repo tree; keeps files focused, cheaper to read, better glob targeting. | 2026-04-07 |
| D2 | Conversion tooling: deferred until cross-tool pain is real | No team using Cursor or Codex on this repo yet. Building tooling now would be speculative. Strawman design captured in Open Questions for when needed. | 2026-04-07 |
| D3 | `alwaysApply` rules: allowed for critical invariants only | e.g., "never use sudo", "never commit secrets". Context token cost is justified only for unconditional safety constraints. | 2026-04-07 |
| D4 | TTSR rules: guidance added under S3 | Use only when violation is cheap to detect (regex) and expensive to fix (rewrite). Regular glob-matched rules are cheaper for most standards. | 2026-04-07 |
| D5 | Context file vs rule boundary: must be general AND short | < 10 lines per topic, not specific to a file type. Otherwise it's a rule. | 2026-04-07 |
| D6 | Skill + Command interaction: command scaffolds, skill is reference | Command embeds a template and creates a file. Skill is separate reference material the agent reads for format guidance. | 2026-04-07 |
| D7 | Scope naming: User-private / Project-private / Project / Folder | "Private" signals uncommitted. Prefix distinguishes breadth. | 2026-04-07 |
| D8 | Tool scope: OMP, Claude Code, Cursor, Codex CLI only | Windsurf and Cline excluded from standards (supported by OMP but not in active use). | 2026-04-07 |

---

## 5. Open Questions

1. **Symlinks vs copies vs build step** for cross-tool distribution (S2).
2. **Rules strategy for Claude Code** -- context file `@`-includes is the current workaround. Is there a better mechanism?
3. **Conversion tooling design** (deferred per D2) -- strawman: a script reads `.omp/rules/*.md` and generates `.cursor/rules/*.mdc`; for skills/commands/tools, creates symlinks from `.omp/` to `.claude/`. Runs as pre-commit hook or on-demand.
4. **Cursor rules format** -- how different is `.mdc` from `.md` in practice? Does OMP's cursor provider handle the differences transparently?
