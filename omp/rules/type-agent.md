---
description: "Format, structure, naming, and best practices for agent entries."
globs:
  - 'omp/agents/*.yaml'
  - 'omp/agents/*.yml'
  - 'omp/agents/*.json'
---

# Agent Entry Type

Agent entries configure the personality, model backend, and provider settings for OMP agent sessions. Each agent is a named configuration that can be selected via the session launcher or configured as the default for a consumer.

## Format

- **File extension:** `.yaml` or `.yml` (preferred) or `.json`
- **Location:** `omp/agents/<name>.yaml`
- **Structure:** A single top-level object with agent configuration keys

### Required Fields

| Field | Type | Description |
|---|---|---|
| `name` | string | Agent identifier, kebab-case |
| `description` | string | One-sentence summary of the agent's role and personality |
| `model` | string | Model identifier (e.g., `claude-sonnet-4-20250514`, `gpt-4o`) |
| `provider` | string | Provider key (e.g., `anthropic`, `openai`, `deepseek`) |
| `system_prompt` | string | System-level instructions defining the agent's persona and behavior |

### Optional Fields

| Field | Type | Description |
|---|---|---|
| `temperature` | number | Sampling temperature (0.0–1.0); default provider-specific |
| `max_tokens` | number | Token budget ceiling; default provider-specific |
| `tools` | list | Allowed tool names; omit for full tool access |
| `skills` | list | Skills always injected for this agent; omit for discovery-based loading |
| `rules` | list | Rules always injected for this agent |
| `context_files` | list | Context files loaded at session start |
| `streaming` | boolean | Enable response streaming; default true |

## Naming Conventions

- **kebab-case:** `code-reviewer`, `architect-advisor`, `default`
- **Descriptive of role:** `security-auditor`, `documentation-writer`, `test-engineer`
- **Reserved names:** `default` is the fallback agent used when no agent is explicitly selected

## Content Best Practices

- **Personality in `system_prompt`.** The system prompt defines the agent's voice, expertise, and behavioral defaults. Be specific: "You are a senior security reviewer who prioritizes OWASP Top 10 concerns" beats "You review code for security."
- **Model selection matches workload.** Provision expensive models (Sonnet, Opus) for complex reasoning tasks; use fast models (Haiku, Flash) for mechanical operations. Match the agent's purpose.
- **Tool restriction.** Limit `tools` for specialized agents that should only perform a subset of operations. A documentation-writer shouldn't run shell commands; a code-reviewer shouldn't write files.
- **Skills and rules.** Pre-load skills that are always relevant (e.g., a `security-auditor` always needs `audit-document`). Don't pre-load skills that are scenario-dependent — let discovery handle those.
- **Temperature.** Use low temperature (0.0–0.3) for deterministic tasks (code generation, test writing). Use higher (0.5–0.8) for creative tasks (naming, prose, design exploration).
- **Test the agent.** After creating an agent entry, launch a session with it and verify the persona, model, and tools are as expected.

## Deployment Behavior

1. **Registration:** `carrel config add agent <name> --file=omp/agents/<name>.yaml`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys to `.omp/agents/<name>.yaml`
4. **Loading:** OMP reads agent configurations at session startup. The selected agent's `system_prompt`, `tools`, `skills`, and `rules` are composed into the session.

Agent entries can be scoped:
- **Universal:** `omp/agents/` — available to all consumers
- **Target-specific:** `<target>/.carrel/agents/` — only that consumer

## Validation

Use the `validate-agent` skill to check:
- File is valid YAML/JSON
- All required fields are present and non-empty
- `model` is a recognized model identifier for the specified `provider`
- `temperature` is in range [0.0, 1.0]
- `max_tokens` is a positive integer
- Referenced `tools`, `skills`, and `rules` exist in the registry
- No duplicate agent with the same name

## Example

```yaml
name: code-reviewer
description: Senior code reviewer focused on security, correctness, and maintainability
model: claude-sonnet-4-20250514
provider: anthropic
system_prompt: >
  You are a senior code reviewer. You prioritize:
  1. Security vulnerabilities (OWASP Top 10)
  2. Logical correctness and edge cases
  3. Maintainability and clarity
  Be thorough but concise. Flag issues with severity: CRITICAL, HIGH, MEDIUM, LOW.
temperature: 0.2
tools:
  - read
  - grep
  - glob
  - ast_grep
skills:
  - audit-document
rules:
  - carrel-configuration
```
