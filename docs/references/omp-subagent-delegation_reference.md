# Subagent Delegation Reference

Reference for oh-my-pi's subagent architecture: agent definitions, discovery, the task tool, execution engine, isolation, inter-agent communication, the swarm extension, and all extension points for customizing delegation behavior.

---

## 1. Architecture Overview

Delegation in oh-my-pi flows through three layers:

| Layer | Component | Location |
|---|---|---|
| Definition | `AgentDefinition` — subagent type descriptor | `packages/coding-agent/src/task/types.ts` |
| Discovery | `discoverAgents()` — merges bundled + filesystem agents | `packages/coding-agent/src/task/discovery.ts` |
| Execution | `TaskTool` → `runSubprocess()` — spawns in-process subagent sessions | `packages/coding-agent/src/task/index.ts`, `executor.ts` |

The main agent delegates work by calling the `task` tool (an LLM-callable function). The tool selects an agent type, configures it, and runs it via `runSubprocess()`. Subagents execute in-process with their own `AgentSession`, tool set, and system prompt.

```
main agent → task tool → runSubprocess → new AgentSession
                │
                ├─ discoverAgents(cwd)  → merged agent list
                ├─ getAgent(name)       → selected AgentDefinition
                ├─ check spawn policy   → allowed/blocked
                ├─ check depth limit    → task tool gated at max depth
                ├─ check disabled list  → agent availability
                └─ mapWithConcurrency   → parallel subagent execution
```

---

## 2. Agent Definition

### 2.1. Shape

```ts
interface AgentDefinition {
    name: string;            // unique identifier
    description: string;     // one-line summary for the task tool description
    systemPrompt: string;    // injected as subagent system prompt
    tools?: string[];        // tool allowlist (CSV/array; "yield" is auto-added)
    spawns?: string[];       // which agents this agent may spawn ("*" = any)
    model?: string;          // per-agent model role override
    thinkingLevel?: ThinkingLevel;
    output?: unknown;        // JTD output schema
    blocking?: boolean;      // force sync execution even when async enabled
    source: "bundled" | "user" | "project";
    filePath?: string;
}
```

`name`, `description`, and `systemPrompt` are required for a valid agent. `spawns` accepts `"*"`, a CSV string, or an array. If `spawns` is missing but `tools` includes `task`, `spawns` is auto-set to `"*"` for backward compatibility.

### 2.2. Bundled Agents

Embedded at build time in `packages/coding-agent/src/task/agents.ts` and `prompts/agents/`:

| Name | Model | Tools | Purpose |
|---|---|---|---|
| `explore` | `pi/smol` | read, grep, find, web_search | Fast read-only codebase scout |
| `plan` | `pi/smol` | read, grep, find, bash, lsp, web_search, ast_grep | Architecture decisions for multi-file changes |
| `designer` | all | All tools | UI/UX implementation and review |
| `reviewer` | `pi/smol` | read, grep, find, bash, lsp, web_search, ast_grep | Code review and quality analysis |
| `task` | `pi/task` | All tools | General-purpose multi-step subagent |
| `quick_task` | `pi/smol` | All tools | Low-reasoning mechanical updates |

Bundled agents load last and are filtered out if a filesystem agent with the same name already exists.

### 2.3. Custom Agent Files

Filesystem agents are `.md` files with YAML frontmatter:

```markdown
---
name: my-agent
description: Does specialized work
tools: read, edit, bash, yield
spawns: explore
model: claude-sonnet-4-5
output:
  properties:
    result: { type: "string" }
---
Your system prompt content here.
```

Missing `name` or `description` makes the file invalid; it is skipped with a warning.

---

## 3. Agent Discovery

`discoverAgents(cwd, home)` in `packages/coding-agent/src/task/discovery.ts` merges from four source categories, first-wins by exact name:

### 3.1. Discovery Sources

| Order | Source | Path Pattern |
|---|---|---|
| 1 | User config agent dirs | `getConfigDirs("agents", { project: false })` |
| 2 | Project config agent dirs | `findAllNearestProjectConfigDirs("agents", cwd)` |
| 3 | Claude plugin agents | `listClaudePluginRoots(home)` → `agents/` subdirs |
| 4 | Bundled agents | `loadBundledAgents()` |

### 3.2. Source Family Order

Each category iterates through source families in priority order:

