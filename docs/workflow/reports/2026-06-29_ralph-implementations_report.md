---
title: Ralph Loop Implementation Survey
date: 2026-06-29
author: Dizan Vasquez
---

# Ralph Loop Implementation Survey

## 1. Summary

Comprehensive survey of every Ralph loop implementation on the Pi package registry (`pi.dev/packages?name=ralph`) as of 2026-06-29. 22 packages matched; 16 are genuine Ralph loop or multi-agent iteration systems. Four architectural families emerged: flat single-agent, external subprocess, multi-agent session, and LLM chained. Only two implementations support per-agent model selection.

## 2. Scope

| In scope | Out of scope |
|---|---|
| All Pi packages matching "ralph" on pi.dev as of 2026-06-29 | OMP or carrel internals |
| Multi-agent spawning, coordination, progress evaluation | Installation UX, documentation quality |
| Agent/model configuration mechanisms | Performance benchmarks |

## 3. Sources

1. Pi package registry: `https://pi.dev/packages?name=ralph` (accessed 2026-06-29)
2. Source repositories for each implementation (see per-implementation URLs in §4)
3. npm registry for packages without public repos: `https://www.npmjs.com` (accessed 2026-06-29)

## 4. Findings


### 4.1. Taxonomy


Ralph implementations fall into four architectural families:

| Family | Members | Core mechanism |
|---|---|---|
| **Flat single-agent** | pi-ralph-wiggum, Unipi/ralph, pi-review-loop, pi-until-done, orchid/ralph | Same agent re-fired via `sendUserMessage(followUp)`; agent self-drives loop progress |
| **External subprocess** | d-kimuson/pi-ralph, lnilluv/pi-ralph-loop, pi-wiggum, rahulmutt/pi-ralph | `pi --mode json -p` or `pi --mode rpc` or `ctx.newSession()` spawns a fresh agent process |
| **Multi-agent session** | ralpi, ralphi, pi-ralph (samfp) | `createAgentSession()` or child sessions, distinct prompts per role |
| **LLM chained** | entelligentsia/pi-ralph | Generates output → Critique rates it → Judge decides; all one model |

---

### 4.2. Implementation Details

#### 4.2.1 pi-ralph-wiggum (@tmustier/pi-ralph-wiggum)

**URL**: `https://github.com/tmustier/pi-extensions/tree/main/pi-ralph-wiggum`
**Source**: Single file `index.ts` (824 lines)
**Downloads**: 2,213/mo

**Architecture**: Flat single-agent. The agent works inside Pi, calls `ralph_done` tool to advance to next iteration. No subagents, no subprocesses.

**Multi-agent spawning**: **None**. Single agent self-drives.

**Coordination**: Event-driven follow-up loop:
1. `ralph_done` tool handler increments iteration counter → calls `pi.sendUserMessage(buildPrompt(state, content), { deliverAs: "followUp" })` → agent receives next iteration prompt.
2. `agent_end` handler checks for completion marker `<promise>COMPLETE</promise>` → `completeLoop()` if found.
3. `before_agent_start` handler injects ralph instructions into system prompt.

**Progress evaluation**: 
- **Primary**: Agent self-reports completion via text marker `<promise>COMPLETE</promise>` in last assistant message.
- **Guardrails**: `maxIterations` (default 50). Optional periodic reflection (`--reflect-every N`).
- **State**: Persisted to `.ralph/<name>.state.json` (iteration count, status, task file reference).

**Agent/model configuration**: **No model selection**. Agent uses whatever model the user's Pi session uses. Loop parameters (`--max-iterations`, `--items-per-iteration`, `--reflect-every`) are CLI flags at start time.

**Key files**:
- `index.ts` — 824 lines, entire extension
- `SKILL.md` — Agent skill for self-starting loops

---

#### 4.2.2 @pi-unipi/ralph (UniPi Ralph)

**URL**: `https://github.com/Neuron-Mr-White/unipi/tree/main/packages/ralph`
**Source**: `index.ts` (555 lines) + `ralph-loop.ts` (438 lines) + `tools.ts` (170 lines)
**Downloads**: 1,287/mo

**Architecture**: Class-refactored clone of pi-ralph-wiggum. Adds event emission for a dashboard (`RALPH_LOOP_START`, `RALPH_LOOP_END`, `RALPH_ITERATION_DONE`) and info-screen integration.

**Multi-agent spawning**: **None**. Same flat single-agent architecture as wiggum.

**Coordination**: Identical to pi-ralph-wiggum:
1. `ralph_done` tool → `pi.sendUserMessage(buildPrompt(...))` as followUp
2. `agent_end` → checks for completion marker → emits `RALPH_LOOP_END` event on completion
3. `before_agent_start` → injects loop instructions
4. State persists across sessions via `.unipi/ralph/` directory

**Progress evaluation**: Identical to wiggum — `<promise>COMPLETE</promise>` marker + max iterations guard.

**Agent/model configuration**: **No model selection**. Per-invocation flags only (`--max-iterations`, `--items-per-iteration`, `--reflect-every`).

**Key files**:
- `index.ts` — Command handlers, event wiring, info-screen registration
- `ralph-loop.ts` — `RalphLoopManager` class (state, prompts, UI)
- `tools.ts` — `ralph_start` and `ralph_done` tool definitions

---

#### 4.2.3 @kimuson/pi-ralph (d-kimuson)

**URL**: `https://github.com/d-kimuson/pi-ralph`
**Source**: Multi-file TypeScript with tests (20+ files in `src/ralph-loop/`)
**Downloads**: 460/mo

