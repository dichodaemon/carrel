# OMP Configuration and Extensibility Reference

Reference for Oh My Pi configuration, capabilities, discovery paths, and runtime behavior.

---

## 1. Main Configuration Files

| File | Scope | Purpose |
|---|---|---|
| `~/.omp/agent/config.yml` | User (global) | Primary settings: model roles, theme, symbol preset, color-blind mode, extensions, disabled extensions, `hideThinkingBlock`, etc. |
| `<cwd>/.omp/settings.json` | Project | Project-level settings overrides (extensions, disabled extensions). Merged with global. |

Settings are layered: `schema defaults <- global config.yml <- project settings.json <- runtime overrides`. Edit via `/settings` command in OMP or by hand. Project settings are read-only from OMP's perspective (discovered, not written).

---

## 2. Capabilities and Discovery Paths

Each subsection follows this template: path table, description, when to use, schema/frontmatter, behavior, constraints, authoring guidance.

### 2.1. Extensions

| Path | Scope | Tools |
|---|---|---|
| `~/.omp/agent/extensions/` | User | OMP |
| `<cwd>/.omp/extensions/` | Project | OMP |

Extensions are TypeScript/JavaScript modules exporting a default factory function. They have full access to the agent lifecycle: tools, commands, shortcuts, event interception, UI, and runtime configuration.

**When to use:** You need runtime behavior -- policy enforcement, tool interception, custom tools, keyboard shortcuts, UI components, or anything that reacts to events.

**Entry resolution for directories:**

1. `package.json` with `omp.extensions` (or legacy `pi.extensions`) -- use declared entries
2. `index.ts`
3. `index.js`
4. Otherwise scan one level: `*.ts`, `*.js`, subdir `index.ts`/`index.js`, subdir `package.json`

No recursive discovery beyond one subdirectory level. TypeScript is preferred over JavaScript when both exist.

**Additional sources:**

- Explicit paths in `config.yml` under `extensions:` array
- CLI flags `--extension`/`-e` or `--hook`

**Precedence:** Project `<cwd>/.omp/extensions/` loads before user `~/.omp/agent/extensions/`. Explicit CLI paths and `config.yml` entries are appended after auto-discovered ones. Deduplicated by absolute path; first seen wins.

**Disabling:**

```yaml
disabledExtensions:
  - extension-module:foo    # name derived from filename or directory
```

**Constraints:**

- Discovery is at startup only (restart required for new extensions).
- Extensions run in the same process (no sandbox).
- Calling runtime methods like `pi.sendMessage()` during factory execution throws `ExtensionRuntimeNotInitializedError`.
- Unhandled exceptions in `tool_call` handlers block tool execution (fail-closed).
- Reserved shortcuts are silently ignored: `ctrl+c`, `ctrl+d`, `ctrl+z`, `ctrl+k`, `ctrl+p`, `ctrl+l`, `ctrl+o`, `ctrl+t`, `ctrl+g`, `shift+tab`, `shift+ctrl+p`, `alt+enter`, `escape`, `enter`.

**Writing effective extensions:**

- **One module per concern.** A safety policy extension and a custom tool extension should be separate files.
- **Register during load, act during events.** Registration methods only during factory; runtime behavior from event handlers, commands, and tools.
- **Guard UI access.** Check `ctx.hasUI` before calling `ctx.ui.*` -- extensions run in headless/background/RPC modes too.

### 2.2. Themes

| Path | Scope | Tools |
|---|---|---|
| `~/.omp/agent/themes/<name>.json` | User | OMP |

Built-in themes are embedded in the binary: `dark`, `light`, `titanium`, `light-paper`, plus others in `defaults/`. Custom themes are JSON files placed in the themes directory.

**When to use:** You want to customize the TUI appearance (colors, symbols, syntax highlighting).

**Required JSON structure:**

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Theme display name |
| `colors` | Yes | All 67 color tokens (see theme docs for full list) |
| `vars` | No | Reusable color variables, referenced by name in `colors` |
| `symbols` | No | `preset` (`unicode`/`nerd`/`ascii`) and per-key `overrides` |
| `export` | No | HTML export colors (`pageBg`, `cardBg`, `infoBg`) |

**Color value formats:** hex (`"#RRGGBB"`), 256-color index (`0..255`), var reference (resolved from `vars`), or `""` for terminal default.

**Loading and validation:**