| Priority | Source Family | Project Path | User Path |
|---|---|---|---|
| Highest | `.omp` | `<cwd>/.omp/agents/*.md` | `~/.omp/agent/agents/*.md` |
| | `.claude` | `<cwd>/.claude/agents/*.md` | `~/.claude/agents/*.md` |
| | `.codex` | `<cwd>/.codex/agents/*.md` | `~/.codex/agents/*.md` |
| Lowest | `.gemini` | `<cwd>/.gemini/agents/*.md` | `~/.gemini/agents/*.md` |

For each source family: project dir is checked before user dir. Plugin `agents/` dirs are appended after all source-family dirs.

### 3.3. Merge Rules

- Deduplication is by exact agent `name` (case-sensitive).
- First occurrence wins — higher-priority sources shadow lower.
- Project overrides user for the same source family.
- Non-bundled agents override bundled agents with the same name.
- Within one directory, files are read in lexicographic filename order.

### 3.4. Discovery Timing

Agents are **rediscovered at execution time** every time the `task` tool is called. New or modified agent files take effect on the next delegation without requiring a restart. The task tool description (shown to the model in the system prompt) is built from agents discovered at tool initialization, so it can be stale relative to the execution-time set if files change mid-session.

---

## 4. The Task Tool

`TaskTool` (in `packages/coding-agent/src/task/index.ts`) is the LLM-callable delegation mechanism. The model invokes `task` with structured parameters.

### 4.1. Parameters

```ts
interface TaskParams {
    agent: string;                              // agent name to use
    tasks: Array<{
        id: string;                             // CamelCase, ≤32 chars
        description: string;                    // UI label only
        assignment: string;                     // full self-contained instructions
    }>;
    context?: string;                           // shared background prepended to each assignment
    schema?: unknown;                           // JTD schema for structured output
    isolated?: boolean;                         // run in isolated filesystem
}
```

### 4.2. Execution Flow

```
TaskTool.execute(params)
    │
    ├─ validateTaskModeParams → reject disallowed fields
    │
    ├─ asyncEnabled && agent.blocking !== true && tasks.length > 0?
    │   YES → register background jobs via AsyncJobManager (one per task)
    │         with Semaphore(maxConcurrency)
    │   NO  → #executeSync
    │
    └─ #executeSync
          ├─ rediscover agents
          ├─ check: agent exists?
          ├─ check: agent in disabledAgents?
          ├─ check: parent spawn policy allows this agent?
          ├─ check: PI_BLOCKED_AGENT self-recursion guard?
          ├─ check: plan mode? → prepend plan-mode subagent prompt, force read-only tools, clear spawns
          ├─ resolve isolation backend (worktree / fuse-overlay / projfs / none)
          ├─ capture baseline if isolated
          ├─ mapWithConcurrencyLimit(runSubprocess, tasks, maxConcurrency)
          ├─ capture delta patches if isolated
          ├─ merge changes back (branch-based or nested-patch)
          └─ aggregate results → return to calling model
```

### 4.3. Guardrails

| Guardrail | Checked In | Behavior |
|---|---|---|
| Agent not found | `#executeSync` | Returns `Unknown agent "X". Available: ...` |
| Agent disabled | `#executeSync` | Returns immediate error listing enabled alternatives |
| Spawn policy denies | `#executeSync` | Returns `Cannot spawn 'X'. Allowed: ...` |
| Self-recursion | Tool construction | `PI_BLOCKED_AGENT` env var check; rejects immediately |
| Recursion depth | `runSubprocess` | At `maxRecursionDepth`: `task` tool removed from child, `spawns` env set to empty |

### 4.4. Parent Spawn Policy

`session.getSessionSpawns()` controls which agents subagents may themselves spawn:

| Value | Effect |
|---|---|
| `"*"` | Allow all |
| `""` | Deny all |
| CSV list | Allow only listed names |

### 4.5. Settings

| Setting | Type | Default | Effect |
|---|---|---|---|
| `task.maxConcurrency` | number | — | Max parallel subagents (semaphore limit) |
| `task.maxRecursionDepth` | number | — | Depth at which sub-subagent spawning is cut off |
| `task.isolation.mode` | string | `"none"` | Filesystem isolation: `none`, `worktree`, `fuse-overlay`, `fuse-projfs` |
| `task.disabledAgents` | string[] | `[]` | Names of agents the model cannot use |
| `task.simple` | string | `"default"` | Mode: `default`, `schema-free`, `independent` |
| `async.enabled` | boolean | `false` | Enable background job execution for tasks |