**Architecture**: **Pipeline with external agent subprocesses**. A delivery loop for coding-agent work: static checks → AI review → acceptance criteria checks → completion (PR) → CI/comment follow-up → merge.

**Multi-agent spawning**: Spawns agents as **subprocesses** via `pi --mode json -p`:
```ts
// decisionAgentRunner.service.ts:131-132
const args = [
  '--mode', 'json', '-p',
  '--session-dir', sessionDir,
  '--no-context-files', '--no-skills', '--no-prompt-templates',
  '--no-themes', '--no-extensions',
  '--extension', options.extensionPath,  // review tool
  '--tools', options.tools,             // "read,bash,grep,find,ls,review"
  '--append-system-prompt', promptPath,
  options.buildInitialPrompt(options.request)
];
```

Two agent types:
1. **Review agent** (`kind: 'review'`): Read-only tools, must call `review` tool with `accept`/`reject`
2. **Acceptance criteria agent** (`kind: 'acceptance-criteria'`): Same tools, checks against provided criteria

**Coordination**: **Strict structured verdict**. The `review` tool extension enforces:
- `{ result: "accept", message: string }` — passes
- `{ result: "reject", reason: string }` — fails, with reason
- Result extracted from JSONL event stream (`tool_execution_end` or `message_end` events)
- On missing tool call (agent finishes without calling `review`): resend with `--continue` (up to 3 continuation attempts)

**Progress evaluation**: Multi-phase with **checkpoint caching**:
1. Static checks (`pnpm typecheck`, `pnpm test`, etc.) — must pass all
2. AI review (optional, cached on pass) — structured accept/reject
3. Acceptance criteria (optional, cached on pass) — structured accept/reject
4. Completion (edit-only / draft-pr / pr) — git clean checks, PR creation
5. Autofix (none / ci / comment) — watches CI, handles comments
6. Merge (optional, approved/not) — watches for approval, merges

Passed AI checks are **cached** across retries so the loop restarts from the failing phase.

**Agent/model configuration**:
- **Review/acceptance agents**: Uses whatever Pi invocation is available (heuristic detection: `PI_RALPH_PI_CLI_PATH` env → `process.execPath` → `pi` fallback)
- **No per-agent model selection**: All agents use same Pi binary
- Tool list fixed per agent type: `"read,bash,grep,find,ls,review"`
- System prompt built programmatically per request kind, written to temp file → `--append-system-prompt`

**Key files**:
- `src/ralph-loop/ralphLoop.service.ts` (837 lines) — Core loop state machine, phase orchestration
- `src/ralph-loop/agentCheckRunner.service.ts` — Agent review/acceptance check dispatch
- `src/ralph-loop/decisionAgentRunner.service.ts` — Subprocess spawn, JSONL parsing, continuation loop
- `src/ralph-loop/reviewTool.extension.ts` — `review` tool with accept/reject schema
- `extensions/ralph-loop-commands.ts` — `/ralph-check`, `/ralph-pr`, `/ralph-delegate`, `/ralph-loop` commands

---

#### 4.2.4 pi-ralph (samfp)

**URL**: `https://github.com/samfoy/pi-ralph`
**Source**: `index.ts` (extension) + `lib.ts` (pure logic, 1492-line test suite)
**Downloads**: 51/mo

**Architecture**: **Hat-based multi-agent orchestration**. Presets define sequences of specialized roles ("hats") — Planner, Builder, Reviewer, Committer, etc. Each hat gets a fresh session with its own instructions injected via `before_agent_start`.

**Multi-agent spawning**: **In-process role switching**. No subprocesses or createAgentSession — the same Pi agent session cycles through hats:
1. Agent finishes turn → `agent_end` handler calls `determineNextAction()`
2. Next action may be `SWITCH_HAT` → `buildHatInjection()` constructs new system prompt for the new hat
3. Hat instructions injected via `before_agent_start` hook returning `{ systemPrompt: ... }`
4. Communication between hats via **shared scratchpad file** (writable by all)

**Coordination**: **Event-driven handoff protocol**:
- Hats communicate via XML event tags in messages: `<event topic="..."> ... </event>`
- `detectPublishedEvent()` parses XML events from agent output
- `determineNextAction()` is a **pure function** (no Pi API) that maps current state + detected event → next action
- 7 presets shipped: `feature.yml`, `code-assist.yml`, `spec-driven.yml`, `debug.yml`, `refactor.yml`, `review.yml`, `iterate.yml`

**Progress evaluation**:
- **Completion promise**: `<promise>COMPLETE</promise>` marker (same as wiggum)
- **Guardrails**: max iterations, max runtime (e.g. 28h for refactor), per-hat activation limits, stale cycle detection
- **Stale detection**: `detectStaleCycle()` — tracks last mtime change; after 3 stagnant resumes → blocks
- Custom completion promises per preset (e.g. `DEBUG_COMPLETE` for debug preset)

**Agent/model configuration**:
- **No model selection**. All hats use user's Pi model.
- Presets are YAML files defining hat sequences, instructions, tool lists, guardrails
- Built-in presets in `presets/` directory; custom presets loadable
- Each hat can have `single_task: true` (implement one task per activation) and response templates

