# Carrel Policy

## OMP Configuration Source of Truth

The `.omp/` directory in the target workspace is **deployed** by carrel's `carrel run` command. Never edit `.omp/` directly in a target repo -- changes will be overwritten by the next `carrel run` deployment.

Before creating, modifying, or deleting any OMP configuration (skills, rules, commands, extensions, agents, tools, hooks, prompts, instructions, or system prompt files), use `carrel <type> add|rm|edit` commands. Configuration lives in registered sources managed by the carrel registry.

For detailed carrel mechanics (sources, slots, CRUD, deployment), see `/workspace/carrel/.omp/rules/carrel-configuration.md`.

---

