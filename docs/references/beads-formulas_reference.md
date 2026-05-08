---
title: Beads Formula System
date: 2026-05-01
author: Dizan Vasquez
---

# Understanding the Beads Formula System

## Why It Exists, What Problem It Solves, and How to Think About It

---

## Part 1: The Problem That Made Formulas Necessary

To understand formulas, you need to understand the journey that created them. Yegge didn't sit down and design a workflow templating system — he was driven to it by repeated, painful failure.

### The Markdown Plan Disaster

The original approach to managing agent work was markdown plan files. An agent would start a task, declare "I'm going to break this into six phases," and create a `.md` file. This sounds reasonable. It is, in practice, a slow catastrophe.

Here's what actually happens:

1. The agent works through phases 1 and 2. Multiple context compactions happen along the way.
2. By phase 3, the agent has mostly forgotten where it came from. It wakes up, reads the plan file, and announces: "Oh wow, this is a big project. I'm going to break it into **five** phases."
3. It begins working on phase 1 of 5 of phase 3 of 6. It calls it "phase 1."
4. Eventually it declares the whole system done at roughly phase 3 of 5 of phase 3 of 6, with six hundred orphaned plan files in the `plans/` directory.

The core problem is structural: **markdown plans are text, not structured data.** They can't be queried. Dependencies exist only as prose ("blocked on auth fix"). The agent has to re-parse and re-interpret the work graph every single time it wants to know what to do next. And after enough compaction cycles, it can't.

### Issues Fixed the Memory Problem — But Introduced a New One

Beads (the issue tracker) solved the amnesia problem. Instead of markdown, work lives in a queryable, git-backed database. `bd ready` tells an agent exactly what unblocked work exists. The agent no longer has to reconstruct the work graph from prose — it just queries it.

But once you have agents working well on individual issues, you immediately hit the next problem: **repeatable multi-step workflows**.

Some work always follows the same shape. Every feature needs design, implementation, review, and merge. Every release needs version bump, changelog, tests, build, tag, publish. You don't want to recreate that graph of linked issues by hand every time. You want a template you can instantiate.

That's the gap formulas were built to fill.

---

## Part 2: The MEOW Stack — How Formulas Fit In

Yegge describes the evolution of his system as the **Molecular Expression of Work (MEOW)** stack. Each layer builds on the one below it:

```
Beads (issues)          ← atomic work units, the primitive
    ↓
Epics                   ← issues with child issues; hierarchical plans
    ↓
Molecules               ← workflows: chains of issues with explicit dependency graphs
    ↓
Protomolecules          ← molecule templates: pre-built graphs ready to instantiate
    ↓
Formulas (TOML/JSON)    ← source layer: human-editable templates that get "cooked" into protomolecules
```

Formulas are the **source code** for protomolecules. Just as you don't ship C source to production but compile it first, you don't pour a formula directly — you cook it into a protomolecule, then instantiate it into a live molecule.

The reason a separate "cook" step exists is macro expansion. Simple templates (just variable substitution) didn't need it. But once you want loops, gates, aspects, and formula composition — one formula wrapping or extending another — you need a pre-processing phase. That's `bd cook`.

---

## Part 3: The Core Concepts, Explained by Their Purpose

### Formulas: The Source Layer

A formula is a TOML or JSON file that describes a workflow as a set of steps with dependencies. It's the thing you write by hand, or distill from an existing epic, or share with teammates.

```toml
formula = "feature-workflow"
version = 1
type = "workflow"

[vars.feature_name]
description = "Name of the feature"
required = true

[[steps]]
id = "design"
title = "Design {{feature_name}}"
type = "human"

[[steps]]
id = "implement"
title = "Implement {{feature_name}}"
needs = ["design"]

[[steps]]
id = "review"
title = "Code review"
needs = ["implement"]
type = "human"

[[steps]]
id = "merge"
title = "Merge to main"
needs = ["review"]
```

The key insight: **steps with `needs` create dependencies, not just sequence.** Steps without `needs` are parallel. Steps that `needs` multiple prior steps create join points. This means formulas can express real dependency graphs, not just linear TODO lists — which is exactly what issue-based planning needs.

### Cooking: From Source to Template

`bd cook <formula>` expands macros, applies aspects, and resolves composition rules. The output is a protomolecule: a fully-resolved graph of template issues, ready to be stamped into real issues.

Think of it like:

```
formula.toml  →  [bd cook]  →  protomolecule  →  [bd pour]  →  live molecule (actual issues in the DB)
```

You rarely interact with protomolecules directly. The cook → pour pipeline handles it.

### Pouring: Instantiation

`bd pour <formula-name> --var key=value` is the everyday command. It cooks (if needed) and instantiates the formula into real issues in your beads database. Those issues have hash IDs, dependencies, assignees — they're first-class beads, queryable with `bd ready`, claimable by agents.

```bash
# Instantiate a release workflow
bd pour release --var version=2.1.0

# The live molecule appears in the DB; bd ready will surface
# the first unblocked step automatically
bd ready
```

### Wisps vs Mols: Ephemeral vs Persistent Workflows

Once instantiated, a workflow exists as either a **mol** (persistent, synced to git) or a **wisp** (ephemeral, not synced).