**Key files**:
- `lib.ts` — Pure decision logic: `determineNextAction()`, `detectPublishedEvent()`, `containsCompletionPromise()`, `detectStaleCycle()`, `buildHatInjection()`
- `lib.test.ts` — 1492-line vitest suite
- `presets/*.yml` — 7 preset configurations
- `index.ts` — Extension wiring, `/ralph` and `/plan` commands, TUI modals

---

#### 4.2.5 @mikefreno/ralpi

**URL**: `https://github.com/mikefreno/ralpi`
**Source**: Multi-file TypeScript (12+ source files, tests)
**Downloads**: 2,090/mo

**Architecture**: **DAG-based multi-agent executor**. Reads a task file (markdown or YAML), parses into a DAG of tasks with dependencies, spawns child Pi agent sessions via `createAgentSession()` for each task. **Parallel batch execution** (controllable via `maxParallel`).

**Multi-agent spawning**: Uses **`createAgentSession()` from Pi SDK**:
```ts
// utils.ts
const { session } = await createAgentSession({
  cwd,
  systemPrompt: buildTaskPrompt(task),
  model: getModel(provider, modelName),
  tools: FULL_TOOLS,
});
await session.prompt(taskPrompt);
// Wait for agent_end, extract reflection
session.dispose();
```

**Coordination**: **Kahn's algorithm for DAG resolution**:
1. Tasks with satisfied dependencies form batches
2. Batches execute sequentially; tasks within a batch execute in parallel (up to `maxParallel`)
3. Retry with exponential backoff on failure
4. **Round-robin model assignment** across tasks from `execution.models` array with automatic failover
5. Failed tasks with transitive dependencies block their dependents (`getBlockedTasks`)

**Progress evaluation**:
- **Persistent JSON state**: `.ralpi/progress.json` tracks per-task status (pending/in_progress/done/failed/blocked)
- **Reflection extraction**: Regex-based extraction of `## REFLECTION` blocks (SUMMARY, FILES, LEARNINGS, BLOCKERS) from agent output
- **Tool usage counting**: Tracks tool invocation counts per task
- **Multi-PRD support**: Multiple task files tracked via `prds` key
- Resumable across sessions via progress file discovery (`findProgressFile`)

**Agent/model configuration**: **Full per-task model selection**:
- `execution.models`: Array of `{provider, model, thinkingLevel?}` — round-robin assignment
- Automatic failover: if model A fails, retry with model B
- `execution.maxParallel`: Controls concurrent task execution
- `execution.timeout`: Per-task timeout
- `execution.maxRetries`: Retry count with exponential backoff
- Per-task timeouts can be overridden via task file meta blocks

**Key files**:
- `src/executor.ts` — `runTask()`, `executeBatch()`, `executeBatchParallel()`, retry/failover logic
- `src/dag.ts` — Kahn's algorithm, `buildBatches`, `getBlockedTasks`, `detectCycles`
- `src/parser.ts` — Task file parsing (markdown, YAML, dependency notation)
- `src/progress.ts` — `ProgressTracker` class, persistent state
- `src/prompts.ts` — `buildTaskPrompt()` with dependency reflections
- `src/utils.ts` — `runAgentSession()` wrapping `createAgentSession`

---

#### 4.2.6 @lnilluv/pi-ralph-loop

**URL**: `https://github.com/lnilluv/pi-ralph-loop`
**Source**: Multi-file TypeScript (~2,868 lines `index.ts`, 1,806 lines `ralph.ts`, 1,244 lines `runner.ts`)
**Downloads**: 475/mo

**Architecture**: **RPC subprocess per iteration**. Each loop iteration spawns a fresh Pi agent as an RPC subprocess. The orchestrator (extension) controls model, tools, and prompt; captures output; evaluates progress deterministically.

**Multi-agent spawning**: Each iteration spawns `pi --mode rpc` as a child process:
```ts
// runner-rpc.ts
const args = ['--mode', 'rpc', '--no-session', '--no-extensions', '-e', 'ralph'];
const proc = spawn(command, args, { cwd, stdio: ['pipe', 'pipe', 'pipe'] });
// JSONL handshake: set_model + set_thinking_level
// stdin: prompt, stdout: event stream parsed for agent_end
```

**Coordination**: **Durable state machine with signals**:
1. Orchestrator reads `RALPH.md` each iteration (supports live-editing)
2. Runs pre-commands, captures pre-snapshot of workspace
3. Spawns RPC agent with model selection, sends iteration prompt
4. Parses JSONL event stream, extracts last assistant text
5. Runs post-commands, computes diff → progress assessment
6. Checks completion gate (required outputs, acceptance tests, `OPEN_QUESTIONS.md`)
7. Writes `.ralph-runner/status.json`, `iterations.jsonl`, `events.jsonl`, transcripts
8. Stop/cancel via signal files (`stop.flag`, `cancel.flag`)

**Progress evaluation**: **Multi-layered deterministic**:
- **Pre/post snapshot diff**: Detects file changes → progress/no-progress
- **No-progress streak tracking**: Configurable threshold before escalation
- **Completion gate**: Checks for completion promise text + validates required outputs + reruns acceptance commands
- **Runner report**: Static HTML report with status badges, iteration table, event analysis

**Agent/model configuration**: **Per-iteration model selection**:
- Model set via RPC handshake: `set_model` command before prompt delivery
- `set_thinking_level` for thinking mode control
- `--no-extensions -e ralph` — strips all extensions except the ralph-specific one
- Fresh context each iteration (no session reuse)
- Per-RALPH.md configuration: timeout, guardrails, model, shell policy

