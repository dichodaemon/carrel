#!/usr/bin/env python3
"""Interactive curses tool for selecting and deleting local git branches."""

import curses
import subprocess
import sys


def get_branches():
    """Return list of (branch_name, is_current, date_str) tuples sorted by most recent first."""
    result = subprocess.run(
        [
            "git", "branch",
            "--format=%(refname:short)\t%(HEAD)\t%(committerdate:format:%Y-%m-%d %H:%M)",
            "--sort=-committerdate",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    branches = []
    current = None
    for line in result.stdout.strip().splitlines():
        parts = line.split("\t", 2)
        if len(parts) != 3:
            continue
        name, head, date_str = parts
        is_current = head == "*"
        if is_current:
            current = name
        branches.append((name, is_current, date_str))
    return branches, current


def delete_branches(branch_names):
    """Delete the given branches. Returns (deleted, failed) lists."""
    deleted = []
    failed = []
    for name in branch_names:
        result = subprocess.run(
            ["git", "branch", "-d", name],
            capture_output=True,
            text=True,
        )
        if result.returncode == 0:
            deleted.append(name)
        else:
            result2 = subprocess.run(
                ["git", "branch", "-D", name],
                capture_output=True,
                text=True,
            )
            if result2.returncode == 0:
                deleted.append(name)
            else:
                failed.append((name, result2.stderr.strip()))
    return deleted, failed


HELP_TEXT = (
    "  j/k or arrows: move    space: toggle    a: select all    "
    "n: select none    d/enter: delete    q: quit"
)


def draw_ui(stdscr, branches, selected, cursor, offset, status_msg):
    stdscr.erase()
    h, w = stdscr.getmaxyx()

    # Title bar
    title = " Git Branch Cleaner "
    stdscr.attron(curses.color_pair(1) | curses.A_BOLD)
    stdscr.addstr(0, 0, title.center(w)[:w])
    stdscr.attroff(curses.color_pair(1) | curses.A_BOLD)

    # Help line
    stdscr.attron(curses.color_pair(3))
    stdscr.addstr(1, 0, HELP_TEXT[:w])
    stdscr.attroff(curses.color_pair(3))

    # Separator
    stdscr.attron(curses.color_pair(3))
    stdscr.addstr(2, 0, "─" * w)
    stdscr.attroff(curses.color_pair(3))

    list_rows = h - 5  # rows available for the branch list
    visible = branches[offset : offset + list_rows]

    date_col = 16  # width reserved on the right for the date field

    for i, (name, is_current, date_str) in enumerate(visible):
        row = i + 3
        idx = offset + i
        is_sel = name in selected
        is_cursor = idx == cursor

        marker = "[x]" if is_sel else "[ ]"
        current_tag = " (current)" if is_current else ""
        label = f" {marker} {name}{current_tag}"

        # Truncate label so it doesn't overlap the date column
        max_label = w - date_col - 1
        label = label[:max_label].ljust(max_label)
        date_part = date_str[:date_col].rjust(date_col)
        line = label + date_part

        if is_cursor:
            stdscr.attron(curses.color_pair(2) | curses.A_BOLD)
            stdscr.addstr(row, 0, line[:w].ljust(w))
            stdscr.attroff(curses.color_pair(2) | curses.A_BOLD)
        elif is_current:
            stdscr.attron(curses.color_pair(4))
            stdscr.addstr(row, 0, line[:w])
            stdscr.attroff(curses.color_pair(4))
        else:
            attr = curses.A_BOLD if is_sel else curses.A_NORMAL
            stdscr.attron(attr)
            stdscr.addstr(row, 0, line[:w])
            stdscr.attroff(attr)

    # Status bar
    sel_count = len(selected)
    footer = f" {sel_count} selected  |  {len(branches)} branches total"
    stdscr.attron(curses.color_pair(1))
    stdscr.addstr(h - 2, 0, footer[:w].ljust(w))
    stdscr.attroff(curses.color_pair(1))

    # Status message
    if status_msg:
        stdscr.attron(curses.color_pair(5) | curses.A_BOLD)
        stdscr.addstr(h - 1, 0, status_msg[:w])
        stdscr.attroff(curses.color_pair(5) | curses.A_BOLD)

    stdscr.refresh()


def confirm_delete(stdscr, to_delete):
    """Show a confirmation prompt. Returns True if user confirms."""
    h, w = stdscr.getmaxyx()
    stdscr.erase()

    stdscr.attron(curses.color_pair(5) | curses.A_BOLD)
    stdscr.addstr(0, 0, f" Confirm deletion of {len(to_delete)} branch(es)? [y/N] "[:w])
    stdscr.attroff(curses.color_pair(5) | curses.A_BOLD)

    max_display = h - 3
    for i, name in enumerate(to_delete[:max_display]):
        stdscr.addstr(i + 1, 2, f"- {name}"[:w])
    if len(to_delete) > max_display:
        stdscr.addstr(max_display + 1, 2, f"  ... and {len(to_delete) - max_display} more"[:w])

    stdscr.refresh()
    key = stdscr.getch()
    return key in (ord("y"), ord("Y"))


def main(stdscr):
    curses.curs_set(0)
    curses.start_color()
    curses.use_default_colors()

    # color pairs: (fg, bg)
    curses.init_pair(1, curses.COLOR_BLACK, curses.COLOR_WHITE)   # title / status bar
    curses.init_pair(2, curses.COLOR_BLACK, curses.COLOR_CYAN)    # cursor row
    curses.init_pair(3, curses.COLOR_WHITE, -1)                   # help / separator
    curses.init_pair(4, curses.COLOR_YELLOW, -1)                  # current branch
    curses.init_pair(5, curses.COLOR_RED, -1)                     # warnings / confirm

    try:
        branches, current_branch = get_branches()
    except subprocess.CalledProcessError as e:
        curses.endwin()
        print(f"Error: not a git repository or git command failed.\n{e}", file=sys.stderr)
        sys.exit(1)

    if not branches:
        curses.endwin()
        print("No local branches found.")
        sys.exit(0)

    selected = set()
    cursor = 0
    offset = 0
    status_msg = ""

    while True:
        h, w = stdscr.getmaxyx()
        list_rows = h - 5

        # Keep cursor in view
        if cursor < offset:
            offset = cursor
        elif cursor >= offset + list_rows:
            offset = cursor - list_rows + 1

        draw_ui(stdscr, branches, selected, cursor, offset, status_msg)
        status_msg = ""

        key = stdscr.getch()

        if key in (ord("q"), ord("Q"), 27):  # q or Escape
            break

        elif key in (curses.KEY_DOWN, ord("j")):
            cursor = min(cursor + 1, len(branches) - 1)

        elif key in (curses.KEY_UP, ord("k")):
            cursor = max(cursor - 1, 0)

        elif key == curses.KEY_PPAGE:
            cursor = max(cursor - list_rows, 0)

        elif key == curses.KEY_NPAGE:
            cursor = min(cursor + list_rows, len(branches) - 1)

        elif key == ord(" "):
            name, is_current, _date = branches[cursor]
            if is_current:
                status_msg = " Cannot delete the current branch."
            elif name in selected:
                selected.discard(name)
            else:
                selected.add(name)

        elif key == ord("a"):
            for name, is_current, _date in branches:
                if not is_current:
                    selected.add(name)

        elif key == ord("n"):
            selected.clear()

        elif key in (ord("d"), ord("D"), 10, 13):  # d or Enter
            if not selected:
                status_msg = " No branches selected."
                continue

            to_delete = sorted(selected)
            if confirm_delete(stdscr, to_delete):
                curses.endwin()
                deleted, failed = delete_branches(to_delete)
                print(f"\nDeleted {len(deleted)} branch(es):")
                for name in deleted:
                    print(f"  - {name}")
                if failed:
                    print(f"\nFailed to delete {len(failed)} branch(es):")
                    for name, err in failed:
                        print(f"  - {name}: {err}")
                return
            else:
                status_msg = " Deletion cancelled."


def run():
    try:
        curses.wrapper(main)
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    run()
