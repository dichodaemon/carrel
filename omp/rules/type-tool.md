---
description: >
  Format, structure, naming, and best practices for tool entries.
  Tools define external integrations available to agents during sessions.
  Use `carrel config add tool <name> --file=omp/tools/<name>.yaml` to register.
  Validate with `validate-tool` skill before registering.
globs:
  - 'omp/tools/*.yaml'
  - 'omp/tools/*.yml'
  - 'omp/tools/*.json'
---

# Tool Entry Type

Tool entries define external integrations that agents can invoke during sessions. Tools extend the agent's capabilities beyond the built-in tool inventory — connecting to APIs, databases, external services, or custom scripts.

## Format

- **File extension:** `.yaml` or `.yml` (preferred) or `.json`
- **Location:** `omp/tools/<name>.yaml`
- **Structure:** A single top-level object defining the tool's interface and implementation

### Required Fields

| Field | Type | Description |
|---|---|---|
| `name` | string | Tool identifier, kebab-case |
| `description` | string | One-sentence summary of what the tool does; injected into the agent's tool inventory |
| `type` | string | Integration type: `mcp`, `http`, `script`, `bash` |

### Tool-Type-Specific Fields

**MCP tools** (`type: mcp`):
| Field | Type | Description |
|---|---|---|
| `command` | string | MCP server command (e.g., `npx`, `uvx`, `bun`) |
| `args` | list | Command-line arguments |
| `env` | object | Environment variables for the server process |

**HTTP tools** (`type: http`):
| Field | Type | Description |
|---|---|---|
| `base_url` | string | Base URL for API calls |
| `endpoints` | list | Endpoint definitions with method, path, and schema |
| `auth` | object | Authentication config (bearer token, API key, oauth) |

**Script tools** (`type: script`):
| Field | Type | Description |
|---|---|---|
| `script` | string | Path to executable script |
| `interpreter` | string | Runtime: `python3`, `bun`, `node`, `bash` |
| `timeout` | number | Maximum execution time in seconds |

**Bash tools** (`type: bash`):
| Field | Type | Description |
|---|---|---|
| `command` | string | Shell command template |
| `timeout` | number | Maximum execution time in seconds |
| `sandbox` | boolean | Run in isolated environment; default true |

### Optional Fields (all types)

| Field | Type | Description |
|---|---|---|
| `parameters` | object | JSON Schema describing the tool's input parameters |
| `requires_approval` | boolean | Tool invocation needs user confirmation; default false |
| `rate_limit` | object | `{ max: int, window: string }` e.g., `{ max: 10, window: "1m" }` |

## Naming Conventions

- **kebab-case:** `github-api`, `slack-notify`, `database-query`
- **Service-prefixed:** `github-`, `slack-`, `jira-` for service integrations
- **Action-descriptive:** `deploy-preview`, `run-migration`, `fetch-logs`
- **Avoid generic names:** `api`, `service`, `tool1`

## Content Best Practices

- **Descriptive `description` field.** The description is shown to the agent when it considers which tool to use. Make it actionable: "Query the PostgreSQL database with read-only SQL" not "Database tool."
- **Minimal parameter surface.** Define only the parameters the agent actually needs. Over-specifying parameters increases the chance of incorrect invocation.
- **Validate inputs server-side.** The parameter schema is a hint for the agent, not a security boundary. Validate all inputs in the tool implementation.
- **Handle errors gracefully.** Return structured error messages the agent can understand and act on. Don't dump stack traces into the agent's context.
- **Rate-limit appropriately.** Set `rate_limit` for tools that call external APIs with quotas or costs.
- **`requires_approval` for destructive operations.** Any tool that modifies data, deploys code, or incurs cost SHOULD require user approval.
- **Test the tool end-to-end.** Verify the tool works from within an OMP session before registering it. Test both success and error paths.

## Deployment Behavior

1. **Registration:** `carrel config add tool <name> --file=omp/tools/<name>.yaml`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys to `.omp/tools/<name>.yaml`
4. **Loading:** OMP discovers tools at session startup and registers them in the agent's tool inventory. The tool's `description` is injected into the system prompt so the agent knows when to use it.

Tools are initialized lazily — the tool's process or connection is established on first invocation, not at session startup.

## Validation

Use the `validate-tool` skill to check:
- File is valid YAML/JSON
- All required fields are present and non-empty
- `type` is one of: `mcp`, `http`, `script`, `bash`
- Type-specific required fields are present
- `parameters` (if present) is valid JSON Schema
- `rate_limit` (if present) has valid `max` (positive integer) and `window` (duration string)
- Referenced scripts or commands are resolvable
- No duplicate tool with the same name exists

## Example

```yaml
name: github-api
description: >
  Query the GitHub API for issues, PRs, and repository metadata.
  Supports search, create, and update operations.
type: http
base_url: https://api.github.com
auth:
  type: bearer
  token_env: GITHUB_TOKEN
endpoints:
  - method: GET
    path: /repos/{owner}/{repo}/issues
    description: List repository issues
  - method: POST
    path: /repos/{owner}/{repo}/issues
    description: Create a new issue
parameters:
  type: object
  properties:
    endpoint:
      type: string
      enum: [list-issues, create-issue, get-pr, search-code]
    owner:
      type: string
    repo:
      type: string
  required: [endpoint, owner, repo]
requires_approval: true
rate_limit:
  max: 30
  window: 1m
```