**Key files**:
- `src/index.ts` (2868 lines) — Command handlers, loop state machine, session replacement
- `src/ralph.ts` (1806 lines) — RALPH.md parsing/frontmatter/types/rendering
- `src/runner.ts` (1244 lines) — Core iteration loop, snapshot assessment, completion gating
- `src/runner-rpc.ts` (617 lines) — RPC subprocess management, JSONL streaming
- `src/runner-state.ts` (1063 lines) — Durable state, signal coordination

---

#### 4.2.7 pi-until-done

**URL**: `https://github.com/srinitude/pi-until-done`
**Source**: Multi-file TypeScript (10 source files)
**Downloads**: 388/mo

**Architecture**: **Single-agent judge-gated loop**. A single executor agent works in a loop; after every turn, progress is scored; when agent calls `until_done_complete`, a **cross-model LLM judge** evaluates the work against acceptance criteria before allowing completion.

**Multi-agent spawning**: **None**. Single agent, no subprocesses. The judge is an **LLM completion call**, not a separate agent session:
```ts
// judge.ts
const result = await complete({
  model: judgeModel,  // can differ from executor model
  system: buildSystemPrompt(goal),
  prompt: buildUserPrompt(goal, evidence),
  temperature: 0,
});
return parseVerdict(result);  // { verdict: 'done' | 'continue', reason }
```

**Coordination**: **Event-driven turn cycle**:
1. `before_agent_start` → injects goal state (north star, phase, budget, tasks) into system prompt
2. Agent works, calls tools (`until_done_task_update`, etc.)
3. `agent_end` → evaluates progress:
   - Budget exhausted? → pause
   - Zero tool calls this turn? → spin guard
   - Code edits made? → auto-run `mise run check` (CI gate)
   - All tasks marked done? → prompt agent to call `until_done_complete`
   - Otherwise → queue continuation as followUp

**Progress evaluation**: **Multi-layered**:
- **Per-turn tool_call counting**: Detects zero-progress turns (spin guard)
- **Turn budget**: `maxTurns` per goal
- **Auto-CI**: Runs verify command after code edits; blocks loop on CI failure
- **LLM judge at completion gate**: Cross-model judge (can use different model than executor) evaluates work against acceptance criteria → `done` or `continue` (with reason)

**Agent/model configuration**:
- **Executor model**: Always user's current Pi model
- **Judge model**: Per-goal configuration:
  - `judgeModel`: Explicit model ID (cross-model) — set via `until_done_set` tool
  - `sameModelJudge`: Uses executor's model for judging (fallback)
  - Session default: `/until-done judge <model>`
- Tools: 8 registered tools (`until_done_set`, `until_done_plan`, `until_done_replan`, `until_done_task_update`, `until_done_progress`, `until_done_complete`, `until_done_block`, `until_done_distill`)

**Key files**:
- `extensions/until-done.ts` (50 LOC) — Composition root
- `extensions/lib/types.ts` — `GoalState`, `JudgeModel`, phases
- `extensions/lib/store.ts` — Session-persistent state via `pi.appendEntry`
- `extensions/lib/tools/complete.ts` — Completion gate with judge dispatch
- `extensions/lib/tools/judge.ts` — LLM judge (cross-model or self)
- `extensions/lib/hooks/agent.ts` — Turn evaluation, budget, spin guard

---

#### 4.2.8 pi-review-loop

**URL**: `https://github.com/nicobailon/pi-review-loop`
**Source**: `index.ts` (~493 lines) + `settings.ts` (~250 lines) + prompt templates
**Downloads**: 386/mo

**Architecture**: **Single-agent self-review loop**. The same Pi agent repeatedly reviews its own work until "no issues found" or max iterations reached. Supports **fresh context mode** where prior review iterations are stripped from history.

**Multi-agent spawning**: **None**. Single agent, event-driven re-fire:
```ts
// agent_end handler
if (matchesExitPattern(text) && !matchesIssuesFixed(text)) {
  exitReviewMode();
} else if (matchesIssuesFixed(text) && iteration < max) {
  pi.sendUserMessage(reviewPrompt, { deliverAs: 'followUp' });  // re-fire
}
```

**Coordination**: **Regex-driven response analysis**:
- `exitPatterns`: Matches "no issues found", "no bugs found", "looks good", "all good"
- `issuesFixedPatterns`: Matches "Fixed N issue(s)", "found and fixed", "ready for another review"
- **Priority**: If response matches BOTH exit and fix patterns → keep looping (fix takes priority)
- **Fresh context mode** (`/review-fresh`): `context` event handler strips prior review iterations from history, injects pass note instructing agent to re-read source documents

**Progress evaluation**:
- **Exit signal**: Agent formats response with exactly one ending: "No issues found." or "Fixed [N] issue(s). Ready for another review."
- **Max iterations**: Default 7, configurable
- **Abort detection**: Empty/missing assistant messages abort
- **Status display**: Footer shows "Review mode (N/7)" with optional "fresh" tag

**Agent/model configuration**: **No model selection**. Agent uses user's Pi model. Config is about loop behavior:
- `settings.json` at `~/.pi/agent/settings.json` under `reviewerLoop` key
- Options: `maxIterations`, `autoTrigger`, `freshContext`, `reviewPrompt`, `triggerPatterns`, `exitPatterns`, `issuesFixedPatterns`
- Pattern config uses `extend` (add to defaults) or `replace` (override entirely)
- Review prompt sources: inline, file path, or template (`template:double-check` / `template:double-check-plan`)