### 4.6. Simple Modes

| Mode | `context` field | `schema` field | Behavior |
|---|---|---|---|
| `default` | Accepted | Accepted | Full capabilities |
| `schema-free` | Accepted | **Rejected** | No custom schema; use agent definition or session schema |
| `independent` | **Rejected** | **Rejected** | Everything must be in assignment text |

---

## 5. Subagent Execution Engine

`runSubprocess(options)` in `packages/coding-agent/src/task/executor.ts` creates and runs a subagent in-process.

### 5.1. ExecutorOptions

```ts
interface ExecutorOptions {
    cwd: string;
    worktree?: string;                    // isolated workspace path
    agent: AgentDefinition;
    task: string;                         // rendered task text
    assignment?: string;
    description?: string;
    index: number;
    id: string;
    modelOverride?: string | string[];
    thinkingLevel?: ThinkingLevel;
    outputSchema?: unknown;
    taskDepth?: number;                   // 0 = top-level, increments per nesting
    enableLsp?: boolean;
    signal?: AbortSignal;
    onProgress?: (progress: AgentProgress) => void;
    sessionFile?: string | null;
    persistArtifacts?: boolean;
    artifactsDir?: string;
    contextFile?: string;
    eventBus?: EventBus;
    contextFiles?: ContextFileEntry[];
    skills?: Skill[];
    promptTemplates?: PromptTemplate[];
    agentsMdSearch?: AgentsMdSearch;
    workspaceTree?: WorkspaceTree;
    mcpManager?: MCPManager;
    authStorage?: AuthStorage;
    modelRegistry?: ModelRegistry;
    settings?: Settings;
    localProtocolOptions?: LocalProtocolOptions;
    parentHindsightSessionState?: HindsightSessionState;
}
```

### 5.2. Execution Steps

1. **Initialize progress** — create `AgentProgress` with status `"running"`
2. **Check abort** — if signal already aborted, return immediately
3. **Create subagent session** via `createAgentSession()` with:
   - The agent's system prompt
   - Configured tool set (from `agent.tools` and `agent.spawns`)
   - Resolved model via `resolveModelOverride()`
   - Subagent settings (parent snapshot with `async.enabled: false`, `bash.autoBackground.enabled: false`)
4. **Wire MCP proxy tools** — if MCP manager exists, create proxy tools that reuse parent connections
5. **Register with AgentRegistry** — enables IRC discovery
6. **Run agent loop** — forwarding `AgentEvent` types for progress tracking
7. **Finalize output** via `finalizeSubprocessOutput()`:
   - If `yield` was called: use last yield's data
   - If no yield: attempt JSON fallback parsing
   - Validate against output schema via Ajv
   - Emit structured warnings for missing/null yield

### 5.3. Tool Set Construction

The subagent receives:
- Tools listed in `agent.tools` (parsed from CSV or array)
- `yield` tool is always auto-added
- If `agent.spawns` includes `task`, the `task` tool is added
- MCP proxy tools (one per parent MCP tool) if MCP manager is available
- At max recursion depth: `task` tool is removed, `spawns` env is set to empty

### 5.4. Output Extraction

Subagents are instructed (via system prompt) to submit results using the `yield` tool. The executor tracks `yieldItems[]` — each yield appends to the array. On subagent exit, the **last** yield item is used.

| Case | Behavior |
|---|---|
| `yield` called with data | Serialize data as JSON; exit code 0 |
| `yield` called with `status: "aborted"` | Return `{ aborted: true, error: "..." }` |
| `yield` called with null data | Prepend warning: `SYSTEM WARNING: Subagent called yield with null data` |
| No `yield`, exit 0, has schema | Attempt JSON fallback from raw output; validate against schema |
| No `yield`, exit 0, no schema, raw output | Pass raw output as-is |
| No `yield`, exit 0, no raw output | Warning: `SYSTEM WARNING: Subagent exited without calling yield tool after 3 reminders` |

### 5.5. Schema Precedence (Structured Output)

1. Task call `params.schema` (when `task.simple` allows custom schemas)
2. Agent frontmatter `output`
3. Parent session `outputSchema`

