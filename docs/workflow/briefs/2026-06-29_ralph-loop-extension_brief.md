---
title: Ralph Loop Extension Module
date: 2026-06-29
author: Dizan Vasquez
---

# Ralph Loop Extension Module

## 1. Objective

Build a programmatic review→implement→verify loop for implementation plan execution, driven by a Command Handler registered via an OMP extension module. The user types `/ralph-execute path/to/plan.md` and the loop runs each task through three distinct agent passes, iterating on verification failure.

## 2. Scope

| In scope | Out of scope |
|---|---|
| Extension module registering a `/ralph-execute` command | Parallel task dispatch |
| Per-task review → implement → verify loop with iteration | Modifying the existing `execute-plan` skill |
| Agent dispatch via `createAgentSession()` (public SDK) | Standalone RPC script |
| Bead management via `bd` CLI | |
| Progress display via `ctx.ui` | |

## 3. Sources

1. OMP extensibility mapping: `docs/workflow/briefs/2026-06-29_omp-extensibility-mapping_brief.md`
2. Extension authoring: `omp://skills/authoring-extensions.md`
3. Extension loading: `omp://extension-loading.md`
4. OMP SDK: `@oh-my-pi/pi-coding-agent` exports `createAgentSession`, `AgentSession`
5. Task agent discovery: `@oh-my-pi/pi-coding-agent/task/discovery` exports `discoverAgents`, `getAgent`
6. Task agent types: `@oh-my-pi/pi-coding-agent/task/types` exports `AgentDefinition`
7. Agent definition format: built-in agents at `prompts/agents/*.md` in the OMP source
8. Ralph implementation report: `docs/workflow/reports/2026-06-29_ralph-implementations_report.md`

## 4. Approach

Register a Command Handler via `pi.registerCommand("ralph-execute", ...)`. The handler reads the plan, manages beads via `bd` CLI, and dispatches three agent types using the public `createAgentSession()` API per task.

### 4.1 Agent Resolution via OMP Agent Discovery

The extension resolves agents by **convention-based name** through OMP's existing agent discovery pipeline. Each Ralph role maps to an agent definition file in `.omp/agents/`:

| Role | Agent name | File | Purpose |
|---|---|---|---|
| Reviewer | `ralph-reviewer` | `.omp/agents/ralph-reviewer.md` | Reviews task definitions for consistency and completeness |
| Implementer | `ralph-implementer` | `.omp/agents/ralph-implementer.md` | Executes task with full tool access |
| Verifier | `ralph-verifier` | `.omp/agents/ralph-verifier.md` | Verifies done conditions via build/test |

If a `ralph-*` agent is not found, the extension falls back to built-in agents (`reviewer`, `task`, `task` respectively).

### 4.2 Agent Discovery Flow

```ts
import { discoverAgents, getAgent } from "@oh-my-pi/pi-coding-agent/task/discovery";
import { createAgentSession } from "@oh-my-pi/pi-coding-agent";
import type { AgentDefinition } from "@oh-my-pi/pi-coding-agent/task/types";

async function resolveAgent(
  cwd: string,
  preferredName: string,
  fallbackName: string,
): Promise<AgentDefinition | undefined> {
  const { agents } = await discoverAgents(cwd);
  return getAgent(agents, preferredName) ?? getAgent(agents, fallbackName);
}

// Per-task dispatch:
const reviewer = await resolveAgent(ctx.cwd, "ralph-reviewer", "reviewer");
const { session } = await createAgentSession({
  cwd: ctx.cwd,
  systemPrompt: reviewer.systemPrompt,
  modelPattern: reviewer.model?.[0],
  toolNames: reviewer.tools,
  disableExtensionDiscovery: true, // fresh context per pass
});
await session.prompt("Review this task for consistency...");
// wait for agent_end, extract result
session.dispose();
```

### 4.3 Agent Definition Format

Users create agent definitions as markdown files with YAML frontmatter — the same format OMP's `task` tool uses for subagent dispatch:

```markdown
---
name: ralph-reviewer
description: Plan task reviewer for Ralph loop — checks task definitions for consistency and completeness
tools: [read, grep, glob, lsp]
model: anthropic/claude-haiku-4-5
---
You are a plan task reviewer. Your job is to inspect a task definition for:
1. Clarity — can the implementer understand what to do?
2. Completeness — are acceptance criteria explicit?
3. Consistency — does the task align with its companion documents?

If the task is underspecified, report what's missing and BLOCK execution.
Do NOT guess or fill gaps — that's the implementer's job.
```

```markdown
---
name: ralph-implementer
description: Task implementer for Ralph loop — executes plan tasks with full tool access
tools: [read, edit, write, bash, eval, grep, glob, lsp, ast_grep, ast_edit]
model: anthropic/claude-sonnet-4-5
---
You are a task implementer. Execute the assigned task exactly as specified.
Do not modify the task's scope or acceptance criteria.
Report what you built, changed, and tested.
```

```markdown
---
name: ralph-verifier
description: Task verifier for Ralph loop — checks that done conditions are satisfied
tools: [bash, read, grep, glob, lsp]
model: anthropic/claude-haiku-4-5
---
You are a task verifier. Check that every acceptance criterion is demonstrably met.
Run the verification commands specified in the task.
Respond with PASS (all criteria met, evidence attached) or FAIL (specific criterion not met, why).
```

### 4.4 Loop Per Task

The loop per task: claim bead → review (block bead if unclear) → implement → verify → if fail, loop back to implement (max 3 iterations) → close bead → commit.

The extension module lives in its own repository. Installed during development via `omp plugin link /path/to/ralph-loop`. For distribution, publish to a marketplace.

## 5. Constraints

- No parallel task dispatch. Tasks execute sequentially within each phase.
- Three-iteration limit per task. After three implement→verify cycles, mark the bead blocked.
- Reviewer blocks on underspecified tasks — does not guess. User unblocks by clarifying the bead description.
- The extension runs inside OMP's process. A long-running loop occupies the session.

## 6. Open Questions

- Should the reviewer prompt the user interactively (via extension UI) or block the bead silently with a note?
- Should the extension reuse the existing `execute-plan` skill's bead scaffolding, or scaffold its own beads from the plan?
