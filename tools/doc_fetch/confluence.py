"""Fetch a Confluence page as Markdown using a personal API token."""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any

import requests

from doc_fetch import ApiError, AuthError, InputError
from doc_fetch import convert, urls

CREDENTIALS_PATH: Path = (
    Path.home() / ".config" / "doc-fetch" / "confluence_credentials.json"
)

# Default instance used when the user provides a raw page ID instead of a URL.
INSTANCE_CACHE_PATH: Path = (
    Path.home() / ".config" / "doc-fetch" / "confluence_instance.json"
)


def _load_credentials() -> tuple[str, str]:
    """Return ``(email, api_token)`` from the credentials file."""
    if not CREDENTIALS_PATH.exists():
        raise InputError(
            f"Confluence credentials not found at {CREDENTIALS_PATH}\n"
            "Create an API token at:\n"
            "  https://id.atlassian.com/manage-profile/security/api-tokens\n"
            "Then save a JSON file there with the following format:\n"
            '  {"email": "you@company.com", "api_token": "..."}'
        )

    try:
        data: dict[str, Any] = json.loads(CREDENTIALS_PATH.read_text())
    except (json.JSONDecodeError, OSError) as exc:
        raise InputError(
            f"Failed to read {CREDENTIALS_PATH}: {exc}"
        ) from exc

    email = data.get("email")
    api_token = data.get("api_token")
    if not email or not api_token:
        raise InputError(
            f'{CREDENTIALS_PATH} must contain "email" and "api_token".\n'
            "Create an API token at:\n"
            "  https://id.atlassian.com/manage-profile/security/api-tokens"
        )

    return str(email), str(api_token)


def _resolve_instance(instance_host: str | None) -> str:
    """Return the Confluence instance base URL (e.g. https://acme.atlassian.net).

    If *instance_host* was extracted from the URL, cache it for future raw-ID
    invocations. If None (raw page ID was given), load from cache.
    """
    if instance_host:
        # Cache for future raw-ID usage.
        INSTANCE_CACHE_PATH.parent.mkdir(parents=True, exist_ok=True)
        INSTANCE_CACHE_PATH.write_text(
            json.dumps({"instance": instance_host}, indent=2) + "\n"
        )
        return instance_host

    # No instance in the URL -- try cache.
    if INSTANCE_CACHE_PATH.exists():
        try:
            cached = json.loads(INSTANCE_CACHE_PATH.read_text())
            inst = cached.get("instance")
            if inst:
                return str(inst)
        except (json.JSONDecodeError, OSError):
            pass

    raise InputError(
        "Cannot determine Confluence instance from a raw page ID.\n"
        "Provide a full URL (https://INSTANCE.atlassian.net/wiki/spaces/SPACE/pages/ID)\n"
        "or run once with a full URL so the instance is cached."
    )


def fetch(url_or_id: str) -> str:
    """Fetch a Confluence page or blog post and return its content as Markdown."""
    email, api_token = _load_credentials()
    content_id, instance_host, content_type = urls.parse_confluence_url(url_or_id)
    base_url = _resolve_instance(instance_host)

    # v2 API: /wiki/api/v2/pages/{id} for pages, /wiki/api/v2/blogposts/{id} for blogs.
    api_url = f"{base_url}/wiki/api/v2/{content_type}/{content_id}?body-format=export_view"
    resp = requests.get(
        api_url,
        auth=(email, api_token),
        headers={"Accept": "application/json"},
        timeout=30,
    )

    if resp.status_code == 401:
        raise AuthError(
            f"Authentication failed (HTTP 401). Check email and API token in\n"
            f"  {CREDENTIALS_PATH}"
        )
    if resp.status_code == 403:
        raise AuthError(
            f"Access denied for Confluence content {content_id} (HTTP 403)"
        )
    if resp.status_code == 404:
        raise ApiError(
            f"Confluence content {content_id} not found (HTTP 404) at {base_url}"
        )
    if not resp.ok:
        raise ApiError(
            f"Confluence API error: HTTP {resp.status_code} from {api_url}"
        )

    payload: dict[str, Any] = resp.json()

    try:
        html: str = payload["body"]["export_view"]["value"]
    except (KeyError, TypeError) as exc:
        raise ApiError(
            f"Confluence response for content {content_id} is missing "
            '"body.export_view.value"'
        ) from exc

    return convert.html_to_markdown(html)