---
description: "Format, structure, naming, and best practices for command entries."
globs:
  - 'omp/commands/*.yaml'
  - 'omp/commands/*.yml'
  - 'omp/commands/*.json'
---

# Command Entry Type

Command entries define slash-commands — structured invocation points that agents and users can trigger during OMP sessions. Commands provide a discoverable, self-documenting interface for common operations like creating resources, running workflows, or querying state.

## Format

- **File extension:** `.yaml` or `.yml` (preferred) or `.json`
- **Location:** `omp/commands/<name>.yaml`
- **Structure:** A single top-level object defining the command's interface and behavior

### Required Fields

| Field | Type | Description |
|---|---|---|
| `name` | string | Command identifier, kebab-case. Invoked as `/<name>` |
| `description` | string | One-sentence summary shown in command listings |
| `handler` | string | What executes the command: `skill`, `prompt`, `agent`, `script`, or `builtin` |

### Handler-Specific Fields

**Skill handler** (`handler: skill`):
| Field | Type | Description |
|---|---|---|
| `skill` | string | Name of the skill to activate |

**Prompt handler** (`handler: prompt`):
| Field | Type | Description |
|---|---|---|
| `prompt` | string | Name of the prompt to resolve and execute |

**Agent handler** (`handler: agent`):
| Field | Type | Description |
|---|---|---|
| `agent` | string | Name of the agent to spawn |
| `subagent` | boolean | Run as subagent; default true |

**Script handler** (`handler: script`):
| Field | Type | Description |
|---|---|---|
| `script` | string | Path to executable script |
| `timeout` | number | Max execution time in seconds |

### Optional Fields

| Field | Type | Description |
|---|---|---|
| `arguments` | list | Positional or named arguments the command accepts |
| `short_help` | string | One-line usage string: `/<name> [args]` |
| `long_help` | string | Multi-line markdown help shown with `/<name> --help` |
| `examples` | list | Usage examples with expected output |
| `category` | string | Grouping category for command listings: `config`, `workflow`, `query`, `utility` |

### Argument Definition

Each item in `arguments`:
```yaml
arguments:
  - name: path
    description: File or directory path
    required: true
    type: string
  - name: force
    description: Skip confirmation prompts
    required: false
    type: boolean
    default: false
```

## Naming Conventions

- **kebab-case:** `create-plan`, `audit-document`, `find-session`
- **Verb-first:** name describes the action the command performs
- **Consistent with handler:** a command that activates `create-plan` skill SHOULD be named `create-plan`
- **No leading slash in the name:** `name: create-plan` is invoked as `/create-plan`
- **Avoid type prefixes:** `cmd-create-plan` is redundant; the type already identifies it

## Content Best Practices

- **Self-documenting.** The `description` and `short_help` are the first things users see. Make them count: "Create a comprehensive implementation plan from companion documents" not "Create plan."
- **Discoverable arguments.** Each argument needs a clear description and type. The agent uses these to construct valid invocations.
- **Examples are critical.** Provide 2–3 `examples` showing different invocation patterns. Examples are the most effective documentation.
- **Handler delegation.** Commands SHOULD delegate to skills or prompts rather than embedding logic directly. This keeps commands thin and reusable.
- **`long_help` for complex commands.** If a command has more than 3 arguments or non-obvious behavior, provide `long_help` with usage patterns, edge cases, and related commands.
- **Category grouping.** Assign a `category` so commands appear organized in listings. Common categories: `config`, `workflow`, `query`, `utility`, `git`.
- **Avoid command bloat.** Not every skill needs a command. Create a command only when direct invocation is a common user workflow.

## Deployment Behavior

1. **Registration:** `carrel config add command <name> --file=omp/commands/<name>.yaml`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys to `.omp/commands/<name>.yaml`
4. **Discovery:** OMP registers commands at session startup. They appear in the slash-command palette and in `/help` output.

Commands are resolved at invocation time — the handler (skill, prompt, agent) is loaded on demand, not at registration.

## Validation

Use the `validate-command` skill to check:
- File is valid YAML/JSON
- All required fields are present and non-empty
- `handler` is one of: `skill`, `prompt`, `agent`, `script`, `builtin`
- Handler-specific required fields are present
- Referenced `skill`, `prompt`, or `agent` exists in the registry
- Referenced `script` is resolvable
- All `arguments` have valid `type` (string, boolean, number, path)
- Required arguments appear before optional arguments
- No duplicate command with the same name exists

## Example

```yaml
name: create-plan
description: >
  Create a comprehensive implementation plan from companion documents
  (spec, arch-design, brief, issue) and codebase investigation
short_help: /create-plan <companion-path> [output-dir]
handler: skill
skill: create-plan
category: workflow
arguments:
  - name: companion-path
    description: Path to the companion document (spec, arch-design, or brief)
    required: true
    type: path
  - name: output-dir
    description: Directory to write the plan; defaults to docs/
    required: false
    type: path
    default: docs/
examples:
  - "/create-plan docs/specs/auth.md"
  - "/create-plan docs/arch-design.md docs/plans/"
  - "/create-plan docs/brief.md --output-dir plans/"
```
