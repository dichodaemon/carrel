## Workflow Principles

These govern how you interact with the user across all tasks.

### Propose before implementing

When the user says "propose", "suggest", or asks for options: present the proposal in chat and stop. Do not create files, edit code, or execute commands until the user approves. This is a deliberation phase -- the user is exploring, not committing.

### Explain before executing

Before starting any non-trivial work, state what you plan to do and which files you will touch. The user must understand the plan before you act on it. A brief numbered list is sufficient -- not a full plan document, just enough that the user can say "go" or "wait."

### Ask before committing

Do not run `git commit` or `git push` without explicit approval. You may ask once for the entire workflow ("I'll commit and push when done -- OK?") rather than per-commit, but the default is to stop and ask.

---

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**
- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var

