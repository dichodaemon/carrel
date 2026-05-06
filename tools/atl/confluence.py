"""Confluence CRUD operations."""
from __future__ import annotations

from typing import Any, Optional

from atl import ApiError, InputError
from atl.client import AtlassianClient
from atl.content import html_to_markdown, markdown_to_storage


def _build_url(client: AtlassianClient, response: dict[str, Any]) -> str:
    """Construct a browser URL from a Confluence API response."""
    links = response.get("_links", {})
    webui = links.get("webui", "")
    if webui:
        return client.instance + "/wiki" + webui
    return ""


def get_content(
    client: AtlassianClient, content_id: str, content_type: str = "pages",
) -> dict[str, Any]:
    """Fetch a page or blog post and return metadata + content."""
    resp = client.get(
        f"/wiki/api/v2/{content_type}/{content_id}",
        params={"body-format": "export_view"},
    )
    data = resp.json()

    html = ""
    body = data.get("body", {})
    export_view = body.get("export_view", {})
    if export_view:
        html = export_view.get("value", "")

    version = data.get("version", {})

    return {
        "id": data.get("id", content_id),
        "title": data.get("title", ""),
        "url": _build_url(client, data),
        "version": version.get("number", 1),
        "html": html,
        "markdown": html_to_markdown(html) if html else "",
    }


def create_page(
    client: AtlassianClient,
    space_key: str,
    title: str,
    body_md: str,
    parent_id: Optional[str] = None,
) -> dict[str, Any]:
    """Create a Confluence page. Body is markdown, converted to storage format.

    Note: the v2 API only supports creating pages, not blog posts.
    """
    # Resolve space key -> space ID.
    resp = client.get("/wiki/api/v2/spaces", params={"keys": space_key})
    spaces = resp.json().get("results", [])
    if not spaces:
        raise ApiError(f"Space not found: {space_key}")
    space_id = spaces[0]["id"]

    payload: dict[str, Any] = {
        "spaceId": space_id,
        "title": title,
        "body": {
            "representation": "storage",
            "value": markdown_to_storage(body_md),
        },
    }
    if parent_id:
        payload["parentId"] = parent_id

    resp = client.post("/wiki/api/v2/pages", payload)
    data = resp.json()
    return {
        "id": data["id"],
        "title": data.get("title", title),
        "url": _build_url(client, data),
    }


def update_content(
    client: AtlassianClient,
    content_id: str,
    title: Optional[str] = None,
    body_md: Optional[str] = None,
    content_type: str = "pages",
) -> dict[str, Any]:
    """Update a page or blog post. Fetches current version automatically."""
    if title is None and body_md is None:
        raise InputError("Nothing to update. Provide --title or --body.")

    # Fetch current state for version number and fallback title.
    current = client.get(f"/wiki/api/v2/{content_type}/{content_id}").json()
    current_version: int = current.get("version", {}).get("number", 1)
    current_title: str = current.get("title", "")

    payload: dict[str, Any] = {
        "id": content_id,
        "status": "current",
        "title": title if title is not None else current_title,
        "version": {"number": current_version + 1},
    }
    if body_md is not None:
        payload["body"] = {
            "representation": "storage",
            "value": markdown_to_storage(body_md),
        }

    resp = client.put(f"/wiki/api/v2/{content_type}/{content_id}", payload)
    data = resp.json()
    return {
        "id": data.get("id", content_id),
        "title": data.get("title", payload["title"]),
        "url": _build_url(client, data),
    }


def delete_content(
    client: AtlassianClient, content_id: str, content_type: str = "pages",
) -> None:
    """Delete a page or blog post."""
    client.delete(f"/wiki/api/v2/{content_type}/{content_id}")


def search(client: AtlassianClient, cql: str) -> list[dict[str, Any]]:
    """Search Confluence via CQL (v1 REST API)."""
    resp = client.get(
        "/wiki/rest/api/search",
        params={"cql": cql, "limit": "25"},
    )
    raw_results = resp.json().get("results", [])
    results: list[dict[str, Any]] = []
    for item in raw_results:
        content = item.get("content", {})
        content_type = content.get("type", "page")
        # Normalize to "page" or "blogpost".
        if content_type not in ("page", "blogpost"):
            content_type = "page"
        links = content.get("_links", {})
        webui = links.get("webui", "")
        url = client.instance + "/wiki" + webui if webui else ""
        results.append({
            "id": content.get("id", ""),
            "title": content.get("title", item.get("title", "")),
            "url": url,
            "type": content_type,
        })
    return results
