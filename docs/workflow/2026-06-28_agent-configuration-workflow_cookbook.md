# Agent Configuration Workflow Cookbook

Miscellaneous findings, tips, and workarounds discovered during development.

---

## 1. Adding a new rule

When you need to add a new agent rule (convention, constraint, or prohibition) to the carrel configuration, write the markdown source file and register it with a single workflow chain.

Write the rule file at the conventional path for rules:

```
omp/rules/my-rule.md
```

All rules live in `omp/rules/` with `.md` extension and the rule name as the filename (lowercase, underscores).

```bash
# Register the rule in the carrel registry and deploy it to OMP
carrel config add rule my-rule --source=carrel-omp --file=omp/rules/my-rule.md
carrel slot sync-all
carrel run
```

The three-command chain is: `add` registers the entry in the registry with a stable UUID, `slot sync-all` wires it to the consumer's output slots, and `run` composes, deploys, and executes OMP with the updated configuration.

The `--source=carrel-omp` flag specifies the project configuration source. The `--file` flag reads the entry content from the source file — no need to paste content inline with `--content`.

---

## 2. Adding a new skill

Skills follow the same workflow as rules but use a different conventional path. Each skill lives in its own directory with a `SKILL.md` file.

Write the skill file:

```
omp/skills/my-skill/SKILL.md
```

The directory name is the skill name (lowercase, underscores). The entry point is always `SKILL.md` inside that directory.

```bash
# Register the skill in the carrel registry and deploy it to OMP
carrel config add skill my-skill --source=carrel-omp --file=omp/skills/my-skill/SKILL.md
carrel slot sync-all
carrel run
```

The workflow is identical to rules: `add` registers the entry, `slot sync-all` links it to the consumer's slots, `run` deploys and executes. Use the type `skill` instead of `rule`, and point `--file` at the `SKILL.md` within the skill directory.

---

## 3. Updating an existing entry

When an entry's source file changes — whether you edit a rule, a skill, or any other configuration type — re-register it with the same `add` command. The command is idempotent: it recognizes the existing entry by its type and name and reuses the stable UUID rather than creating a duplicate.

Edit the source file in place:

```bash
# After editing omp/rules/my-rule.md or omp/skills/my-skill/SKILL.md
carrel config add rule my-rule --source=carrel-omp --file=omp/rules/my-rule.md
carrel slot sync-all
carrel run
```

The same three-command chain (`add` → `slot sync-all` → `run`) applies regardless of whether the entry is new or updated. There is no separate edit command — `add` with `--file=` is the single registration path for both creation and updates.

To preview what `slot sync-all` would change without modifying anything, use `--dry-run`:

```bash
carrel slot sync-all --dry-run
```

This shows which unwired entries would be linked to slots without applying any changes. Useful for verifying that your newly added or updated entry will be deployed before running the full chain.

---
