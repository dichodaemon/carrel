---
title: OMP Extensibility Mapping
date: 2026-06-29
author: Dizan Vasquez
---

# OMP Extensibility Mapping

## 1. Objective

Map every extension point in vanilla OMP — capabilities, mechanisms, and invocation patterns — into a single structured inventory that disambiguates the layers and their relationships.

## 2. Scope

| In scope | Out of scope |
|---|---|
| OMP capability inventory (from `capability/*.ts`) | Carrel deployment |
| Mapping capabilities to mechanisms | Settings schema enumeration |
| Mapping capabilities to invocation patterns | Provider precedence details |
| Agents and themes (non-capability extension points) | Plugin/marketplace distribution |
| Layering model (mechanism → capability → invocation) | |

## 3. Sources

1. Capability definitions: `/workspace/oh-my-pi/packages/coding-agent/src/capability/*.ts`
2. Internal OMP docs under `omp://`:
   - `omp://slash-command-internals.md` — command resolution pipeline
   - `omp://custom-tools.md` — custom tool module contract
   - `omp://context-files.md` — context file discovery and injection
   - `omp://task-agent-discovery.md` — agent definition fields and discovery
   - `omp://extension-loading.md` — extension module loading
   - `omp://skills/authoring-extensions.md` — extension API surface
   - `omp://system-prompt-customization.md` — SYSTEM.md/APPEND_SYSTEM.md behavior
   - `omp://marketplace.md` — plugin distribution layer
3. Existing reference document: `docs/references/omp-configuration_reference.md`

## 4. Approach

Start with the capability inventory extracted from the codebase, then layer on mechanisms, invocation patterns, and cross-cutting concerns. Expand the mapping as each layer is verified.

## 5. Capability Inventory

Fourteen capabilities registered via `defineCapability`, plus two extension points that bypass the capability registry. Mechanism mappings are in §6.

| ID | Display Name |
|---|---|
| `context-files` | Context Files |
| `extension-modules` | Extension Modules |
| `extensions` | Extensions |
| `hooks` | Hooks |
| `instructions` | Instructions |
| `mcps` | MCP Servers |
| `prompts` | Prompts |
| `rules` | Rules |
| `settings` | Settings |
| `skills` | Skills |
| `slash-commands` | Slash Commands |
| `ssh` | SSH Hosts |
| `system-prompt` | System Prompt |
| `tools` | Custom Tools |

Non-capability extension points:

| Name |
|---|
| Agents (task subagents) |
| Themes |

`extension-modules` feeds two runtime mechanisms not represented as capabilities: Command Handler (via `pi.registerCommand()`) and Tool (via `pi.registerTool()`). These are invocation-time behaviors, not discovered items.

## 6. Mechanism Inventory

Twelve distinct runtime mechanisms. Canonical names and definitions:

| # | Mechanism | Definition |
|---|---|---|
| 1 | **Template Expansion** | Text substitution (`$1`, `$ARGUMENTS`) on a markdown file; result inserted into the prompt. No code execution. |
| 2 | **Command Handler** | TypeScript function registered via `pi.registerCommand()` that executes when the user types `/name`. Runs code, can spawn processes, can return void or replacement text. |
| 3 | **Tool** | TypeScript function that executes when the agent calls it during a turn. Registered via `pi.registerTool()` or as a standalone module in `tools/`. Returns content + details. |
| 4 | **File Injection — Static** | Content placed into the system prompt at session start. The agent does not request it — it is always present. |
| 5 | **File Injection — On-Demand** | Content loaded when the agent explicitly requests it via `skill://<name>` or `rule://<name>` through the read tool. |
| 6 | **Stream Interception** | Regex or AST pattern matches mid-generation on the agent's output stream; matching aborts the turn, injects rule content, and retries. |
| 7 | **Process Spawning** | External child process started and communicated with via stdio or HTTP/SSE (MCP protocol). |
| 8 | **Script Execution** | Shell script run via `execCommand()` before or after tool calls. |
| 9 | **Module Loading** | TypeScript/JavaScript file dynamically `import()`ed; its factory function executes and receives the ExtensionAPI. |
| 10 | **Configuration Merging** | JSON/YAML/TOML files parsed and deep-merged into the runtime Settings object. |
| 11 | **Connection Configuration** | SSH host entries surfaced to the SSH tool and `ssh://` URL resolver. |
| 12 | **Manifest Interpretation** | Gemini-style JSON manifest parsed; provides MCP servers, tools, and context references. |

