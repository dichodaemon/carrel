---
description: "Format, structure, naming, and best practices for hook entries."
globs:
  - 'omp/hooks/*.sh'
  - 'omp/hooks/*.bash'
  - 'omp/hooks/*.py'
---

# Hook Entry Type

Hooks are executable scripts that fire at specific points in the OMP session lifecycle. They enable automation such as environment setup, cleanup, notifications, and custom validation that runs outside the agent's normal interaction loop.

## Format

- **File extension:** `.sh` (bash, preferred), `.bash`, `.py` (Python), or any executable script
- **Location:** `omp/hooks/<name>.<ext>`
- **Requirements:**
  - File must be executable (`chmod +x`)
  - Exit code 0 indicates success; non-zero aborts the session or logs a warning (hook-type dependent)
  - Scripts receive the session's environment; avoid assuming a specific shell

### Hook Types & Trigger Points

| Hook Type | Trigger | Abort on Failure |
|---|---|---|
| `pre-session` | Before agent initialization | Yes — session fails to start |
| `post-session` | After session teardown | No — logged as warning |
| `pre-command` | Before each user command | No — logged as warning |
| `post-command` | After each user command | No — logged as warning |
| `pre-deploy` | Before `carrel run` deploys | Yes — deployment aborts |
| `post-deploy` | After `carrel run` deploys | No — logged as warning |

## Naming Conventions

- **kebab-case:** `pre-session-setup`, `post-session-cleanup`, `check-disk-space`
- **Lifecycle-prefixed (convention):** `pre-` for before hooks, `post-` for after hooks. Not enforced, but strongly recommended.
- **Descriptive of action:** `validate-environment`, `notify-slack`, `warm-cache`
- **Avoid generic names:** `hook.sh`, `script.sh`, `run.sh`

## Content Best Practices

- **Idempotent.** Hooks may run multiple times (session restarts, retries). Make them safe to re-run: check if the work is already done before doing it.
- **Fast.** Hooks block the lifecycle event they're attached to. A `pre-session` hook that takes 30 seconds adds 30 seconds to every session start. Target <5 seconds for pre-session hooks, <1 second for per-command hooks.
- **Silent on success.** Output on success clutters the session log. Print only on error, or use a `--verbose` flag for debugging.
- **Clear error messages.** On failure, print to stderr what went wrong and how to fix it. The agent sees hook errors and may need to act on them.
- **Environment-agnostic.** Don't assume the script runs from a specific directory. Use absolute paths or resolve relative to `$OMP_SESSION_DIR`.
- **Use `set -euo pipefail`** in bash hooks to fail fast on errors, unset variables, and pipe failures.
- **Log to a known location.** Write logs to `$OMP_LOG_DIR/hooks/` rather than stdout.
- **Avoid side effects that survive sessions.** Hooks run in the session's environment. If you set environment variables, they persist for the session. If you modify files, they remain after the hook exits.

## Deployment Behavior

1. **Registration:** `carrel config add hook <name> --file=omp/hooks/<name>.sh`
2. **Slot wiring:** `carrel slot sync-all`
3. **Deployment:** `carrel run` deploys to `.omp/hooks/<name>.sh` with execute permissions
4. **Execution:** OMP discovers hooks in `.omp/hooks/` at session startup and registers them with the lifecycle engine

Hooks are deployed with their execute bit preserved. If a hook is not executable after deployment, OMP logs a warning and skips it.

## Validation

Use the `validate-hook` skill to check:
- File is executable
- Script has a valid shebang line (`#!/usr/bin/env bash`, `#!/bin/bash`, `#!/usr/bin/env python3`)
- For bash hooks: passes `shellcheck` without errors
- For Python hooks: syntax check passes (`python3 -m py_compile`)
- No dangerous patterns (unbounded `rm -rf`, infinite loops, `curl | bash`)
- Hook type can be determined from naming convention or metadata

## Example

```bash
#!/usr/bin/env bash
# pre-session: Validate that required tools are installed
set -euo pipefail

missing=()

command -v gh >/dev/null 2>&1 || missing+=("gh (GitHub CLI)")
command -v bun >/dev/null 2>&1  || missing+=("bun")
command -v carrel >/dev/null 2>&1 || missing+=("carrel")

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "ERROR: Missing required tools:" >&2
  for tool in "${missing[@]}"; do
    echo "  - $tool" >&2
  done
  echo "Install them before starting an OMP session." >&2
  exit 1
fi
```
