---
description: "Format, structure, naming, and best practices for extension entries."
globs:
  - 'omp/extensions/*/'
---

# Extension Entry Type

Extensions are OMP plugins that extend the platform with new capabilities — custom tool types, UI components, session middlewares, or integration layers. Unlike tools (which are configured instances of built-in types), extensions define new types or modify platform behavior.

## Format

- **Location:** `omp/extensions/<name>/` — one directory per extension
- **Entry point:** `omp/extensions/<name>/extension.yaml` (or `extension.json`) — required manifest
- **Implementation:** One or more files in the extension directory

### Required Manifest Fields

| Field | Type | Description |
|---|---|---|
| `name` | string | Extension identifier, kebab-case; must match directory name |
| `version` | string | Semver version: `major.minor.patch` |
| `description` | string | One-sentence summary of what the extension provides |
| `type` | string | Extension category: `tool-provider`, `middleware`, `ui-component`, `hook-provider`, `command-provider`, `skill-provider` |

### Type-Specific Manifest Fields

| Extension Type | Required Fields | Implementation |
|---|---|---|
| `tool-provider` | `tool_type` (string) | Defines a new tool type; implementation handles tool lifecycle |
| `middleware` | `hook_point` (string), `order` (number) | Intercepts session events at the specified hook point |
| `ui-component` | `component_type` (string), `entry` (string) | Registers a UI component in the OMP interface |
| `hook-provider` | `hooks` (list of hook names) | Provides reusable hook implementations |
| `command-provider` | `commands` (list of command names) | Bundles a set of related commands |
| `skill-provider` | `skills` (list of skill names) | Bundles a set of related skills |

### Optional Manifest Fields

| Field | Type | Description |
|---|---|---|
| `author` | string | Extension author or maintainer |
| `license` | string | SPDX license identifier (e.g., `MIT`, `Apache-2.0`) |
| `dependencies` | object | Required extensions or tool types: `{ extension-name: version-constraint }` |
| `config_schema` | object | JSON Schema for extension-specific configuration |
| `min_omp_version` | string | Minimum OMP version required |

## Naming Conventions

- **kebab-case:** `github-integration`, `slack-notifier`, `custom-editor-tools`
- **Feature-descriptive:** name reflects what the extension provides, not how it's implemented
- **Consistent with directory name:** `omp/extensions/github-integration/extension.yaml` → `name: github-integration`
- **Avoid `omp-` prefix:** extensions live in the OMP namespace already
- **Avoid generic names:** `tools`, `utils`, `helpers`

## Content Best Practices

- **Single responsibility.** One extension = one capability. Don't bundle a tool provider, a middleware, and a UI component in one extension — split them.
- **Version semantically.** Follow semver strictly. Breaking changes → major version bump. New features → minor. Fixes → patch.
- **Declare dependencies.** If your extension depends on another extension or a specific tool type, declare it in `dependencies`. This prevents silent failures.
- **Provide a config schema.** If your extension accepts configuration, define `config_schema` so carrel can validate it.
- **Document the directory layout.** Include a `README.md` in the extension directory explaining the file structure, entry points, and any build steps.
- **Test across consumers.** Extensions run in the context of a consumer. Test with multiple consumers to ensure no consumer-specific assumptions.
- **Minimal `min_omp_version`.** Only set it if you actually use a version-gated feature. Over-constraining blocks adoption.
- **Keep implementation files in the extension directory.** Don't reference files outside `omp/extensions/<name>/`.

## Deployment Behavior

1. **Registration:** `carrel config add extension <name> --file=omp/extensions/<name>/extension.yaml`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys the entire extension directory to `.omp/extensions/<name>/`
4. **Loading:** OMP discovers extensions at session startup. Extensions are loaded in dependency order (dependencies before dependents). Each extension's initialization hooks are called during session setup.

Extensions are loaded once at session startup. Changes require a session restart. Extension initialization failures are logged and may block session startup depending on the extension type.

## Validation

Use the `validate-extension` skill to check:
- Manifest (`extension.yaml` or `extension.json`) is valid and at the correct path
- All required manifest fields are present and non-empty
- `version` is valid semver
- `type` is a recognized extension type
- Type-specific required fields are present
- `dependencies` reference existing extensions with resolvable version constraints
- `config_schema` (if present) is valid JSON Schema
- Directory contains the entry point and all referenced implementation files
- No duplicate extension with the same name exists

## Example

```yaml
# omp/extensions/github-integration/extension.yaml
name: github-integration
version: 1.2.0
description: >
  GitHub API integration for OMP — provides tool types for issue tracking,
  PR management, and code search via the GitHub REST and GraphQL APIs.
type: tool-provider
tool_type: github
author: Carrel Team
license: MIT
dependencies:
  http-client: ">=1.0.0"
config_schema:
  type: object
  properties:
    token_env:
      type: string
      description: Environment variable containing the GitHub token
      default: GITHUB_TOKEN
    api_url:
      type: string
      description: GitHub API base URL (for GitHub Enterprise)
      default: https://api.github.com
    max_retries:
      type: integer
      minimum: 1
      maximum: 10
      default: 3
  required: []
min_omp_version: "2.0.0"
```