## 7. Invocation Pattern Inventory

Eight invocation patterns — how capabilities are triggered at runtime.

| Pattern | Definition |
|---|---|
| **Session start** | Automatic at initialization. Capability items are loaded and applied before the first turn. |
| **Agent tool call** | Agent invokes a tool during a turn. The tool's `execute` function runs. |
| **On-demand read** | Agent explicitly requests content via `skill://<name>` or `rule://<name>` through the read tool. |
| **Pipeline step** | Automatic during prompt processing. Text is expanded via template substitution before reaching the agent. |
| **Stream match** | Regex or AST pattern matches mid-generation. The turn is aborted, rule content is injected, and generation retries. |
| **Before/after tool** | Automatic hook execution immediately before or after a tool call. |
| **URL resolution** | Automatic when the agent reads an `ssh://` URL. The capability provides host configuration. |
| **User slash command** | User types `/name` in the prompt. The command handler executes or the template expands. |

## 8. Capability–Invocation Mapping

| Capability | Invocation |
|---|---|
| `context-files` | Session start |
| `extension-modules` | Session start |
| `extensions` | Session start |
| `hooks` | Before/after tool |
| `instructions` | _(none)_ |
| `mcps` | Session start, Agent tool call |
| `prompts` | Pipeline step |
| `rules` (always-apply) | Session start |
| `rules` (rulebook) | On-demand read |
| `rules` (TTSR) | Stream match |
| `settings` | Session start |
| `skills` | On-demand read, User slash command |
| `slash-commands` | Pipeline step |
| `ssh` | Agent tool call, URL resolution |
| `system-prompt` | Session start |
| `tools` | Agent tool call |
| Agents | Agent tool call |
| Themes | Session start, User slash command |

## 9. Capability–Mechanism Mapping

### Capability → Mechanism

| Capability | Mechanism(s) |
|---|---|
| `context-files` | File Injection — Static |
| `extension-modules` | Module Loading |
| `extensions` | Manifest Interpretation |
| `hooks` | Script Execution |
| `instructions` | _(none — no runtime consumer)_ |
| `mcps` | Process Spawning |
| `prompts` | Template Expansion |
| `rules` | Stream Interception, File Injection — Static, File Injection — On-Demand |
| `settings` | Configuration Merging |
| `skills` | File Injection — On-Demand |
| `slash-commands` | Template Expansion |
| `ssh` | Connection Configuration |
| `system-prompt` | File Injection — Static |
| `tools` | Tool |
| Agents | File Injection — Static |
| Themes | Theme Rendering |

### Mechanism → Capability

| Mechanism | Capabilities |
|---|---|
| Template Expansion | `slash-commands`, `prompts` |
| Command Handler | _(registered by `extension-modules`; not a capability)_ |
| Tool | `tools`, _(registered by `extension-modules`)_ |
| File Injection — Static | `context-files`, `rules` (always-apply), `system-prompt`, Agents |
| File Injection — On-Demand | `skills`, `rules` (rulebook) |
| Stream Interception | `rules` (TTSR) |
| Process Spawning | `mcps` |
| Script Execution | `hooks` |
| Module Loading | `extension-modules` |
| Configuration Merging | `settings` |
| Connection Configuration | `ssh` |
| Manifest Interpretation | `extensions` |
| Theme Rendering | Themes |

