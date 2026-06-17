---
name: find-session
description: >
  Search OMP session transcripts by date and keyword content across all
  workspaces. Returns matching sessions with relevant excerpts.
  Usage: /find-session <query> [--date=<date-or-range>] [--title]
---

# Find Session

Search past OMP session transcripts to locate a specific conversation.
Scans all workspace session directories under `~/.omp/agent/sessions/`,
matching by keyword and optionally filtering by date or title.

## Trigger

User says "find session", "search sessions", "look at past sessions",
"find a conversation where...", or invokes `/find-session`.

## Arguments

- **query** (required): Keywords to search for (e.g., "velocity floor",
  "lane follow sampling").
- **--date** (optional): Date filter. Accepts ISO date (`2026-06-13`),
  relative expressions (`4 days ago`, `last week`, `yesterday`), or a
  range (`2026-06-10..2026-06-13`). When omitted, searches all sessions.
- **--title** (optional flag): Search only session titles, not full
  message content. Faster for large histories.

## Step 1: Discover session files

List all JSONL files across every workspace session directory:

```
~/.omp/agent/sessions/*/  ->  *.jsonl
```

Each filename encodes its timestamp: `<ISO-timestamp>_<uuid>.jsonl`.
Parse the timestamp prefix to apply date filtering.

If `--date` is provided, resolve it to a concrete date or range and
keep only files whose timestamp prefix falls within bounds.

Sort candidates by timestamp, most recent first.

## Step 2: Index sessions

Read line 0 (the `type: "session"` header) of each candidate file to
extract:

- `title` (may be null or absent)
- `timestamp`
- `id`
- `cwd` (derives the workspace name)

If `--title` flag is set, filter sessions whose title matches the query
(case-insensitive substring). Present results and skip to Step 5.

## Step 3: Search message content

For each candidate session, scan every JSONL line where:

- `type` is `"message"`
- `message.role` is `"user"` or `"assistant"`
- `message.content` contains an entry with `type: "text"`

Check each text block for case-insensitive matches against ALL query
keywords. A session matches when at least one message contains all
keywords.

Count matching messages per session. Rank sessions by match count
(descending), then by recency.

Cap the search at 50 session files to avoid unbounded scan times. If
more candidates exist, prefer recent files and inform the user that
older sessions were not searched.

## Step 4: Present results

Show a numbered list of matching sessions:

```
Found N sessions matching "<query>":

  1. [2026-06-13 22:05] core-stack
     "Debug ego pull-over failure in simulation"
     12 matching messages

  2. [2026-06-12 17:47] core-stack
     "Velocity sampling changes"
     3 matching messages
```

Include workspace name (extracted from the directory name, stripping
the `--workspace-` prefix/suffix) and session title.

If no matches, say so and suggest broadening the query or adjusting
the date range.

## Step 5: Drill into a session

When the user selects a session (by number or by asking for details),
extract the relevant conversation:

- Find all messages containing the query keywords.
- For each match, include the message and its immediate neighbors
  (one message before, one after) for context.
- Show role (`user` / `assistant`) and the text content, truncated to
  500 characters per message with `... [truncated]` when clipped.
- Exclude `toolResult` and `toolCall` content unless the user
  explicitly asks for tool details.

Present as a readable conversation transcript with clear delimiters
between messages.

## Rules

- **Read-only.** Never modify, delete, or rename session files.
- **Skip binary and non-text content.** Ignore `image` content blocks
  and `thinking` blocks in messages.
- **Cap scan scope.** Search at most 50 session files per invocation.
  Prefer recent files when the candidate set exceeds the cap.
- **All workspaces by default.** Scan every subdirectory under
  `~/.omp/agent/sessions/`. Do not ask which workspace to search.
- **Python inline.** All search logic runs via `eval` cells. No
  external scripts.
