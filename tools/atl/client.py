"""Atlassian HTTP client with credential loading and error handling."""
from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Optional

import requests

from atl import ApiError, AuthError, InputError

_CREDS_PATH = Path.home() / ".config" / "doc-fetch" / "atlassian_credentials.json"


class AtlassianClient:
    """Thin HTTP wrapper around the Atlassian REST APIs."""

    def __init__(self) -> None:
        if not _CREDS_PATH.exists():
            raise InputError(
                f"Atlassian credentials not found at {_CREDS_PATH}\n"
                "Create a JSON file with:\n"
                '  {"email": "you@company.com", "api_token": "...", '
                '"instance": "https://INSTANCE.atlassian.net"}'
            )

        try:
            data: dict[str, Any] = json.loads(_CREDS_PATH.read_text())
        except (json.JSONDecodeError, OSError) as exc:
            raise InputError(f"Failed to read {_CREDS_PATH}: {exc}") from exc

        self.email: str = data.get("email", "")
        self.api_token: str = data.get("api_token", "")
        self.instance: str = data.get("instance", "").rstrip("/")

        missing = [
            k for k, v in [
                ("email", self.email),
                ("api_token", self.api_token),
                ("instance", self.instance),
            ]
            if not v
        ]
        if missing:
            raise InputError(
                f"{_CREDS_PATH} is missing required fields: {', '.join(missing)}"
            )

    # -----------------------------------------------------------------
    # HTTP verbs
    # -----------------------------------------------------------------

    def get(
        self, path: str, params: Optional[dict[str, str]] = None,
    ) -> requests.Response:
        resp = requests.get(
            self.instance + path,
            params=params,
            auth=(self.email, self.api_token),
            headers={"Accept": "application/json"},
            timeout=30,
        )
        self._check(resp)
        return resp

    def post(
        self, path: str, json_body: Optional[dict[str, Any]] = None,
    ) -> requests.Response:
        resp = requests.post(
            self.instance + path,
            json=json_body,
            auth=(self.email, self.api_token),
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            timeout=30,
        )
        self._check(resp)
        return resp

    def put(
        self, path: str, json_body: Optional[dict[str, Any]] = None,
    ) -> requests.Response:
        resp = requests.put(
            self.instance + path,
            json=json_body,
            auth=(self.email, self.api_token),
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            timeout=30,
        )
        self._check(resp)
        return resp

    def delete(self, path: str) -> requests.Response:
        resp = requests.delete(
            self.instance + path,
            auth=(self.email, self.api_token),
            timeout=30,
        )
        self._check(resp)
        return resp

    # -----------------------------------------------------------------

    @staticmethod
    def _check(resp: requests.Response) -> None:
        if resp.status_code in (200, 201, 204):
            return
        if resp.status_code in (401, 403):
            raise AuthError(
                f"HTTP {resp.status_code} from {resp.url}"
            )
        if not resp.ok:
            # Include the first 300 chars of response body for diagnostics.
            body = resp.text[:300] if resp.text else ""
            raise ApiError(
                f"HTTP {resp.status_code} from {resp.url}: {body}"
            )
