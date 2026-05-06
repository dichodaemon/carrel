"""Content conversion helpers: ADF, markdown, Confluence storage, body reading."""
from __future__ import annotations

import re
import sys
from typing import Any, Optional

import markdown as md_lib  # type: ignore[import-untyped]
import markdownify  # type: ignore[import-untyped]

# ---------------------------------------------------------------------------
# ADF (Atlassian Document Format) <-> plain text
# ---------------------------------------------------------------------------


def text_to_adf(text: str) -> dict[str, Any]:
    """Convert plain text to minimal Atlassian Document Format."""
    paragraphs: list[dict[str, Any]] = []
    for chunk in text.split("\n\n"):
        stripped = chunk.strip()
        if stripped:
            paragraphs.append(
                {"type": "paragraph", "content": [{"type": "text", "text": stripped}]}
            )
    if not paragraphs:
        paragraphs.append(
            {"type": "paragraph", "content": [{"type": "text", "text": ""}]}
        )
    return {"type": "doc", "version": 1, "content": paragraphs}


def adf_to_text(adf: Optional[dict[str, Any]]) -> str:
    """Recursively extract plain text from an ADF document."""
    if adf is None:
        return ""
    parts: list[str] = []
    _walk_adf(adf, parts)
    return "\n".join(parts).strip()


def _walk_adf(node: dict[str, Any], parts: list[str]) -> None:
    if node.get("type") == "text":
        parts.append(node.get("text", ""))
    for child in node.get("content", []):
        if isinstance(child, dict):
            _walk_adf(child, parts)


# ---------------------------------------------------------------------------
# HTML / Markdown conversion
# ---------------------------------------------------------------------------


def html_to_markdown(html: str) -> str:
    """Convert HTML to ATX-heading markdown, stripping images."""
    return markdownify.markdownify(html, heading_style="ATX", strip=["img"]).strip()


def markdown_to_storage(text: str) -> str:
    """Convert markdown to Confluence storage format (HTML)."""
    return md_lib.markdown(text, extensions=["tables", "fenced_code"])


# ---------------------------------------------------------------------------
# Confluence URL parsing
# ---------------------------------------------------------------------------

_CONF_PAGE_RE = re.compile(
    r"atlassian\.net/wiki/spaces/[^/]+/pages/([0-9]+)"
)
_CONF_BLOG_RE = re.compile(
    r"atlassian\.net/wiki/spaces/[^/]+/blog/[0-9/]+/([0-9]+)"
)


def parse_confluence_url(url_or_id: str) -> tuple[str, str]:
    """Return ``(content_id, content_type)`` from a Confluence URL.

    *content_type* is ``"pages"`` or ``"blogposts"``, matching the v2 API
    endpoint names.  If the input doesn't look like a URL, treat it as a
    raw page ID and default to ``"pages"``.
    """
    m = _CONF_PAGE_RE.search(url_or_id)
    if m:
        return m.group(1), "pages"
    m = _CONF_BLOG_RE.search(url_or_id)
    if m:
        return m.group(1), "blogposts"
    return url_or_id, "pages"


# ---------------------------------------------------------------------------
# Body input helper
# ---------------------------------------------------------------------------


def read_body(flag_value: Optional[str]) -> Optional[str]:
    """Return body text from *flag_value*, stdin, or None.

    Priority:
    1. *flag_value* if not None.
    2. stdin if it is not a TTY (piped content).
    3. None (no body provided).
    """
    if flag_value is not None:
        return flag_value
    if not sys.stdin.isatty():
        return sys.stdin.read()
    return None
