"""Fetch a Google Doc as Markdown."""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any

import google.oauth2.credentials  # type: ignore[import-untyped]
import googleapiclient.discovery  # type: ignore[import-untyped]
import googleapiclient.errors  # type: ignore[import-untyped]

from doc_fetch import ApiError, AuthError, InputError
from doc_fetch import convert, urls
from lib.oauth_ssh.auth import PlatformConfig, authenticate

CREDENTIALS_PATH: Path = Path.home() / ".config" / "doc-fetch" / "google_credentials.json"

_TOKEN_URL = "https://oauth2.googleapis.com/token"


def _load_credentials() -> tuple[str, str]:
    """Return ``(client_id, client_secret)`` from the credentials file."""
    if not CREDENTIALS_PATH.exists():
        raise InputError(
            f"Google credentials not found at {CREDENTIALS_PATH}\n"
            "Download an OAuth client-secret JSON from the Google Cloud Console "
            "and save it there.  Expected format:\n"
            '  {"installed": {"client_id": "...", "client_secret": "...", ...}}'
        )

    try:
        data: dict[str, Any] = json.loads(CREDENTIALS_PATH.read_text())
    except (json.JSONDecodeError, OSError) as exc:
        raise InputError(f"Failed to read {CREDENTIALS_PATH}: {exc}") from exc

    installed = data.get("installed")
    if not isinstance(installed, dict):
        raise InputError(
            f"{CREDENTIALS_PATH} is not in the expected format.\n"
            'It must contain an "installed" key with "client_id" and '
            '"client_secret" inside.'
        )

    client_id = installed.get("client_id")
    client_secret = installed.get("client_secret")
    if not client_id or not client_secret:
        raise InputError(
            f'{CREDENTIALS_PATH}: "installed" object must contain '
            '"client_id" and "client_secret".'
        )

    return str(client_id), str(client_secret)


def fetch(url_or_id: str) -> str:
    """Fetch a Google Doc and return its content as Markdown."""
    client_id, client_secret = _load_credentials()
    doc_id = urls.parse_google_doc_id(url_or_id)

    config = PlatformConfig(
        name="google",
        auth_url="https://accounts.google.com/o/oauth2/v2/auth",
        token_url=_TOKEN_URL,
        scopes=["https://www.googleapis.com/auth/drive.readonly"],
        client_id=client_id,
        client_secret=client_secret,
        extra_auth_params={"access_type": "offline", "prompt": "consent"},
        token_content_type="form",
    )

    access_token = authenticate(config)

    creds = google.oauth2.credentials.Credentials(
        token=access_token,
        token_uri=config.token_url,
        client_id=client_id,
        client_secret=client_secret,
    )

    service = googleapiclient.discovery.build("drive", "v3", credentials=creds)

    try:
        html_bytes: bytes = (
            service.files()
            .export(fileId=doc_id, mimeType="text/html")
            .execute()
        )
    except googleapiclient.errors.HttpError as exc:
        status = exc.resp.status if exc.resp else 0
        if status == 403:
            raise AuthError(
                f"Access denied for document {doc_id} (HTTP 403)"
            ) from exc
        raise ApiError(
            f"Google Drive API error (HTTP {status}) for document {doc_id}: {exc}"
        ) from exc

    html = html_bytes.decode("utf-8") if isinstance(html_bytes, bytes) else html_bytes
    return convert.html_to_markdown(html)