### 5.6. Event Forwarding

The following `AgentEvent` types are forwarded from subagent to parent for progress tracking:

```
agent_start, agent_end, turn_start, turn_end,
message_start, message_update, message_end,
tool_execution_start, tool_execution_update, tool_execution_end
```

Three EventBus channels carry subagent events:
- `task:subagent:event` — raw event stream
- `task:subagent:progress` — aggregated progress summaries
- `task:subagent:lifecycle` — start/end lifecycle events

---

## 6. Filesystem Isolation

`packages/coding-agent/src/task/worktree.ts` and `isolation-backend.ts` implement three strategies:

### 6.1. Isolation Modes

| Mode | Mechanism | OS | Behavior |
|---|---|---|---|
| `none` | No isolation | All | Subagent works directly in `cwd` |
| `worktree` | `git worktree add` | All | Separate git worktree directory; changes committed to a branch and cherry-picked |
| `fuse-overlay` | FUSE overlayfs | Linux/macOS | Lower=original dir, upper=tmp dir; writes go to upper |
| `fuse-projfs` | Windows ProjFS | Windows | Virtualized filesystem projection |

### 6.2. Isolation Lifecycle

```
resolveIsolationBackendForTaskExecution(requestedMode, isIsolated, repoRoot)
    │
    ├─ mode === "none" or !isIsolated → no isolation
    │
    ├─ repoRoot === null → forced "none" (no git repo)
    │
    ├─ "worktree" → ensureWorktree() → captureBaseline() → applyBaseline()
    ├─ "fuse-overlay" → ensureFuseOverlay() (Unix only)
    └─ "fuse-projfs" → ensureProjfsOverlay() (Windows only)
    │
    ▼
Run subagent inside isolated directory
    │
    ▼
Capture changes:
    ├─ worktree: commitToBranch() → mergeTaskBranches() → cleanupTaskBranches()
    └─ overlay:  captureDeltaPatch() → applyNestedPatches() → cleanupFuseOverlay/ProjfsOverlay
```

### 6.3. Per-Task Isolation

The `isolated` parameter on individual task items enables per-task isolation. When set, each task gets its own workspace. The parent's `params.isolated` controls the default; individual tasks can override.

---

## 7. Plan Mode Delegation

When the parent session is in plan mode, subagents receive a modified execution:

1. A plan-mode-specific subagent prompt is prepended (`prompts/system/plan-mode-subagent.md`)
2. The prompt forces read-only operation:
   ```
   Plan mode active. You MUST perform READ-ONLY operations only.
   You MUST NOT: create, edit, delete, move, or copy files,
   run state-changing commands, or make any changes to the system.
   ```
3. The subagent is cast as an "architect and planning specialist"
4. Output format requires a "Critical Files for Implementation" section

**Implementation caveat**: In the current code, `TaskTool.execute()` computes an `effectiveAgent` for model/thinking/output derivation, but `runSubprocess()` is called with the original `agent` parameter, not `effectiveAgent`. System prompt and tool/spawn restrictions from `effectiveAgent` are not passed through. This is a known limitation.

---

## 8. Inter-Agent Communication

### 8.1. Agent Registry

`AgentRegistry` (in `packages/coding-agent/src/registry/agent-registry.ts`) is a process-global singleton tracking all live `AgentSession` instances:

```ts
interface AgentRef {
    id: string;             // "0-Main" for main, generated IDs for subagents
    displayName: string;    // agent name + task description
    status: "running" | "idle" | "completed" | "aborted";
    kind: "main" | "sub";
}
```

### 8.2. IRC Tool

Agents communicate via a dedicated `irc` tool. On subagent startup, the executor renders a peer roster:

```
renderIrcPeerRoster(selfId) →
  - 0-Main — main session (main, running)
  - 0-Explore-Ab12 — explore subagent (sub, running)
```

Subagents can `irc send` to peers by ID or broadcast to `all`. The `irc list` operation returns currently visible peers. Only agents with status `running` or `idle` appear in the roster.

### 8.3. Subprocess Tool Registry

`subprocessToolRegistry` (in `packages/coding-agent/src/task/subprocess-tool-registry.ts`) is a singleton that allows registration of per-tool event handlers scoped to subagent execution. These handlers receive parsed tool events from subagent JSONL output and can extract structured data for progress tracking.