| | Mol | Wisp |
|---|---|---|
| Storage | `.beads/` | `.beads-wisp/` |
| Synced to git | Yes | No |
| Survives agent restart | Yes | No |
| Use case | Real project work | Orchestration patrols, throwaway runs |

The key use case for wisps is high-velocity orchestration. In Gas Town, every patrol run (Refinery, Witness, Deacon) creates a wisp molecule for each loop iteration — they get the transactional step-by-step tracking, but without polluting git history with orchestration noise.

---

## Part 4: Composition — The Real Power

The `extends`, `compose`, and aspect `advice` features are what make formulas genuinely powerful, and they're the least-documented part of the system. The intent, based on what Yegge describes in the Gas Town post, is this:

**You should be able to wrap any workflow with cross-cutting concerns without modifying the original formula.**

### The "Rule of Five" Example

Yegge gives the clearest concrete example of formula composition in practice. Jeffrey Emanuel's "Rule of Five" is the observation that having an LLM review something five times with different focus areas produces superior outputs. Yegge implemented this as a formula.

The intent:

```
Take any existing workflow  →  cook it with the Rule of Five formula
                            →  each step gets wrapped with 4 additional review steps
                            →  the agent executes more review-intensive work automatically
```

This is an **aspect**: a formula of type `"aspect"` that targets steps matching a pattern and injects steps before or after them. It doesn't care what the base workflow is — it just attaches to matching steps during cooking.

```toml
# Conceptual structure of an aspect formula
formula = "rule-of-five"
type = "aspect"

[[advice]]
target = "*.implement"    # match any "implement" step in any formula

[advice.before]
id = "review-focus-1-{step.id}"
title = "Architecture review: {step.title}"

[advice.after]
id = "review-focus-2-{step.id}"
title = "Security review: {step.title}"
# ... and so on
```

### Why This Matters

This is the answer to "why do formulas need a cook step?" Because aspects and composition can't be resolved at the formula level — they need a compilation step that sees all participating formulas together and assembles the final graph.

The practical workflow Yegge describes is:

```bash
# Define your base work
bd pour feature-workflow --var feature_name="payment refactor"

# Or, cook the base workflow with a quality aspect first
bd cook feature-workflow --aspect rule-of-five | bd pour --var feature_name="payment refactor"
```

---

## Part 5: The Durability Guarantee

This is what formulas ultimately exist to provide, and it's worth stating clearly.

Before formulas, a 20-step release process would get abandoned mid-way when the agent hit its context limit, got confused, or took a shortcut. There was no durable record of "we are on step 14 of 20."

With a molecule instantiated from a formula:

- **The agent is a persistent identity in git** (a Bead, not a session)
- **The hook is persistent** (a Bead pointing to the current molecule)
- **The molecule is persistent** (a chain of Beads in git)

So it doesn't matter if Claude Code crashes or compacts. The next session starts, finds its hook, finds the molecule, finds the first unclaimed ready step, and picks up exactly where the previous session left off. Agents don't manage their own TODO list — they walk a pre-built, pre-verified dependency graph one step at a time.

Yegge calls this **Nondeterministic Idempotence**: the path through the workflow is nondeterministic (the agent decides how to execute each step), but the outcome — the workflow completing — is guaranteed as long as you keep throwing agents at it.

---

## Part 6: Practical Summary — When to Use What

| Situation | What to use |
|---|---|
| One-off multi-step work | Create issues + deps manually with `bd create` / `bd dep` |
| Repeatable standard workflow | Write a formula, `bd pour` each time you need it |
| Learned from past work | `bd mol distill <epic-id>` to extract a formula from an existing epic |
| Cross-cutting quality gates | Write an aspect formula, compose it at cook time |
| Orchestration / patrol loops | `bd wisp` with a formula — ephemeral, no git noise |
| Long-running project work | `bd pour` into a mol — persistent, survives restarts |

### The `distill` Learning Path

The fastest way to understand formula structure is to build a workflow manually, then distill it:

```bash
# 1. Do some work the normal way
bd create "Design auth system" -p 1 -t task
bd create "Implement auth" -p 1 --deps design-id
bd create "Review auth" -p 1 --deps implement-id
bd create "Ship auth" -p 1 --deps review-id

# 2. Distill it into a formula (--dry-run first to preview)
bd mol distill <epic-id> auth-workflow --dry-run

# 3. The output shows you exactly what the TOML would look like
# for this workflow shape — with variables parameterized
```

This is the missing worked example the docs don't provide: let Beads show you what your actual workflow looks like as a formula, then generalize from there.

---

## Where the Docs Don't Go (Yet)

As of early 2026, the official docs cover the basic formula format well but don't have worked examples of:

- `extends` (formula inheritance)
- `compose` (formula composition at pour time)
- Aspect `advice` targets with wildcards
- The full `bd cook` pipeline with multiple participating formulas

The Mol Mall (a planned marketplace for formulas) hasn't launched yet. Until it does, the best sources for real formula examples are the `.beads/formulas/` directories in the Gas Town and Beads repos themselves, and the `bd mol distill` command applied to your own epics.
