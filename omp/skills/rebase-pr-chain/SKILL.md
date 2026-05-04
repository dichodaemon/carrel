---
name: rebase-pr-chain
description: >
  Rebase all branches in a GitHub PR chain onto their updated parents,
  bottom-up from trunk to the current branch. Discovers the chain via
  gh pr view baseRefName. Usage: /rebase-pr-chain [starting-branch]
---

# Rebase PR Chain

Discover the full PR chain for a branch and rebase each link bottom-up so every
branch is current with its parent.

## Step 1: Pre-flight checks

**a) Verify clean working tree:**

```bash
git status --porcelain
```

If output is non-empty, stop and tell the user to commit or stash changes first.
Do not proceed with a dirty worktree.

**b) Record the starting branch:**

If the user provided a branch name argument, use that. Otherwise:

```bash
git rev-parse --abbrev-ref HEAD
```

This is the **top** of the chain. Save it -- you will return here at the end.

## Step 2: Discover the chain

Walk from the starting branch down to trunk by querying each PR's base branch.

Initialize an empty list `chain = []`.
Set `current = <starting-branch>`.

**Loop:**

```bash
gh pr view <current> --json baseRefName,headRefName,state
```

- If the command fails (no PR exists for `current`), stop and report:
  `"No PR found for branch '<current>'. Cannot discover the chain."`
- If `state` is `MERGED` or `CLOSED`, note that this branch is already
  merged/closed -- it does not need rebasing. Record it as a skip but continue
  walking: set `current = baseRefName` and repeat.
- If `state` is `OPEN`, prepend `{head: headRefName, base: baseRefName}` to
  `chain` and set `current = baseRefName`.
- If `baseRefName` is `master` (or the repo's trunk), stop. The walk is
  complete.
- If `baseRefName` equals `current` (self-referential), stop and report an
  error.

After the loop, `chain` is ordered bottom-up: `chain[0]` is the branch closest
to trunk, `chain[-1]` is the starting branch.

**Show the discovered chain to the user before proceeding:**

```
Discovered PR chain (bottom-up rebase order):

  master
    <- branch-a  (#12345)
    <- branch-b  (#12346)
    <- branch-c  (#12347)  <- you are here

Will fetch and rebase 3 branches. Proceed?
```

Wait for user confirmation before continuing.

## Step 3: Fetch

```bash
git fetch origin
```

One fetch before any rebases. This ensures all remote refs are current.

## Step 4: Rebase bottom-up

For each entry in `chain`, in order from index 0 (closest to trunk) to the end:

**a) Checkout the branch:**

```bash
git checkout <chain[i].head>
```

**b) Rebase onto its parent:**

For `chain[0]`, the parent is `master` (or whatever trunk the walk terminated
at). For `chain[i]` where `i > 0`, the parent is `chain[i-1].head`.

```bash
git rebase <parent>
```

**c) Handle conflicts:**

If the rebase exits with a non-zero status, stop and report:

```
Rebase conflict on branch '<chain[i].head>' while rebasing onto '<parent>'.

Resolve conflicts, then run:
  git add <resolved-files>
  git rebase --continue

After resolving, re-invoke /rebase-pr-chain <chain[i].head> to continue
from where you left off. The remaining branches are:
  <list remaining chain entries>
```

Do not attempt to resolve conflicts automatically. Do not skip commits. Stop
and let the user handle it.

**d) Report progress** after each successful rebase:

```
Rebased <chain[i].head> onto <parent>  [i+1/total]
```

## Step 5: Return to starting branch

After all rebases succeed:

```bash
git checkout <starting-branch>
```

## Step 6: Summary

Report the result:

```
PR chain rebased successfully (N branches):

  master
    <- branch-a  (rebased)
    <- branch-b  (rebased)
    <- branch-c  (rebased)  <- you are here

To push all rebased branches:
  git push --force-with-lease origin branch-a branch-b branch-c
```

Do not push automatically. The user decides when to force-push.