---

## 9. Extension Points for Customizing Delegation

### 9.1. Custom Agent Definitions

Add `.md` files to any discovery directory:

```
<project>/.omp/agents/my-agent.md         → project-scoped
~/.omp/agent/agents/my-agent.md           → user-scoped
<project>/.claude/agents/my-agent.md      → Claude-compatible
```

All agent directories use non-recursive scanning of one level.

### 9.2. Extension API Events

Extensions (`packages/coding-agent/src/extensibility/extensions/`) can intercept delegation at these points:

| Event | When | Can block/modify? |
|---|---|---|
| `tool_call` | Before any tool execution (including `task`) | **Block** (`{block: true}`) |
| `tool_result` | After any tool execution | **Modify** `content`/`details` |
| `session_start` | Session creation (including subagent sessions) | Observe only |
| `session_before_compact` | Before compaction | **Cancel**, provide custom compaction |
| `session_before_tree` | Before tree navigation | **Cancel**, provide summary |
| `context` | Before LLM context assembly | **Replace** messages |
| `before_agent_start` | Before agent begins generation | Inject pre-agent message |

Extensions can also:
- Register new callable tools (`pi.registerTool(...)`)
- Register slash commands (`pi.registerCommand(...)`)
- Send messages with delivery control (`sendMessage`, `sendUserMessage`)
- Access session branching/navigation APIs

### 9.3. Custom Tools

Tool modules from `~/.omp/agent/tools/`, `<cwd>/.omp/tools/`, `.claude/tools/`, `.codex/tools/`, and plugin manifests. Custom tools are **always included** in subagent tool sets (names must not conflict with built-ins). Module contract:

```ts
export default function (pi: CustomToolAPI) {
    return {
        name: "my_tool",
        description: "...",
        parameters: pi.typebox.Type.Object({ ... }),
        async execute(toolCallId, params, onUpdate, ctx, signal) {
            return { content: [{ type: "text", text: "done" }], details: {} };
        },
    };
}
```

### 9.4. Skills

Passive knowledge packs from `skills/<name>/SKILL.md`. Skills are **forwarded to subagents** through normal session creation. There is no per-task skill pinning override. Skills inject metadata into the system prompt and are readable via `skill://<name>`.

### 9.5. Prompt Templates

The subagent system prompt and related prompts are Handlebars-templated `.md` files:

| File | Purpose |
|---|---|
| `prompts/system/subagent-system-prompt.md` | Full subagent system prompt |
| `prompts/system/plan-mode-subagent.md` | Injected during plan mode |
| `prompts/system/subagent-yield-reminder.md` | 3-strike yield reminder |
| `prompts/tools/task.md` | Task tool description rendered to the model |

### 9.6. Settings-Based Control

All gating behavior is settings-driven via `settings-schema.ts`:

```yaml
# ~/.omp/agent/config.yml
task:
  maxConcurrency: 4
  maxRecursionDepth: 3
  isolation:
    mode: worktree         # none | worktree | fuse-overlay | fuse-projfs
  disabledAgents:
    - designer
  simple: default          # default | schema-free | independent

async:
  enabled: true

extensions:
  - ~/my-exts/delegation-policy.ts

disabledExtensions:
  - extension-module:some-ext
```

```json
// <cwd>/.omp/settings.json
{
    "task.maxConcurrency": 2,
    "task.disabledAgents": ["designer"]
}
```

---

## 10. Swarm Extension

The swarm extension (`packages/swarm-extension/`) is a YAML-driven multi-agent orchestration layer built atop `runSubprocess`. It provides two entry points: a standalone CLI runner (`omp-swarm`) and a TUI integration (`/swarm run`).

### 10.1. Architecture

```
swarm.yaml
    │
    ▼
schema.ts          → parse + validate YAML → SwarmDefinition
    │
    ▼
dag.ts             → build dep graph → detect cycles → topological sort → waves
    │
    ▼
pipeline.ts        → iteration loop (pipeline mode) + wave controller
    │
    ▼
executor.ts        → executeSwarmAgent() → runSubprocess()
    │
    ▼
state.ts           → filesystem state (.swarm_<name>/state/pipeline.json + logs)
```

### 10.2. YAML Schema

