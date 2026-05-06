"""URL parsing utilities for Google Docs and Confluence pages."""
from __future__ import annotations

import re


# Matches docs.google.com/document/d/<ID> with optional trailing path/query/fragment.
_GDOC_RE = re.compile(r"docs\.google\.com/document/d/([^/?#]+)")

# Matches Confluence content URLs. Captures the instance host, content type,
# and numeric content ID. Handles both page URLs (.../pages/<ID>/...) and
# blog post URLs (.../blog/YYYY/MM/DD/<ID>/...).
_CONFLUENCE_PAGE_RE = re.compile(
    r"(https?://[^/]+\.atlassian\.net)/wiki/spaces/[^/]+/pages/([0-9]+)"
)
_CONFLUENCE_BLOG_RE = re.compile(
    r"(https?://[^/]+\.atlassian\.net)/wiki/spaces/[^/]+/blog/[0-9/]+/([0-9]+)"
)


def parse_google_doc_id(url_or_id: str) -> str:
    """Extract the document ID from a Google Docs URL.

    If *url_or_id* doesn't look like a URL, return it unchanged (assumed to
    be a raw document ID).
    """
    m = _GDOC_RE.search(url_or_id)
    if m:
        return m.group(1)
    return url_or_id


def parse_confluence_url(url_or_id: str) -> tuple[str, str | None, str]:
    """Extract ``(content_id, instance_host, content_type)`` from a Confluence URL.

    *content_type* is ``"pages"`` for regular pages or ``"blogposts"`` for blog
    posts, matching the v2 API endpoint names.

    *instance_host* is the scheme + authority portion before ``/wiki/``
    (e.g. ``https://acme.atlassian.net``).  If *url_or_id* doesn't match
    the expected URL shape, return ``(url_or_id, None, "pages")``.
    """
    m = _CONFLUENCE_PAGE_RE.search(url_or_id)
    if m:
        return m.group(2), m.group(1), "pages"
    m = _CONFLUENCE_BLOG_RE.search(url_or_id)
    if m:
        return m.group(2), m.group(1), "blogposts"
    return url_or_id, None, "pages"