**Key files**:
- `index.ts` — Event handlers, 8 commands, `review_loop` tool
- `settings.ts` — Configuration loader, pattern parsing, prompt resolution
- `prompts/double-check.md` — Code review template
- `prompts/double-check-plan.md` — Plan review template

---

#### 4.2.9 @manojlds/ralphi

**URL**: `https://github.com/manojlds/ralphi`
**Source**: Multi-file TypeScript (18+ source files)
**Downloads**: 151/mo

**Architecture**: **Multi-phase controller/child session architecture**. A controller session manages the loop lifecycle; child sessions execute individual iterations with fresh context. Phases: init → PRD → convert → loop (iterations).

**Multi-agent spawning**: **Child sessions via Pi session tree**:
```ts
// loop-controller.ts
const childSession = await ctx.createChildSession({
  systemPrompt: buildIterationPrompt(story),
  skills: ['ralphi-loop'],
  cwd: ctx.cwd,
});
await childSession.prompt(kickoffMessage);
// On completion: switch back to controller, finalize
```

**Coordination**: **Controller-driven lifecycle**:
1. Controller session manages phases (init, PRD, convert, loop)
2. Each iteration: controller creates child session → child implements one story → returns → controller finalizes iteration
3. PRD stored as `.ralphi/prd.json` with user stories and dependencies
4. `loop-engine.ts` sorts stories by dependency order, selects next `pending` story
5. `loop-controller.ts` manages iteration guards (max iterations, reflection intervals)
6. Child sessions get structured skill instructions via `skills/ralphi-loop/SKILL.md`

**Progress evaluation**:
- **Structured review**: `ralphi_phase_done` tool requires all three fields: `reviewOutcome`, `trajectoryClassification`, `reflection`
- **Trajectory classifier**: Agent self-classifies progress as one of several trajectory types
- **Reflection checkpoints**: Configurable interval (default every N iterations)
- **State persistence**: `.ralphi/runtime-state.json` — survives session restart
- **Deterministic summaries**: `buildDeterministicSummary()` for tree collapse events

**Agent/model configuration**:
- **No model selection**. Both controller and child sessions use user's Pi model.
- **Config**: `.ralphi/config.yaml` — `loop.guidance`, `reviewPasses`, `trajectoryGuard`, `reflectEvery`
- **Skills**: Three skills loaded per phase (`ralphi-init`, `ralphi-prd`, `ralphi-convert`, `ralphi-loop`)
- **Tools**: Two registered tools — `ralphi_phase_done` (structured completion) and `ralphi_ask_user_question` (interactive Q&A)

**Key files**:
- `extensions/ralphi/src/runtime.ts` — `RalphiRuntime` central manager
- `extensions/ralphi/src/loop-controller.ts` — `startLoop`, `runLoopIteration`, child session creation
- `extensions/ralphi/src/loop-engine.ts` — PRD parsing, story sorting, dependency resolution
- `extensions/ralphi/src/loop-finalizer.ts` — Iteration finalization, completion evaluation
- `extensions/ralphi/src/loop-config.ts` — Config parsing, reflection logic
- `extensions/ralphi/src/tools.ts` — `ralphi_phase_done`, `ralphi_ask_user_question`

---

#### 4.2.10 pi-wiggum (jdostal)

**URL**: `https://github.com/jasondostal/pi-wiggum`
**Source**: `extensions/goal-guard.ts` (339 lines) + `prompts/judge.md` + `prompts/goal.md`
**Downloads**: 130/mo

**Architecture**: **Two-role, external evaluator**. A single worker agent pursues a goal; on each stop, an **independent evaluator model** (different from worker) judges progress via git diff analysis and returns a verdict.

**Multi-agent spawning**: **Evaluator runs as subprocess** (`pi -p -nt --model $WIGGUM_JUDGE_MODEL`). Not a full agent session — a single `-p` completion:
```ts
// goal-guard.ts
const args = ['-p', '-nt', '--model', judgeModel];
const result = execSync(`pi ${args.join(' ')}`, {
  input: buildJudgePrompt(goal, acceptanceCriteria, gitDiff),
  cwd,
});
const verdict = JSON.parse(result.stdout);  // { state, rationale, evidence, next_directive }
```

**Coordination**: **File-system state machine**:
1. Worker agent writes `.wiggum/goal.md` with goal + concrete acceptance criteria
2. Worker pursues goal directly using tools (no planning phase, no subagents)
3. `agent_end` hook triggers evaluator
4. Evaluator reads `.wiggum/goal.md` + `git diff HEAD` + untracked new files
5. Evaluator returns JSON verdict: DONE | CONTINUE | REDIRECT | BLOCKED
6. Extension injects verdict + directive as next agent message via `pi.sendUserMessage()`
7. `.wiggum/.escalate` file signals hard block requiring user intervention

**Progress evaluation**: **Independent evaluator + mechanical backstops**:
- **Primary**: Evaluator model judges git diff against rubric (acceptance criteria). Each criterion must be demonstrable. When in doubt → CONTINUE over DONE.
- **Hard cap**: 30 iterations per goal
- **Mechanical fallback**: If judge fails/returns garbage → checks mtime change; after 3 stagnant resumes → escalates via `.wiggum/.escalate`
- **Evaluator prompt** (`prompts/judge.md`): Rubric-driven with four verdicts, evidence requirements, strict JSON output schema

