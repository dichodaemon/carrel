"""CLI entry point for doc-fetch."""
from __future__ import annotations

import argparse
import sys
import traceback

from doc_fetch import AuthError, ApiError, DocFetchError, InputError
from doc_fetch import confluence, google


def main() -> None:
    parser = argparse.ArgumentParser(prog="doc-fetch", description="Fetch documents as markdown.")
    subparsers = parser.add_subparsers(dest="command")

    gdoc_parser = subparsers.add_parser("gdoc", help="Fetch a Google Doc.")
    gdoc_parser.add_argument("url_or_id", help="Google Doc URL or document ID.")
    gdoc_parser.add_argument("-o", "--output", help="Write output to file instead of stdout.")

    confluence_parser = subparsers.add_parser("confluence", help="Fetch a Confluence page.")
    confluence_parser.add_argument("url_or_id", help="Confluence page URL or page ID.")
    confluence_parser.add_argument("-o", "--output", help="Write output to file instead of stdout.")

    args = parser.parse_args()

    if args.command is None:
        parser.print_usage(sys.stderr)
        raise SystemExit(2)

    try:
        if args.command == "gdoc":
            result = google.fetch(args.url_or_id)
        else:
            result = confluence.fetch(args.url_or_id)

        if args.output:
            with open(args.output, "w", encoding="utf-8") as f:
                f.write(result)
        else:
            sys.stdout.write(result)

    except DocFetchError as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(exc.exit_code)
    except KeyboardInterrupt:
        raise SystemExit(130)
    except Exception:
        traceback.print_exc(file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
