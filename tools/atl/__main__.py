from __future__ import annotations

import argparse
import json
import sys
import traceback
from typing import Optional

from atl import AtlError, InputError
from atl.client import AtlassianClient
from atl import content
from atl import jira as jira_ops
from atl import confluence as confluence_ops


def _build_parser() -> tuple[argparse.ArgumentParser, argparse.ArgumentParser, argparse.ArgumentParser]:
    parser = argparse.ArgumentParser(prog="atl", description="Atlassian CLI for Jira and Confluence")
    subs = parser.add_subparsers(dest="service")

    # -- Jira ----------------------------------------------------------------
    jira_parser = subs.add_parser("jira", help="Jira operations")
    jira_subs = jira_parser.add_subparsers(dest="command")

    # jira get
    p = jira_subs.add_parser("get", help="Get an issue")
    p.add_argument("key", help="Issue key (e.g. PROJ-123)")

    # jira create
    p = jira_subs.add_parser("create", help="Create an issue")
    p.add_argument("-p", "--project", required=True, help="Project key")
    p.add_argument("-t", "--type", required=True, help="Issue type name")
    p.add_argument("-s", "--summary", required=True, help="Issue summary")
    p.add_argument("--body", default=None, help="Description text")
    p.add_argument("--parent", default=None, help="Parent issue key (creates a child task)")
    p.add_argument("--assignee", default=None, help="Assignee account ID")

    # jira edit
    p = jira_subs.add_parser("edit", help="Edit an issue")
    p.add_argument("key", help="Issue key")
    p.add_argument("-s", "--summary", default=None, help="New summary")
    p.add_argument("--body", default=None, help="New description text")

    # jira transition
    p = jira_subs.add_parser("transition", help="Transition an issue")
    p.add_argument("key", help="Issue key")
    p.add_argument("status", help="Target status name")

    # jira delete
    p = jira_subs.add_parser("delete", help="Delete an issue")
    p.add_argument("key", help="Issue key")

    # jira search
    p = jira_subs.add_parser("search", help="Search issues via JQL")
    p.add_argument("jql", help="JQL query string")

    # jira comment
    p = jira_subs.add_parser("comment", help="Add a comment to an issue")
    p.add_argument("key", help="Issue key")
    p.add_argument("message", nargs="?", default=None, help="Comment text")

    # -- Confluence -----------------------------------------------------------
    conf_parser = subs.add_parser("confluence", help="Confluence operations")
    conf_subs = conf_parser.add_subparsers(dest="command")

    # confluence get
    p = conf_subs.add_parser("get", help="Get a page or blog post")
    p.add_argument("url_or_id", help="Page URL or numeric ID")
    p.add_argument("--json", action="store_true", dest="as_json", help="Output full JSON instead of markdown")

    # confluence create
    p = conf_subs.add_parser("create", help="Create a page")
    p.add_argument("-s", "--space", required=True, help="Space key")
    p.add_argument("-t", "--title", required=True, help="Page title")
    p.add_argument("-p", "--parent", default=None, help="Parent page ID")
    p.add_argument("--body", default=None, help="Page body (markdown)")

    # confluence edit
    p = conf_subs.add_parser("edit", help="Edit a page or blog post")
    p.add_argument("url_or_id", help="Page URL or numeric ID")
    p.add_argument("-t", "--title", default=None, help="New title")
    p.add_argument("--body", default=None, help="New body (markdown)")

    # confluence delete
    p = conf_subs.add_parser("delete", help="Delete a page or blog post")
    p.add_argument("url_or_id", help="Page URL or numeric ID")

    # confluence search
    p = conf_subs.add_parser("search", help="Search via CQL")
    p.add_argument("cql", help="CQL query string")

    return parser, jira_parser, conf_parser


def _out(data: object) -> None:
    """Print JSON to stdout, compact and UTF-8 safe."""
    print(json.dumps(data, ensure_ascii=False))


def _run_jira(client: AtlassianClient, args: argparse.Namespace, jira_parser: argparse.ArgumentParser) -> None:
    cmd: Optional[str] = args.command
    if cmd is None:
        jira_parser.print_usage(sys.stderr)
        sys.exit(2)

    if cmd == "get":
        _out(jira_ops.get_issue(client, args.key))

    elif cmd == "create":
        body = content.read_body(args.body)
        _out(jira_ops.create_issue(client, args.project, args.type, args.summary, body,
                                   parent=args.parent, assignee_account_id=args.assignee))

    elif cmd == "edit":
        body = content.read_body(args.body)
        jira_ops.edit_issue(client, args.key, summary=args.summary, description=body)

    elif cmd == "transition":
        jira_ops.transition_issue(client, args.key, args.status)

    elif cmd == "delete":
        jira_ops.delete_issue(client, args.key)

    elif cmd == "search":
        _out(jira_ops.search_issues(client, args.jql))

    elif cmd == "comment":
        message = args.message
        if message is None:
            message = content.read_body(None)
        if message is None:
            raise InputError("Provide a comment message or pipe via stdin")
        _out(jira_ops.add_comment(client, args.key, message))


def _run_confluence(client: AtlassianClient, args: argparse.Namespace, conf_parser: argparse.ArgumentParser) -> None:
    cmd: Optional[str] = args.command
    if cmd is None:
        conf_parser.print_usage(sys.stderr)
        sys.exit(2)

    if cmd == "get":
        content_id, content_type = content.parse_confluence_url(args.url_or_id)
        result = confluence_ops.get_content(client, content_id, content_type)
        if args.as_json:
            _out(result)
        else:
            print(result["markdown"])

    elif cmd == "create":
        body = content.read_body(args.body)
        if body is None:
            raise InputError("Page body is required")
        _out(confluence_ops.create_page(client, args.space, args.title, body, parent_id=args.parent))

    elif cmd == "edit":
        content_id, content_type = content.parse_confluence_url(args.url_or_id)
        body = content.read_body(args.body)
        _out(confluence_ops.update_content(client, content_id, title=args.title, body_md=body, content_type=content_type))

    elif cmd == "delete":
        content_id, content_type = content.parse_confluence_url(args.url_or_id)
        confluence_ops.delete_content(client, content_id, content_type=content_type)

    elif cmd == "search":
        _out(confluence_ops.search(client, args.cql))


def main() -> int:
    parser, jira_parser, conf_parser = _build_parser()
    args = parser.parse_args()

    service: Optional[str] = args.service
    if service is None:
        parser.print_usage(sys.stderr)
        return 2

    # Check for missing sub-subcommand before creating the client.
    command: Optional[str] = getattr(args, "command", None)
    if command is None:
        sub = jira_parser if service == "jira" else conf_parser
        sub.print_usage(sys.stderr)
        return 2

    try:
        client = AtlassianClient()

        if service == "jira":
            _run_jira(client, args, jira_parser)
        elif service == "confluence":
            _run_confluence(client, args, conf_parser)
    except AtlError as exc:
        print(str(exc), file=sys.stderr)
        return exc.exit_code
    except KeyboardInterrupt:
        return 130
    except Exception:
        traceback.print_exc(file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