**Agent/model configuration**: **Evaluator model independently configurable**:
| Env var | Default | Purpose |
|---|---|---|
| `WIGGUM_JUDGE_MODEL` | `xiaomi/mimo-v2.5-pro` | Evaluator model (must differ from worker) |
| `WIGGUM_PI_BIN` | `pi` | Pi binary for subprocess |
| `WIGGUM_JUDGE_PROMPT` | (auto-resolved) | Override path to judge.md |

Worker model = whatever user's Pi session uses. No control from extension.

**Key files**:
- `extensions/goal-guard.ts` (339 lines) — Core loop, evaluator spawn, verdict dispatch
- `prompts/judge.md` — Evaluator rubric (4 verdicts, JSON schema)
- `prompts/goal.md` — Worker prompt (write goal.md, work directly, no planning)
- `ARCHITECTURE.md` — Design doc

**Note**: v0.4.0 deliberately removed earlier v0.1-0.3 planning/execution split and worker/reviewer subagents. Current design is minimalist: one worker, one evaluator subprocess.

---

#### 4.2.11 @rahulmutt/pi-ralph

**URL**: `https://github.com/rahulmutt/pi-ralph`
**Source**: Single file `extensions/index.ts` (507 lines) + `tests/e2e.test.ts` (914 lines)
**Downloads**: 126/mo

**Architecture**: **Fresh session per iteration via `ctx.newSession()`**. Runs a prompt file repeatedly across fresh agent sessions. Session replacement with global Symbol state for coordination.

**Multi-agent spawning**: **Session replacement**, not spawning:
```ts
// index.ts
ctx.newSession({
  parentSession: originalSessionId,
  withSession: (newCtx) => {
    newCtx.sendUserMessage(promptContent);  // runs in fresh session
  },
});
// Wait for agent_end resolved via Symbol.for('@rahulmutt/pi-ralph.state')
```

**Coordination**: **Symbol.for-global state + promise resolver**:
1. Global state shared via `Symbol.for('@rahulmutt/pi-ralph.state')` on `globalThis`
2. `ctx.newSession()` creates replacement session; `withSession` callback fires on new session's context
3. Promise resolver set before `newSession()`; resolved by `agent_end` handler on new session
4. Extension re-loads on every `newSession()` call (Pi ≥0.65 module isolation); Symbol survives module cache isolation

**Progress evaluation**:
- **Accumulated summaries**: Last assistant message text per iteration
- **Output files**: `.ralph/RALPH.md` (overwritten each iteration), `.ralph/<date>/RALPH-<ts>.md` (appended per invocation), per-iteration `.ralph/<date>/RALPH-<ts>-iter-<NNN>.jsonl` transcripts
- **Iteration cap**: Default 3, max 1000

**Agent/model configuration**: **No model selection**. Prompt text reused verbatim each iteration. No config file — args only: `--iterations N`, prompt file path.

**Key files**:
- `extensions/index.ts` (507 lines) — Entire extension
- `tests/e2e.test.ts` (914 lines) — Faux (mock) LLM provider tests
- `AGENTS.md` — Coordination pattern documentation

---

#### 4.2.12 @entelligentsia/pi-ralph

**URL**: `https://github.com/Entelligentsia/pi-ralph`
**Source**: Multi-file TypeScript (10 files)
**Downloads**: 63/mo

**Architecture**: **Generator→Critique→Judge chain, single model**. All three agents are functions calling the same LLM with different system prompts. No subprocesses, no tool use, no parallelism.

**Multi-agent spawning**: **Function calls, not processes**:
```ts
// orchestrator.ts
const genResult = await generator.execute(goal, priorResult, criticism);
const critique = await critiqueAgent.execute(goal, genResult);
const verdict = await judgeAgent.execute(goal, genResult, critique);
if (!verdict.done && iteration < maxLoops) {
  // Loop: Generator gets result + criticism as context
  continue;
}
```

**Coordination**: **Sequential chain-of-thought refinement**:
1. Feasibility check via LLM before loop starts
2. Dynamic prompt generation: prompts for all three agents are LLM-generated per goal
3. Generator → Critique → Judge, strictly sequential
4. Judge emits `{done: boolean, reason: string}` → orchestrator decides continue/stop
5. On `done: false`, Generator gets previous result + criticism on next iteration

**Progress evaluation**: **Judge verdict only**:
- After each iteration, Judge agent parses `{done, reason}` JSON from LLM output
- Loop exits when `done: true` or iterations reach `maxLoops` (default 3)
- No continuous scoring, decaying threshold, or adaptive stopping

