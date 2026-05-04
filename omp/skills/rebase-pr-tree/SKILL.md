---
name: rebase-pr-tree
description: >
  Rebase an entire dependency tree of branches starting from a root branch.
  Discovers children via GitHub PRs and local git ancestry, then rebases in
  topological order (parents before children). Handles trees, not just chains.
  Usage: /rebase-pr-tree <root-branch> [--local-prefix=<prefix>] [--rebase-root]
---

# Rebase PR Tree

Discover the full dependency tree rooted at a branch and rebase every link
top-down so every branch is current with its parent.

Unlike `rebase-pr-chain` (which walks leaf-to-root), this skill starts at the
**root** and discovers all descendants -- including forks where one parent has
multiple children.

## Input

- `<root-branch>` — required. The branch at the top of the tree.
- `--local-prefix=<prefix>` — optional. A branch name prefix (e.g., `dizan/`)
  to scan local branches for ancestry relationships that don't have PRs. If
  omitted, local-only discovery is skipped.
- `--rebase-root` — optional. If set, rebase the root branch itself onto its
  PR base (or master/main) before processing children.

## Step 1: Pre-flight checks

**a) Verify clean working tree:**

```bash
git status --porcelain
```

If output is non-empty, stop and tell the user to commit or stash changes first.
Do not proceed with a dirty worktree.

**b) Record the current branch:**

```bash
git rev-parse --abbrev-ref HEAD
```

Save this -- you will return here at the end.

**c) Verify the root branch exists locally:**

```bash
git rev-parse --verify <root-branch>
```

If it does not exist, try `git fetch origin <root-branch>` and
`git checkout <root-branch>`. If that also fails, stop and report.

## Step 2: Discover tree via GitHub PRs

Build the tree top-down using BFS. Each node is
`{branch, pr_number, pr_state, parent, children[]}`.

Initialize:
- `root = {branch: <root-branch>, pr_number: null, pr_state: null, parent: null, children: []}`
- `queue = [root]`
- `visited = {<root-branch>: root}`

**BFS loop:**

While `queue` is not empty, dequeue `node`:

```bash
gh pr list --repo <repo> --base <node.branch> --state open \
  --json number,headRefName,baseRefName --limit 50
```

For each result:
- If `headRefName` is already in `visited`, skip (prevents diamonds from
  creating duplicates).
- Otherwise create a child node:
  `{branch: headRefName, pr_number: number, pr_state: "OPEN", parent: node, children: []}`
- Append child to `node.children` and to `queue`.
- Add to `visited`.

Also check closed/merged PRs that might have open descendants:

```bash
gh pr list --repo <repo> --base <node.branch> --state all \
  --json number,headRefName,baseRefName,state --limit 50
```

For results with `state` = `MERGED` or `CLOSED`:
- If `headRefName` is already in `visited`, skip.
- Create a child node with `pr_state: "MERGED"` or `"CLOSED"`. These will be
  skipped during rebase but their children are still traversed.
- Append to `node.children`, `queue`, and `visited`.

**Detect the repo** from the git remote:

```bash
gh repo view --json nameWithOwner -q '.nameWithOwner'
```

## Step 3: Discover local orphans (optional)

Only if `--local-prefix` was provided.

```bash
git branch --list '<prefix>*' --format='%(refname:short)'
```

For each local branch not already in `visited`:

Check ancestry against every node in the tree, deepest first (reverse BFS
order). For each tree node:

```bash
git merge-base --is-ancestor <tree-node.branch> <local-branch>
```

If true, this local branch descends from that tree node. Attach it as a child
of the **deepest** matching tree node (first match in reverse-BFS order). Mark
it as `{branch: <local-branch>, pr_number: null, pr_state: "LOCAL", parent: <deepest-match>, children: []}`.

Add it to `visited` and re-enqueue it so its own children (via GitHub PRs) are
also discovered.

**Important:** Exclude branches where the relationship is reversed (the local
branch is an ancestor of the root). Also exclude the root branch itself and
`master`/`main`.

## Step 4: Display tree and confirm

Show the full tree using indentation. Example:

```
Discovered dependency tree from 'dizan/secondary_adp_integration':

  dizan/secondary_adp_integration  (root)
  ├── dizan/fallback_full_architecture  #77707 OPEN
  └── dizan/fallback-behavior-rename  #78714 OPEN
      └── dizan/fallback-latency-benchmark-tool  #78803 OPEN

4 branches total. 3 will be rebased (1 root, 3 descendants).
Merged/closed branches are traversed but skipped during rebase.
Local-only branches (no PR) are included but flagged.

Proceed?
```

Wait for user confirmation before continuing.

## Step 5: Optionally rebase root

If `--rebase-root` was set:

```bash
gh pr view <root-branch> --json baseRefName -q '.baseRefName'
```

If the root has a PR with a base:

```bash
git checkout <root-branch>
git rebase <base-of-root>
```

If no PR exists for root, rebase onto `master` (or `main`).

Report result. On conflict, stop with recovery instructions (see Step 7).

## Step 6: Fetch

```bash
git fetch origin
```

One fetch before any rebases. This ensures all remote refs are current.

## Step 7: Rebase in topological order

Process the tree in BFS order (parents before children). Skip the root unless
`--rebase-root` was handled in Step 5.

For each node in BFS order:

**a) Skip if merged/closed:**

If `pr_state` is `MERGED` or `CLOSED`, report:

```
Skipping <branch> (PR #<number> is <state>)  [i/total]
```

Continue to next node. Children of merged/closed nodes are still processed --
they are rebased onto the merged/closed node's branch position.

**b) Checkout the branch:**

```bash
git checkout <node.branch>
```

**c) Rebase onto parent:**

```bash
git rebase <node.parent.branch>
```

**d) Handle conflicts:**

If the rebase exits with a non-zero status, stop and report:

```
Rebase conflict on branch '<node.branch>' while rebasing onto '<parent.branch>'.

Resolve conflicts, then run:
  git add <resolved-files>
  git rebase --continue

After resolving, re-invoke:
  /rebase-pr-tree <root-branch> [same flags]

The following branches still need rebasing:
  <list remaining nodes in BFS order>
```

Do not attempt to resolve conflicts automatically. Do not skip commits. Stop
and let the user handle it.

**e) Report progress** after each successful rebase:

```
Rebased <node.branch> onto <parent.branch>  [i/total]
```

## Step 8: Return to original branch

After all rebases succeed:

```bash
git checkout <original-branch>
```

## Step 9: Summary

Report the result:

```
PR tree rebased successfully (N branches):

  dizan/secondary_adp_integration  (root)
  ├── dizan/fallback_full_architecture  (rebased)
  └── dizan/fallback-behavior-rename  (rebased)
      └── dizan/fallback-latency-benchmark-tool  (rebased)

To push all rebased branches:
  git push --force-with-lease origin branch-a branch-b branch-c
```

Do not push automatically. The user decides when to force-push.
