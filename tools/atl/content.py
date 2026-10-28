"""Content conversion helpers: ADF, markdown, Confluence storage, body reading."""
from __future__ import annotations

import re
import sys
from html.parser import HTMLParser
from typing import Any, Optional

import markdown as md_lib  # type: ignore[import-untyped]
import markdownify  # type: ignore[import-untyped]

# ---------------------------------------------------------------------------
# ADF (Atlassian Document Format) <-> markdown / plain text
# ---------------------------------------------------------------------------


def markdown_to_adf(text: str) -> dict[str, Any]:
    """Convert markdown to Atlassian Document Format.

    Supported: paragraphs, ATX headings, bullet and numbered lists (nested),
    fenced and indented code blocks, block quotes, horizontal rules, and the
    inline marks bold, italic, code, strikethrough and links. Hard line
    breaks (two trailing spaces) become ADF hard breaks. Plain text without
    markup comes out as paragraphs split on blank lines, as before.
    Unsupported HTML (tables, images) is flattened to its text.
    """
    html = md_lib.markdown(text, extensions=["fenced_code", "sane_lists"])
    builder = _AdfBuilder()
    builder.feed(html)
    builder.close()
    content = builder.result()
    if not content:
        content = [{"type": "paragraph", "content": [{"type": "text", "text": ""}]}]
    return {"type": "doc", "version": 1, "content": content}


_HEADING_TAGS = {"h1": 1, "h2": 2, "h3": 3, "h4": 4, "h5": 5, "h6": 6}
_INLINE_MARKS = {
    "strong": {"type": "strong"},
    "b": {"type": "strong"},
    "em": {"type": "em"},
    "i": {"type": "em"},
    "code": {"type": "code"},
    "del": {"type": "strike"},
    "s": {"type": "strike"},
}


class _AdfBuilder(HTMLParser):
    """Walk markdown-rendered HTML into an ADF node list."""

    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self._root: dict[str, Any] = {"type": "doc", "content": []}
        self._stack: list[dict[str, Any]] = [self._root]
        self._marks: list[dict[str, Any]] = []
        self._pre_depth = 0

    def result(self) -> list[dict[str, Any]]:
        return self._root["content"]

    # -- containers ------------------------------------------------------

    @property
    def _top(self) -> dict[str, Any]:
        return self._stack[-1]

    def _push(self, node: dict[str, Any]) -> None:
        self._top.setdefault("content", []).append(node)
        self._stack.append(node)

    def _pop(self, node_type: str) -> None:
        while len(self._stack) > 1:
            node = self._stack.pop()
            if node.get("type") == node_type:
                break

    def _text_container(self) -> dict[str, Any]:
        """Return the node inline text may be appended to, creating a
        paragraph when the current container only holds blocks."""
        top = self._top
        if top["type"] in ("paragraph", "heading", "codeBlock"):
            return top
        if top["type"] == "listItem":
            # Tight list items: markdown emits no <p>, so open one.
            self._push({"type": "paragraph", "content": []})
            return self._top
        if top["type"] in ("doc", "blockquote"):
            self._push({"type": "paragraph", "content": []})
            return self._top
        return top

    # -- HTMLParser hooks ------------------------------------------------

    def handle_starttag(self, tag: str, attrs: list[tuple[str, Optional[str]]]) -> None:
        if tag == "p":
            if self._top["type"] == "paragraph":
                self._pop("paragraph")
            self._push({"type": "paragraph", "content": []})
        elif tag in _HEADING_TAGS:
            self._push({"type": "heading", "attrs": {"level": _HEADING_TAGS[tag]}, "content": []})
        elif tag == "ul":
            self._close_open_paragraph()
            self._push({"type": "bulletList", "content": []})
        elif tag == "ol":
            self._close_open_paragraph()
            self._push({"type": "orderedList", "content": []})
        elif tag == "li":
            self._push({"type": "listItem", "content": []})
        elif tag == "pre":
            self._pre_depth += 1
            self._push({"type": "codeBlock", "content": []})
        elif tag == "blockquote":
            self._push({"type": "blockquote", "content": []})
        elif tag == "hr":
            self._top.setdefault("content", []).append({"type": "rule"})
        elif tag == "br":
            self._text_container()["content"].append({"type": "hardBreak"})
        elif tag == "a":
            href = dict(attrs).get("href") or ""
            self._marks.append({"type": "link", "attrs": {"href": href}})
        elif tag in _INLINE_MARKS:
            if tag == "code" and self._pre_depth:
                return  # <pre><code>: the codeBlock carries the text
            self._marks.append(dict(_INLINE_MARKS[tag]))

    def handle_endtag(self, tag: str) -> None:
        if tag == "p":
            self._pop("paragraph")
        elif tag in _HEADING_TAGS:
            self._pop("heading")
        elif tag == "ul":
            self._close_open_paragraph()
            self._pop("bulletList")
        elif tag == "ol":
            self._close_open_paragraph()
            self._pop("orderedList")
        elif tag == "li":
            self._close_open_paragraph()
            self._pop("listItem")
        elif tag == "pre":
            self._pre_depth -= 1
            self._pop("codeBlock")
        elif tag == "blockquote":
            self._close_open_paragraph()
            self._pop("blockquote")
        elif tag == "a":
            self._pop_mark("link")
        elif tag in _INLINE_MARKS:
            if tag == "code" and self._pre_depth:
                return
            self._pop_mark(_INLINE_MARKS[tag]["type"])

    def handle_data(self, data: str) -> None:
        if self._pre_depth:
            text = data
        else:
            text = re.sub(r"\s+", " ", data)
            if not text.strip() and self._top["type"] not in ("paragraph", "heading"):
                return  # inter-block whitespace
        if not text:
            return
        container = self._text_container()["content"]
        if not self._pre_depth and (not container or container[-1]["type"] == "hardBreak"):
            text = text.lstrip()
            if not text:
                return
        node: dict[str, Any] = {"type": "text", "text": text}
        if self._marks and not self._pre_depth:
            node["marks"] = [dict(m) for m in self._marks]
        container.append(node)

    # -- helpers ---------------------------------------------------------

    def _close_open_paragraph(self) -> None:
        if self._top["type"] == "paragraph":
            self._pop("paragraph")

    def _pop_mark(self, mark_type: str) -> None:
        for i in range(len(self._marks) - 1, -1, -1):
            if self._marks[i]["type"] == mark_type:
                del self._marks[i]
                return


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
    2. stdin if it is not a TTY and carries non-whitespace content.
    3. None (no body provided).

    Empty piped stdin counts as "no body": agent and CI shells have a
    non-TTY stdin even when nothing is piped, and treating that as an empty
    body would blank the field (e.g. ``jira edit -s`` clearing the
    description).
    """
    if flag_value is not None:
        return flag_value
    if not sys.stdin.isatty():
        piped = sys.stdin.read()
        if piped.strip():
            return piped
    return None