**Agent/model configuration**: **No model-per-agent selection**:
- All agents use `ctx.model` (user's current Pi model) via `@earendil-works/pi-ai` `complete()` API
- Prompts dynamically generated each run (feasibility-check + prompt-generation phase tailors to goal domain)
- Fallback prompts exist if dynamic generation fails
- Pure text completions — no tool use by any agent

**Key files**:
- `agents/orchestrator.ts` — Core coordinator: feasibility, prompt generation, loop
- `agents/generator.ts` — Generator: calls LLM with goal + prior context
- `agents/critique.ts` — Critique: calls LLM with goal + result
- `agents/judge.ts` — Judge: calls LLM, parses JSON verdict
- `llm.ts` — `oneshotLLM()` and `parseJsonResponse()` wrappers
- `prompts.ts` — Static fallback system prompts

**Key limitation**: Not a multi-agent system in the distributed sense. Pure sequential chain-of-thought refinement. No tool use, spawned sub-tasks, parallel evaluation, or agent routing by capability.

---

#### 4.2.13 @pi-agents/orchid

**URL**: No public repo; npm only (`https://www.npmjs.com/package/@pi-agents/orchid`)
**Source**: Multi-file TypeScript (831-line ralph, 579-line subagents)
**Downloads**: 413/mo

**Architecture**: **Unified orchestration package** combining batch orchestrator, Ralph loop, and subagent dispatch. The Ralph component is a **flat single-agent** loop identical to pi-ralph-wiggum.

**Multi-agent spawning**: **None in Ralph component** (same flat wiggum clone). **Separate subagent module** provides sync/async subagent dispatch, chain/parallel modes, async job tracking — but this is distinct from the Ralph loop.

**Coordination**: Ralph component: `ralph_start`/`ralph_done` tools, event-driven re-fire, completion marker detection, reflection intervals — identical to pi-ralph-wiggum.

**Progress evaluation**: File-based `.ralph/` directory loops, iteration counting, completion marker `<promise>COMPLETE</promise>`.

**Agent/model configuration**: **No model selection** in Ralph component. Subagent module uses Pi's built-in agent dispatch.

**Key files**:
- `src/ralph/index.ts` (831 lines) — File-based ralph loop (wiggum clone)
- `src/subagents/extension/index.ts` (579 lines) — Subagent dispatch (chain/parallel)

---

#### 4.2.14 pi-ralph-loop (mazli)

**URL**: No public repo; npm only
**Source**: `extensions/index.ts` (155 lines) + `extensions/loop-controller.ts` (456 lines)
**Downloads**: 81/mo

**Architecture**: **Simple session checkpoint reset**. `/loop` command starts a loop; each iteration resets the session to a checkpoint, effectively giving the agent "fresh eyes" each time.

**Multi-agent spawning**: **None**. Session checkpoint/restore mechanism. Not a multi-agent system — session tree navigation for context reset.

**Coordination**: Checkpoint-based context reset:
1. `/loop start <goal> [--iterations=N]` creates a checkpoint
2. Agent works → calls `loop_done` or iteration completes
3. Extension navigates session tree to restore checkpoint context
4. Re-fires with same goal prompt, fresh context

**Progress evaluation**:
- Iteration counting with hard cap (default 10)
- Graceful stop via `/loop` while running

**Agent/model configuration**: **No model selection**. Simple extension.

---

#### 4.2.15 ralph-loop-pi / pi-hooks (ralph-loop component) (prateekmedia)

**URL**: `https://www.npmjs.com/package/ralph-loop-pi` (standalone), `https://www.npmjs.com/package/pi-hooks` (collection)
**Source**: Single file `ralph-loop.ts` (2025 lines) — identical code in both packages
**Downloads**: 77/mo (ralph-loop-pi), 189/mo (pi-hooks)

**Architecture**: **Condition-gated subagent looping with UI controls**. Agent calls `ralph_loop` tool with condition, subagents executed repeatedly until condition met.

**Multi-agent spawning**: **Subagent dispatch** via Pi's agent infrastructure. Tool registration for `ralph_loop` with condition evaluation.

**Coordination**: Condition-evaluated loop with steering:
- Agent calls `ralph_loop` tool with condition description
- Subagent(s) spawned to execute work
- Condition evaluated after each subagent completes
- Steering/follow-up controls for loop direction
- UI mode with status display

**Progress evaluation**: Condition-gated: loop continues until condition is met (agent-evaluated). Iteration counting, status display.

**Agent/model configuration**: Uses Pi's standard subagent dispatch. No custom model selection documented.

**Key files**:
- `ralph-loop.ts` (2025 lines) — Subagent spawn, condition evaluation, chain mode

---

#### 4.2.16 pi-pai / @artale/pi-pai

**URL**: `https://github.com/arosstale/pi-pai`
**Downloads**: 349/mo (pi-pai), 99/mo (@artale/pi-pai)

**Note**: PAI is primarily a Personal AI Infrastructure framework. Ralph iteration is mentioned as one feature among many (ISC splitting, capability tracking, agent personas, sentiment tracking, self-evolution). Not a dedicated Ralph loop implementation — excluded from detailed analysis.

---

## 5. Conclusions

### 5.1 Multi-Agent Spawning Mechanisms

| Mechanism | Implementations | Description |
|---|---|---|
| **No multi-agent** (flat single) | pi-ralph-wiggum, Unipi/ralph, pi-review-loop, orchid/ralph | Same agent re-fired via followUp |
| **External subprocess** (`pi --mode json -p`) | d-kimuson/pi-ralph | Specialized review agents with custom tool extensions |
| **External subprocess** (`pi --mode rpc`) | lnilluv/pi-ralph-loop | RPC agent with model selection, JSONL streaming |
| **External subprocess** (`pi -p -nt`) | pi-wiggum | Single-shot evaluator subprocess (no agent session) |
| **Session replacement** (`ctx.newSession()`) | rahulmutt/pi-ralph | Fresh Pi session per iteration, parent-child with global state |
| **createAgentSession()** (SDK) | ralpi | Full agent sessions with model selection, parallel execution |
| **Child sessions** (session tree) | ralphi | Controller creates child sessions for each iteration |
| **In-process role switching** | pi-ralph (samfp) | Same session cycles through hat roles via system prompt injection |
| **LLM completion chain** | entelligentsia/pi-ralph, pi-until-done | LLM `complete()` calls, not full agent sessions |

### 5.2 Coordination Patterns

| Pattern | Implementations | Mechanism |
|---|---|---|
| **Follow-up message** | wiggum, unipi, pi-review-loop | `pi.sendUserMessage(msg, {deliverAs: 'followUp'})` re-fires agent |
| **Structured verdict tool** | d-kimuson | `review` tool with `accept`/`reject` schema, JSONL event parsing |
| **File-system state machine** | pi-wiggum, rahulmutt | `.wiggum/goal.md` or `.ralph/` directory → file existence = state |
| **DAG resolution** | ralpi | Kahn's algorithm → batches → parallel/sequential execution |
| **Durable state + signals** | lnilluv, ralphi, pi-until-done | JSON state files + signal files (stop.flag) + event logs |
| **Event-driven handoff** | samfp | XML event tags → `determineNextAction()` pure function |
| **Symbol.for global state** | rahulmutt | Cross-module-isolation state sharing for session replacement |
| **Sequential function chain** | entelligentsia | Async function calls passing text outputs |

### 5.3 Progress Evaluation

| Approach | Implementations | How |
|---|---|---|
| **Completion marker** | wiggum, unipi, samfp, lnilluv | `<promise>COMPLETE</promise>` in agent text output |
| **Structured verdict** | d-kimuson, pi-wiggum, entelligentsia | `accept`/`reject` or `DONE`/`CONTINUE`/`BLOCKED` from judge |
| **LLM judge** | pi-until-done, pi-wiggum | Independent model evaluation against acceptance criteria |
| **Deterministic diff** | lnilluv | Pre/post snapshot → file change detection → progress/no-progress |
| **Regex pattern match** | pi-review-loop | Exit/fix patterns in agent response text |
| **Checklist completion** | wiggum, unipi, ralphi | Markdown checklist items marked `[x]` in task file |
| **Reflection extraction** | ralpi, ralphi | Structured reflection blocks parsed from agent output |
| **Auto-CI gate** | pi-until-done, d-kimuson | Verify command run after code edits; failure blocks |
| **Iteration cap** | ALL | Hard limit on iterations (3–1000) |
| **Stagnation detection** | pi-wiggum, lnilluv, samfp | mtime staleness, no-progress streaks |

### 5.4 Agent/Model Configuration Capabilities

| Capability | Implementations | Details |
|---|---|---|
| **Per-role model selection** | ralpi | `execution.models[]` array with round-robin assignment |
| **Evaluator separate model** | pi-wiggum | `WIGGUM_JUDGE_MODEL` env var (must differ from worker) |
| **Cross-model judge** | pi-until-done | Per-goal `judgeModel` or `sameModelJudge` fallback |
| **RPC model selection** | lnilluv | `set_model` handshake before prompt delivery |
| **No model selection** | wiggum, unipi, samfp, rahulmutt, entelligentsia, pi-review-loop, ralphi | User's Pi model used for all agents |
| **Model auto-detection** | d-kimuson | `PI_RALPH_PI_CLI_PATH` env → `process.execPath` → `pi` fallback |
| **Per-task tool restriction** | d-kimuson, samfp, lnilluv | Review agents get read-only tools; implementation agents get full tools |

### 5.5 Most Novel Approaches

1. **ralpi** — Only implementation with true multi-agent parallelism via `createAgentSession()` + Kahn's DAG resolution. Round-robin model assignment with failover.

2. **lnilluv/pi-ralph-loop** — Only implementation with pre/post snapshot diff-based deterministic progress assessment. RPC subprocess with model selection handshake.

3. **pi-wiggum** — Only implementation where the evaluator is a different model by design (not optional). Evaluator prompt is rubric-driven with 4 distinct verdicts.

4. **samfp/pi-ralph** — Only implementation using hat-based role modeling with YAML preset configuration. Pure `determineNextAction()` function makes loop logic fully testable (1492-line test suite).

5. **d-kimuson/pi-ralph** — Only implementation with a composable pipeline (static → AI → completion → follow-up → merge). Agent decisions cached across retries.

6. **pi-until-done** — Only implementation with both per-turn tool_call counting and auto-CI gates. Cross-model judge at completion.

---

### 5.6 Implications for Extension Design

| Decision | Choice | Rationale |
|---|---|---|
| Agent dispatch mechanism | `createAgentSession()` | Public SDK, in-process, full control. Same API used by OMP's own `task` tool |
| Agent configuration | OMP agent discovery (`discoverAgents` + convention-based names) | Reuses existing agent definition format (`.md` + frontmatter). No new config format. Users create `ralph-reviewer`, `ralph-implementer`, `ralph-verifier` agent files with custom models/prompts/tools |
| Model selection | Per-agent via `AgentDefinition.model` | Model string from agent frontmatter passed as `modelPattern` to `createAgentSession()`. Each role independently configurable |
| Tool restriction | Per-agent via `AgentDefinition.tools` | Reviewer gets read-only tools; implementer gets full tools; verifier gets test-execution tools — all defined in the agent file, not hardcoded in extension |
| Progress evaluation | Structured verdict (d-kimuson precedent) | `accept`/`reject` with reason → deterministic, testable. Implemented as a custom tool registered by the extension for each agent pass |
| Coordination | Sequential per-task + per-task iteration loop | Review → implement → verify per task. Max 3 implement→verify cycles |
| Bead management | `bd` CLI (`bd claim`, `bd close`, `bd block`) | Reuses existing bead infrastructure. Each task corresponds to one bead |
| Iteration limit | 3 per task | Stricter than all existing implementations (30-50) — appropriate for focused per-task loops with independent verification |