1. Read and parse JSON
2. Validate against `ThemeJsonSchema` -- all 67 `colors` tokens must be present
3. Resolve `vars` references recursively (circular references throw)
4. Convert to ANSI by terminal color capability (truecolor or 256-color)

**Precedence:** Built-in themes checked first, then custom directory. Built-in names win on collision. Auto dark/light selection uses `COLORFGBG` environment variable -- background index `< 8` selects `theme.dark`, `>= 8` selects `theme.light`.

**Constraints:**

- All 67 color tokens are required for custom themes -- no partial overrides.
- Live reload watches only the active custom theme file. Deleting it falls back to `dark`.

**Writing effective themes:**

- **Start from a built-in.** Copy a built-in theme's color values as a starting point rather than writing 67 tokens from scratch.
- **Use `vars` for consistency.** Define your palette colors once in `vars` and reference them in `colors` to avoid hex duplication.
- **Test critical surfaces.** Check markdown rendering, tool blocks (pending/success/error), diff rendering, status line readability, thinking-level borders, and bash/python mode borders.
- **Test both color modes.** If your terminal supports truecolor, test with `COLORTERM=truecolor`; hex values downgrade to 256-color on limited terminals.

### 2.3. Slash Commands

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/commands/*.md` | native | User | OMP |
| `<cwd>/.omp/commands/*.md` | native | Project | OMP |
| `~/.claude/commands/*.md` | claude | User | OMP, Claude Code |
| `<cwd>/.claude/commands/*.md` | claude | Project | OMP, Claude Code |
| `~/.codex/commands/*.md` | codex | User | OMP, Codex CLI |
| `<cwd>/.codex/commands/*.md` | codex | Project | OMP, Codex CLI |

Slash commands are markdown files that expand into prompt text when the user types `/name` in the editor. They support argument substitution and frontmatter metadata.

**When to use:** You have a canned prompt or instruction you want to reuse -- "review this file for X", "summarize changes since last commit", etc.

**Frontmatter fields:**

| Field | Effect |
|---|---|
| `description` | Shown in autocomplete. If absent, first non-empty body line is used (trimmed, max 60 chars). |
| `name` | Override command name (default: filename without `.md`). Codex provider only. |

**Argument substitution in body:**

- `$1`, `$2`, ... -- positional arguments (quote-aware splitting: `'single'` and `"double"` preserve spaces)
- `$ARGUMENTS` or `$@` -- all arguments as a single string
- Template rendering via `renderPromptTemplate` with `{ args, ARGUMENTS, arguments }` context

**Command resolution order at prompt time:**

1. Extension-registered commands (`pi.registerCommand`) -- execute immediately, even during streaming
2. TypeScript custom commands -- can return replacement text or void
3. File-based slash commands -- markdown expansion as described above
4. Prompt templates -- applied after all slash processing
5. If nothing matches, the literal `/...` text is sent to the agent as-is

**Precedence:** native (100) > claude (80) > claude-plugins (70) > codex (70). First command with a given name wins. For native, project beats user on collision. For claude/codex, user beats project.

**Constraints:**

- Commands are discovered at init and after `/move`. No file watcher -- new command files require restart or `/move`.
- Name collisions with built-ins (`/settings`, `/model`, `/mcp`, `/move`, `/exit`, etc.) are consumed before file commands and cannot be overridden.

**Writing effective commands:**

- **Include `description` frontmatter.** Without it, autocomplete shows a truncated first line which is often unhelpful.
- **Use argument substitution for reusable prompts.** `/review $1` with a file path argument is more flexible than hardcoding paths.
- **Keep the body focused on instructions.** The expanded text becomes the user prompt -- write it as you would type it.

### 2.4. Rules

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/rules/*.{md,mdc}` | native | User | OMP |
| `<cwd>/.omp/rules/*.{md,mdc}` | native | Project | OMP |
| `~/.cursor/rules/*.{mdc,md}` | cursor | User | OMP, Cursor |
| `<cwd>/.cursor/rules/*.{mdc,md}` | cursor | Project | OMP, Cursor |

Rules are markdown files with frontmatter that define coding standards, conventions, and constraints. The agent reads them via `rule://<name>` when working in a matching domain.

**When to use:** You have coding standards, conventions, or constraints that should apply when the agent works on specific file types or in specific domains.

**Frontmatter fields:**

| Field | Effect |
|---|---|
| `description` | Required for the rule to appear in the rulebook (listed in system prompt by name + description). |
| `globs` | File patterns indicating when the rule applies (e.g., `**/*.cc,**/*.h`). Surfaced to the agent as advisory context but not enforced by code. |
| `alwaysApply` | If `true`, full rule content is injected directly into the system prompt on every turn. Not listed in the rulebook. |
| `ttsr_trigger` | If present, rule becomes a TTSR (time-travel stream interruption) rule -- triggers mid-stream, not surfaced in the rulebook. |

**Bucket routing (mutually exclusive, priority order):**

1. **TTSR** -- has `ttsr_trigger`. Not addressable via `rule://`.
2. **Always-apply** -- `alwaysApply: true`. Full content auto-injected into system prompt. Also addressable via `rule://`.
3. **Rulebook** -- has `description`, not always-apply, not TTSR. Listed by name + description in system prompt. Agent reads on demand via `rule://<name>`.

**Precedence:** native (100) > cursor (50) > windsurf (50) > cline (40). First rule with a given name wins.

**Writing effective rules:**

- **Always include `paths` frontmatter.** Without it, the rule has no glob signal and won't appear in the rulebook. The agent has no way to know when to read it.
- **One concern per rule file.** Testing, ownership, error handling -- not "C++ best practices". Smaller files are cheaper to read and easier to maintain.
- **Bullet points, not prose.** Each line should be a standalone directive the agent can act on immediately.
- **Concrete over abstract.** "Use `absl::StatusOr<T>`" not "handle errors appropriately".
- **Good/bad examples when ambiguity exists.** Style and formatting rules need examples to be unambiguous (e.g., active vs passive voice).
- **Keep it short.** The agent reads rules on every matching file -- every token costs context window space.
- **Delete rules that moved.** If a rule became a skill or was superseded, remove the old file. Two representations of the same concept is a maintenance fork.

#### 2.4.1. Time-Traveling Stream Rules (TTSR)

TTSR is a stream-interception mechanism that catches rule violations while the agent is generating output, before the response is complete.

**How it works:**

1. A rule with `ttsr_trigger` frontmatter registers a regex pattern against the agent's output stream.
2. Every stream chunk (text and tool-call deltas) is checked against all registered TTSR patterns.
3. When a pattern matches:
   - The in-progress generation is **immediately aborted**.
   - Depending on `contextMode`, the partial output is either discarded (`"discard"`) or kept (`"keep"`).
   - A synthetic user message is injected containing the rule's content inside `<system-interrupt reason="rule_violation" rule="<name>">` XML.
   - The agent **retries generation** with the violation flagged in context.

The agent effectively gets a second chance without the user seeing the violating output.

**Example rule:**

```markdown
---
ttsr_trigger: "std::endl"
---
Do not use `std::endl`. Use `"\n"` instead. `std::endl` flushes the stream buffer
and causes unnecessary performance overhead.
```

If the agent starts writing `std::endl` in generated code, the stream is interrupted and the agent retries knowing it violated that rule.

**Repeat policies:**

- `repeatMode: "once"` -- rule fires at most once per session.
- `repeatMode: "after-gap"` -- rule can re-fire after N completed turns (`repeatGap`), measured at `turn_end`.

**Caveats:**

- TTSR rules are **not** in the rulebook and **not** addressable via `rule://`. They exist purely as stream interceptors.
- The regex runs on every stream chunk, so expensive patterns add latency.
- There is a 50ms window between abort and retry where state can change.
- Injection state is in-memory only -- not persisted across session restarts.
- A rule with both `ttsr_trigger` and `alwaysApply` goes to TTSR only (TTSR takes priority).

### 2.5. Skills

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/skills/<name>/SKILL.md` | native | User | OMP |
| `<cwd>/.omp/skills/<name>/SKILL.md` | native | Project | OMP |
| `~/.claude/skills/<name>/SKILL.md` | claude | User | OMP, Claude Code |
| `<cwd>/.claude/skills/<name>/SKILL.md` | claude | Project | OMP, Claude Code |

Skills are passive knowledge packs -- named directories containing a `SKILL.md` and optional supplementary files. The agent reads them on demand via `skill://<name>` or `skill://<name>/path/to/file`.

**When to use:** You have knowledge or a workflow the agent should follow -- coding standards, debugging playbooks, architecture guides, step-by-step procedures. The agent decides when to read the skill based on context matching (name + description) or when you explicitly invoke it.

**`SKILL.md` frontmatter fields:**

| Field | Effect |
|---|---|
| `name` | Override skill name (default: directory name). |
| `description` | Required for native provider discovery. Shown in system prompt skill list. |
| `globs` | File patterns for context matching (advisory). |
| `alwaysApply` | If `true`, skill content is always available. |

**`skill://` URL resolution:**

- `skill://<name>` -- resolves to that skill's `SKILL.md` (frontmatter stripped)
- `skill://<name>/<relative-path>` -- resolves inside the skill directory
- Rejects absolute paths and `..` traversal; resolved path must stay within `baseDir`

**Precedence:** native (100) > claude (80) > claude-plugins/agents/codex (70). First skill with a given name wins. Deduplicated by `realpath` (symlink-safe).

**Constraints:**

- Non-recursive scan of `skills/` -- only `skills/<name>/SKILL.md` is found, not `skills/group/<name>/SKILL.md`. For nested taxonomies, point `skills.customDirectories` at the nested parent.
- Optional `/skill:<name>` commands require `skills.enableSkillCommands: true` in settings.

**Writing effective skills:**

- **Always include `name` and `description` frontmatter.** Native provider requires `description` for discovery. Without it, the skill is invisible.
- **Keep `SKILL.md` as the entry point.** Put detailed references, examples, and templates in separate files under the skill directory and reference them as `skill://<name>/references/file.md`.
- **Write for the agent, not the user.** Skills are read by the model -- use direct instructions, not conversational prose.
- **Avoid duplicate skill names across sources.** First match wins by provider precedence; a shadowed skill is silently ignored.

### 2.6. Context Files

Context files are persistent instruction files merged into the system prompt. They define project-wide or directory-scoped constraints, conventions, and architectural context.

**When to use:** You have project-wide or directory-scoped constraints that should always be in the agent's context -- build system instructions, repo structure, dev environment, coding conventions.

#### Project-level context files

| Path | Provider | Priority | Scope | Tools |
|---|---|---|---|---|
| `<cwd>/.omp/AGENTS.md` | builtin | 100 | Project | OMP |
| `<ancestor>/.omp/AGENTS.md` | builtin | 100 | Project (walk-up) | OMP |
| `<cwd>/.claude/CLAUDE.md` | claude | 80 | Project | OMP, Claude Code |
| `<ancestor>/.claude/CLAUDE.md` | claude | 80 | Project (walk-up) | OMP, Claude Code |
| `<ancestor>/AGENTS.md` | agents-md | 10 | Project (walk-up) | OMP, Claude Code, Codex CLI |

#### Discovery mechanics

Three independent providers discover context files. All run in parallel; results are merged and deduplicated by content.

**Builtin/OMP provider (priority 100):** Walks up from `cwd` through ancestor directories to the git repo root (or `$HOME` if not in a git repo). At each ancestor, checks for a non-empty `.omp/` directory. Stops at the **first** `.omp/` directory found. Reads `.omp/AGENTS.md` from that directory if it exists. A closer `.omp/` directory shadows a farther one -- if `<cwd>/sub/.omp/` exists, `<cwd>/.omp/AGENTS.md` is not discovered by this provider.

**Claude provider (priority 80):** Same walk-up pattern for `.claude/` directories. Reads `CLAUDE.md` (and other Claude Code config) from each `.claude/` found on the walk up.

**Standalone AGENTS.md provider (priority 10):** Walks up from `cwd` to repo root (or `$HOME`). At each ancestor, checks for a bare `AGENTS.md` file. Skips files whose immediate parent is a dotdir (`.omp/AGENTS.md` is not found by this provider -- that's the builtin provider's job). All discovered files are included (not just the nearest).

**Walk-up boundaries:**

- `repoRoot` = directory containing `.git`, or `null` if not in a git repo.
- Walk stops at `repoRoot` (or `$HOME` if `repoRoot` is null).
- Walk stops at the filesystem root if neither is reached.

#### Injection into the system prompt

Project-level context file content is rendered verbatim inside `<file path="...">` blocks in the system prompt:

```xml
<context>
Follow the context files below for all tasks:
<file path="/path/to/AGENTS.md">
...file content verbatim...
</file>
</context>
```

**No processing occurs on the content.** No path resolution, no `@`-include expansion, no directive handling. Absolute paths, relative paths, or any other text in the file appear as literal strings in the prompt.

#### Subdirectory AGENTS.md files

In addition to project-level context files, OMP scans **downward** from `cwd` for `AGENTS.md` files in subdirectories:

- Scan depth: 1 to 4 levels below `cwd`.
- Maximum 200 files discovered.
- Dotdirs (`.git`, `.omp`, `node_modules`, etc.) are excluded.
- Content is **not** read into the prompt.
- Only file paths are listed in a `<dir-context>` block:

```xml
<dir-context>
Some directories may have their own rules. Deeper rules override higher ones.
**MUST** read before making changes within:
- trucking/fallback/AGENTS.md
- trucking/offboard/triage/AGENTS.md
</dir-context>
```

The agent is expected to `read` these files on demand when working in the corresponding directories.

#### Constraints

- No frontmatter support -- the entire file is content.
- No `@`-include or `@`-import directives. This is a Claude Code feature, not an OMP feature.
- Content is injected verbatim -- no path resolution or transformation.
- Subdirectory `AGENTS.md` files are path-listed only, not content-injected.

#### Writing effective context files

- **Put project-wide constraints in the root context file.** Build system, dev environment, repo structure, coding conventions.
- **Use directory-scoped files for local overrides.** A `backend/AGENTS.md` can specify workspace-specific instructions without affecting other workspaces.
- **Avoid duplication with rules.** Context files are always loaded; rules are loaded on demand. Put "always needed" constraints in context files and domain-specific standards in rules.
- **Do not use `@`-includes in `AGENTS.md`.** OMP does not expand them. If you need to reference another file, name the path and instruct the agent to read it.

### 2.7. Custom Tools

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/tools/` | native | User | OMP |
| `<cwd>/.omp/tools/` | native | Project | OMP |
| `~/.claude/tools/` | claude | User | OMP, Claude Code |
| `<cwd>/.claude/tools/` | claude | Project | OMP, Claude Code |
| `~/.codex/tools/` | codex | User | OMP, Codex CLI |
| `<cwd>/.codex/tools/` | codex | Project | OMP, Codex CLI |

Custom tools are TypeScript/JavaScript modules that define LLM-callable functions. The model can invoke them during a turn just like built-in tools (read, edit, bash, etc.).

**When to use:** You need the model to call specific code during a turn -- running a linter, querying an API, checking a database, or any operation that requires executable logic with structured input/output.

**Module contract:** Export a default factory that receives `CustomToolAPI` and returns a tool definition (or array of definitions). Each tool needs `name`, `description`, `parameters` (TypeBox schema), and an `execute` function.

**API available to tools:**

| Method | Description |
|---|---|
| `pi.exec(cmd, args, opts)` | Run a subprocess (forward `signal` for cancellation) |
| `pi.ui` / `pi.hasUI` | UI context (guard with `hasUI` before interactive calls) |
| `pi.cwd` | Host working directory |
| `pi.typebox` | TypeBox for schema definitions |
| `pi.logger` | Shared file logger |
| `onUpdate(partial)` | Stream partial results to the UI during execution |

**Optional hooks:**

- `renderCall(args, theme)` and `renderResult(result, options, theme)` for custom TUI visualization.
- `onSession(event, ctx)` receives session events (start, switch, branch, shutdown, etc.) for state reconstruction.

**Constraints:**

- Tool names must be globally unique. Conflicts with built-ins or other custom tools are rejected.
- `.md` and `.json` files in tool directories are treated as metadata, not executable modules.
- Throwing in `execute` is treated as tool failure -- the agent sees `isError: true` with the error text.
- Forward `signal` to subprocess work for cooperative cancellation.

**Writing effective custom tools:**

- **Define a TypeBox schema for every parameter.** The agent sees the schema; vague parameters produce vague invocations.
- **Use `onUpdate` for long-running operations.** Stream partial results so the user sees progress instead of a frozen UI.
- **Guard UI access with `pi.hasUI`.** Tools run in headless and background modes too.

### 2.8. Agents (Task Subagents)

| Path | Scope | Tools |
|---|---|---|
| `~/.omp/agent/agents/*.md` | User | OMP |
| `<cwd>/.omp/agents/*.md` | Project | OMP |
| `~/.claude/agents/*.md` | User (legacy) | OMP |
| `<cwd>/.claude/agents/*.md` | Project (legacy) | OMP |

Agents are markdown files with YAML frontmatter that define subagent types for the `task` tool. When the main agent delegates work via the task tool, it selects an agent type (e.g., `explore`, `librarian`, `task`) that determines the subagent's system prompt, available tools, model, and thinking level.

**When to use:** You want to define a specialized subagent for a recurring type of delegated work, or override a built-in agent's behavior (e.g., give `explore` access to additional tools, or change `librarian`'s research procedure).

**Frontmatter fields:**

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Agent identifier used in the `task` tool's `agent` parameter. |
| `description` | Yes | One-line summary shown in the task tool documentation. |
| `tools` | No | Comma-separated list of tools the agent has access to (e.g., `read, grep, find, bash`). |
| `model` | No | Which role model to use (e.g., `pi/smol`, `pi/task`). Defaults to `pi/smol`. |
| `thinking-level` | No | Reasoning effort level (`minimal`, `low`, `med`, `high`). |
| `output` | No | JTD schema defining the structure of `submit_result` output. |

The markdown body after the frontmatter becomes the agent's system prompt.

**Built-in agents:**

| Name | Model | Tools | Purpose |
|---|---|---|---|
| `explore` | `pi/smol` | read, grep, find, web_search | Fast read-only codebase scout |
| `plan` | `pi/smol` | read, grep, find, bash, lsp, web_search, ast_grep | Architecture decisions for complex multi-file changes |
| `designer` | all | All tools | UI/UX implementation and review |
| `reviewer` | `pi/smol` | read, grep, find, bash, lsp, web_search, ast_grep | Code review and quality analysis |
| `oracle` | `pi/smol` | read, grep, find, bash, lsp, web_search, ast_grep | Deep reasoning advisor, read-only |
| `librarian` | `pi/smol` | read, grep, find, bash, lsp, web_search, ast_grep | External library and API research |
| `task` | `pi/task` | All tools | General-purpose subagent for multi-step tasks |
| `quick_task` | `pi/smol` | All tools | Low-reasoning mechanical updates |

**Precedence:** Filesystem-discovered agents load first, deduplicated by name (first seen wins). Built-in agents load last. A custom agent with the same `name` as a built-in **completely replaces** it.

**Discovery is per-invocation:** Agents are discovered each time the `task` tool is called, not at startup. New or modified agent files take effect on the next task delegation without restarting OMP.

**Constraints:**

- OMP-only. No cross-tool portability.
- The main agent (the one you talk to) is not an "agent" in this sense -- it uses the default model and the full system prompt. These agents are only for task tool delegation.
- Agent files must be valid markdown with parseable YAML frontmatter. Invalid files are skipped with a warning.

**Writing effective agents:**

- **Start by overriding a built-in.** Copy a built-in agent's prompt (from the OMP source at `packages/coding-agent/src/prompts/agents/`) and modify it, rather than writing from scratch.
- **Restrict tools to what's needed.** An agent with fewer tools is faster (smaller tool schema in context) and safer (can't accidentally modify files if it only has read access).
- **Define an `output` schema.** Structured output from `submit_result` is more useful to the calling agent than free-form text.
- **Use `pi/smol` for read-only agents.** The cheaper model is sufficient for investigation and research tasks.

### 2.9. Hooks (Legacy)

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/hooks/{pre,post}/*` | native | User | OMP |
| `<cwd>/.omp/hooks/{pre,post}/*` | native | Project | OMP |
| `~/.claude/hooks/{pre,post}/*` | claude | User | OMP, Claude Code |
| `<cwd>/.claude/hooks/{pre,post}/*` | claude | Project | OMP, Claude Code |

Hooks are the legacy event interception API. **In the current runtime, `--hook` is aliased to `--extension` and hooks are processed through the extension runner.**

**When to use:** You have existing hook modules that work. For new interception logic, use an extension instead.

Hooks can intercept tool calls (block/allow), modify tool results, filter LLM context, inject pre-agent messages, and cancel/customize session operations. They use the same factory pattern as extensions but with a `HookAPI` instead of `ExtensionAPI`.

**Constraints:**

- Hook API surface still works but extensions are the preferred and actively maintained path.
- `tool_call` handler errors propagate and block execution (fail-closed). Other event handler errors are caught and reported.
- Last returned result wins for `tool_result` overrides; first block wins for `tool_call`.

### 2.10. Prompts

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/prompts/*.md` | native | User | OMP |
| `<cwd>/.omp/prompts/*.md` | native | Project | OMP |

Reusable prompt templates in Codex format. Markdown files available via prompt template expansion at prompt time.

**When to use:** You want reusable prompt templates that are applied after slash command expansion -- a different mechanism from slash commands (which replace the `/name` text).

**Schema:** No frontmatter required. Name is derived from filename (without extension).

**Precedence:** Deduplicated by name; first seen wins.

**Constraints:**

- Applied after slash command expansion in the prompt pipeline -- distinct from file-based slash commands.
- No file watcher; new files require restart.

### 2.11. Instructions

| Path | Provider | Scope | Tools |
|---|---|---|---|
| `~/.omp/agent/instructions/*.md` | native | User | OMP |
| `<cwd>/.omp/instructions/*.md` | native | Project | OMP |

GitHub Copilot-style file-specific instructions with optional glob pattern matching.

**When to use:** You have Copilot-format instruction files and want OMP to honor them.

**Schema:**

| Field | Source | Description |
|---|---|---|
| `name` | Filename | Derived from filename (strips `.instructions.md` or `.md` suffix) |
| `content` | Body | Markdown instruction content |
| `applyTo` | Frontmatter | Optional glob pattern for file matching |

**Precedence:** Deduplicated by name; first seen wins.

**Constraints:**

- Native provider only -- no cross-tool portability.
- No file watcher; new files require restart.

### 2.12. System Prompt (`SYSTEM.md`)

| Path | Scope | Tools |
|---|---|---|
| `~/.omp/agent/SYSTEM.md` | User | OMP |
| `<cwd>/.omp/SYSTEM.md` | Project | OMP |
| `~/.omp/agent/APPEND_SYSTEM.md` | User | OMP |
| `<cwd>/.omp/APPEND_SYSTEM.md` | Project | OMP |

Custom system prompt files that modify or replace the agent's base system prompt. Distinct from context files (`CLAUDE.md`/`AGENTS.md`) which are user instructions shown in conversation -- `SYSTEM.md` modifies the underlying system prompt itself.

**When to use:** You need to change the agent's base personality, constraints, or behavior at the system prompt level rather than as user instructions.

**Two rendering paths:**

OMP assembles the system prompt from a template. Which template is used depends on whether `SYSTEM.md` exists:

| Condition | Template | Behavior |
|---|---|---|
| No `SYSTEM.md` | `system-prompt.md` (default, ~334 lines) | Full OMP prompt: identity, behavior, code integrity, stakes, contract, design integrity, 7-step procedure, tool precedence, parallelization, verification rules. Context files, skills, and rules are injected within this structure. |
| `SYSTEM.md` exists | `custom-system-prompt.md` (~69 lines) | **Full replacement.** Your `SYSTEM.md` content becomes the prompt. OMP's identity, behavior, contract, procedure, and all other built-in guidance are removed. Only context files, skills, rules, always-apply rules, git status, date, and cwd are retained in lightweight XML wrappers. |

**`APPEND_SYSTEM.md`:** Appends content to the prompt without replacing it. However, if `SYSTEM.md` is also present, the append content is stitched into the custom (replacement) path -- it does not restore the default prompt.

**`APPEND_SYSTEM.md` discovery:** OMP checks for `APPEND_SYSTEM.md` in project-level `.omp/` (via `findConfigFile`), then falls back to user-level `~/.omp/agent/APPEND_SYSTEM.md`. The file content is read as raw text (`Bun.file(path).text()`) and injected verbatim into the system prompt template as `{{appendPrompt}}`. **No processing occurs:** no path resolution, no include expansion, no directive handling. Absolute paths or references to other files appear as literal strings in the prompt.

**`SYSTEM.md` discovery:** Same nearest-ancestor `.omp/` walk as context files (see §2.6). Project-level overrides user-level. Read as raw text, no processing.

**Default prompt contents (when no `SYSTEM.md` exists):**

The base template includes, in order:

1. RFC 2119 keyword definitions and XML tag semantics
2. Workspace info (OS, CPU, GPU, terminal)
3. Context files (`CLAUDE.md`, `AGENTS.md`)
4. Identity: role ("distinguished staff engineer"), communication style, behavior guidelines
5. Code integrity principles (think outside-in, callers/system/time reasoning)
6. Stakes (high-reliability domain framing)
7. Environment: internal URLs, skills list, always-apply rules, rulebook rules, tool list
8. Tool precedence and usage rules (LSP > grep, AST tools > sed, etc.)
9. Contract (inviolable rules: no incomplete work, no fabricated outputs, full cutover)
10. Design integrity principles (one concept one representation, earn every abstraction)
11. 7-step procedure (scope, before-edit, parallelization, tracking, working, if-blocked, verification)
12. Session state (cwd, date, action directives)

**Schema:** No frontmatter. Entire file content is used.

**Precedence:** Project-level `SYSTEM.md` overrides user-level. Found via nearest-ancestor `.omp/` directory walk.

**Constraints:**

- `SYSTEM.md` is a **full replacement**, not an injection. Creating one removes the entire OMP identity, contract, procedure, and all built-in guidance. Only use this if you intend to write your own complete system prompt.
- `APPEND_SYSTEM.md` is the safe option for adding content without losing defaults -- but only when `SYSTEM.md` is absent.
- Both files are injected verbatim -- no `@`-includes, no path resolution, no content transformation.
- Native provider only -- not portable to other tools.
- The base prompt template is hardcoded in the OMP binary (`packages/coding-agent/src/prompts/system/system-prompt.md`). It cannot be edited without rebuilding OMP.
---

## 3. Managed State Files (Not User-Edited)

| File | Purpose |
|---|---|
| `~/.omp/agent/agent.db` | Agent state database |
| `~/.omp/agent/history.db` | Command/prompt history |
| `~/.omp/agent/models.db` | Model usage/cache |
| `~/.omp/agent/sessions/` | Session JSONL files |
| `~/.omp/agent/search-db/` | Search index |
| `~/.omp/agent/terminal-sessions/` | Terminal session state |

These are SQLite databases and directories managed by OMP. Do not edit by hand.

---

## 4. Decision Flow: What to Create

| Need | Create |
|---|---|
| Knowledge the agent should reference on demand | **Skill** |
| A prompt you want to reuse by typing `/name` | **Slash command** (`.md` file) |
| Coding standards for specific file types | **Rule** |
| Project-wide instructions always in context | **Context file** (`CLAUDE.md` / `AGENTS.md`) |
| LLM-callable executable function | **Custom tool** |
| Runtime behavior: event interception, shortcuts, UI | **Extension** |
| Custom TUI appearance | **Theme** |
| Add to the system prompt without replacing it | **`APPEND_SYSTEM.md`** |
| Replace the entire system prompt (removes all OMP defaults) | **`SYSTEM.md`** (caution: full replacement) |
| A reusable prompt AND code execution on invocation | **Extension** with `registerCommand` |
| Specialized subagent for delegated work | **Agent** (`.md` in `.omp/agents/`) |

Key distinction: skills, commands, rules, and context files are **passive content**. Extensions and custom tools are **active code** running in the OMP process.

---

## 5. Appendix: Hotkeys Quick Reference

### 5.1. Thinking

| Key | Action |
|---|---|
| `shift+tab` | Cycle thinking level (off -> minimal -> low -> medium -> high -> xhigh) |
| `ctrl+t` | Toggle thinking block visibility |

### 5.2. Navigation and Editing

| Key | Action |
|---|---|
| `ctrl+p` | Command palette / selector |
| `ctrl+k` | Clear / compact conversation |
| `ctrl+l` | Clear screen |
| `ctrl+o` | Open file picker |
| `ctrl+g` | Go to definition / navigate |
| `#` | Open prompt actions (copy, undo, cursor movement) |
| `/` | Slash command autocomplete |
| `alt+enter` | Newline in editor (multi-line input) |

### 5.3. Session Control

| Key | Action |
|---|---|
| `escape` | Cancel current operation / exit mode / close selector |
| `ctrl+c` | Clear editor (single press), shutdown (double press within 500ms) |
| `ctrl+d` | Shutdown (on empty editor) |
| `ctrl+z` | Suspend OMP (resume with `fg`) |

### 5.4. During Streaming

| Key | Action |
|---|---|
| `escape` | Abort current generation and restore queued messages to editor |

### 5.5. Reserved (Cannot Be Used by Extensions)

`ctrl+c`, `ctrl+d`, `ctrl+z`, `ctrl+k`, `ctrl+p`, `ctrl+l`, `ctrl+o`, `ctrl+t`, `ctrl+g`, `shift+tab`, `shift+ctrl+p`, `alt+enter`, `escape`, `enter`