```yaml
swarm:
  name: my-pipeline
  workspace: ./workspace
  mode: pipeline          # pipeline | sequential | parallel
  target_count: 10        # iterations (pipeline mode only, default: 1)
  model: claude-opus-4-6  # default model (optional)

  agents:
    agent_name:
      role: short-role-name           # becomes system prompt
      task: |
        Full instructions for this agent.
      extra_context: |                # appended to system prompt (optional)
        Additional context.
      reports_to:                     # agents that depend on this one
        - downstream_agent
      waits_for:                      # agents this one depends on
        - upstream_agent
      model: claude-sonnet-4-5        # per-agent override (optional)
```

### 10.3. Execution Modes

| Mode | Behavior |
|---|---|
| `pipeline` | Full agent graph repeats `target_count` times. Each iteration runs all waves in order. |
| `sequential` | Run each agent once, chained by declaration order (or explicit deps). Default mode. |
| `parallel` | All agents run simultaneously (pending dependency constraints). |

### 10.4. Dependency Model

- `waits_for: [a, b]` — agent blocks until both `a` and `b` complete
- `reports_to: [x]` — equivalent to `x` having `waits_for: [this_agent]`
- No explicit deps + `pipeline`/`sequential` mode → agents chain by YAML declaration order
- No explicit deps + `parallel` mode → all agents in one wave
- Cycles are detected and rejected before execution

### 10.5. Inter-Agent Communication

The swarm orchestrator manages lifecycle and ordering only. Agents communicate through **shared workspace files**. Common patterns:

| Pattern | Example |
|---|---|
| Signal files | `signals/finder_out.txt` → `"FOUND:https://example.com"` |
| Structured output | `analyzed/item_1.md`, `results/report.json` |
| Tracking files | `processed.txt`, `tracking/count.txt` |

### 10.6. State Persistence

While running, state is written to `<workspace>/.swarm_<name>/`:

```
.swarm_<name>/
  state/pipeline.json    # Live pipeline status + per-agent status
  logs/orchestrator.log  # Wave transitions, iteration progress
  logs/<agent>.log       # Per-agent timestamps and errors
  context/               # Agent session artifacts
```

### 10.7. Entry Points

| Entry Point | Usage | Notes |
|---|---|---|
| Standalone CLI | `omp-swarm path/to/swarm.yaml` | No timeout, runs until complete |
| Background | `nohup omp-swarm ... & disown` | Survives terminal close |
| TUI command | `/swarm run path/to/swarm.yaml` | Requires extension registration |
| Status check | `/swarm status <name>` | Reads persisted state |

---

## 11. Session Handoff

The `/handoff` command (not task delegation but a related session-continuation mechanism) generates a summary of the current session, creates a brand-new session, and injects the summary as a `custom_message` entry:

```
New session:
  <handoff-context>
  ...generated handoff text...
  </handoff-context>

  The above is a handoff document from a previous session.
  Use this context to continue the work seamlessly.
```

The new session's LLM context contains only the handoff, not the old transcript. The handoff is extracted heuristically from the last `assistant` message's text blocks. No lineage metadata (`parentSession`) is set in this path. When `compaction.handoffSaveToDisk` is enabled, a timestamped `handoff-*.md` artifact is saved under the session artifacts directory.

---

## 12. Key Constraints and Edge Cases

| Constraint | Details |
|---|---|
| Agent names are case-sensitive | `Task` and `task` are distinct |
| Discovery is per-invocation | Agents rediscovered on each `task` call; no restart needed |
| Tool description can be stale | Description built at init; runtime set may differ mid-session |
| Invalid agent files are skipped | One bad file doesn't abort discovery for other files |
| Extensions run same-process | No sandbox; same EventBus and ExtensionRuntime |
| `tool_call` handler errors block | Fail-closed: throwing in a `tool_call` handler blocks the tool |
| Reserved shortcuts ignored | Extensions cannot override `ctrl+c`, `ctrl+d`, `escape`, etc. |
| Plan mode subagent restrictions | Read-only tools enforced; spawns cleared; plan-specific prompt prepended |
| Max recursion depth | At limit, `task` tool removed from child tool set entirely |
| Self-recursion blocked | `PI_BLOCKED_AGENT` env var prevents agent from spawning itself |
| Swarm cycles rejected | DAG cycle detection runs before any agents execute |
