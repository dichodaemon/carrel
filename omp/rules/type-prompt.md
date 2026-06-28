---
description: "Format, structure, naming, and best practices for prompt entries."
globs:
  - 'omp/prompts/*.md'
---

# Prompt Entry Type

Prompts are reusable templates that encode interaction patterns for common agent tasks. They provide structured starting points for interactions like code review, planning, debugging, or documentation — reducing repetition and ensuring consistency.

## Format

- **File extension:** `.md` (Markdown)
- **Location:** `omp/prompts/<name>.md`
- **Frontmatter fields:**
  - `description` (required) — one-line summary of the prompt's purpose and usage
  - `arguments` (optional) — list of template variables the prompt accepts

Template variables use the `{{variable}}` syntax within the prompt body.

### Template Variables

| Variable | Description |
|---|---|
| `{{file}}` | Current file being worked on |
| `{{selection}}` | Selected text or code region |
| `{{repo}}` | Repository name |
| `{{branch}}` | Current git branch |
| `{{language}}` | Programming language or file type |

Custom arguments are defined in the frontmatter `arguments` field.

## Naming Conventions

- **kebab-case:** `review-changes`, `debug-error`, `explain-code`, `write-tests`
- **Verb-first:** name describes the action the prompt facilitates
- **Task-specific:** `generate-commit-message` not `git-help`
- **Avoid prefixes like `prompt-`:** the type already identifies it

## Content Best Practices

- **Start with role.** Define the agent's persona for this interaction: "You are a senior TypeScript developer reviewing a pull request."
- **Structure with sections.** Use markdown headings to organize the prompt body. Common sections: `## Context`, `## Instructions`, `## Output Format`, `## Constraints`.
- **Be explicit about output format.** If the prompt expects structured output (JSON, a specific template, a checklist), specify it in an `## Output Format` section.
- **Use template variables sparingly.** Only add variables that meaningfully change the prompt's behavior. Over-parameterization makes prompts harder to use.
- **Test with variations.** Run the prompt with different inputs to ensure template variables resolve correctly and edge cases are handled.
- **Include examples.** A `## Example` section showing expected input and output helps users understand the prompt's intent.
- **Constraints are rules-lite.** Prompts can include constraints within their scope, but system-wide rules should be separate rule entries. Don't re-encode rules in prompts.

## Deployment Behavior

1. **Registration:** `carrel config add prompt <name> --file=omp/prompts/<name>.md`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys to `.omp/prompts/<name>.md`
4. **Invocation:** Agents discover prompts from the deployed directory. Users invoke them via slash-command or natural language: `/review-changes` or "review my changes with the review-changes prompt."

Prompts are templates — they are resolved with session context at invocation time, not at deployment time.

## Validation

Use the `validate-prompt` skill to check:
- Frontmatter is valid YAML with required `description`
- Template variables use valid `{{...}}` syntax
- All `arguments` in frontmatter appear in the body (and vice versa)
- Body is non-empty and contains actionable instructions
- No duplicate prompt with the same name exists

## Example

```markdown
---
description: Review staged changes for correctness, security, and style
arguments:
  - focus: area to focus on (security, performance, style, or all)
---

# Review Changes

You are a senior {{language}} developer reviewing a set of changes.

## Context

- Repository: {{repo}}
- Branch: {{branch}}

## Instructions

1. Review the changes for correctness, edge cases, and potential regressions
2. Check for security vulnerabilities (OWASP Top 10)
3. Evaluate code clarity and maintainability
4. Focus area: {{focus}}

## Output Format

For each issue found, report:
- **Severity:** CRITICAL | HIGH | MEDIUM | LOW
- **File:** path and line range
- **Issue:** one-line description
- **Suggestion:** concrete fix
```
