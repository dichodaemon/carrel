"""HTML-to-Markdown conversion."""
from __future__ import annotations

import markdownify  # type: ignore[import-untyped]


def html_to_markdown(html: str) -> str:
    """Convert *html* to ATX-heading Markdown with images stripped."""
    return markdownify.markdownify(
        html, heading_style="ATX", strip=["img"]
    ).strip()
