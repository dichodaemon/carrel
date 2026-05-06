from __future__ import annotations

import dataclasses
import json
import pathlib
import sys
import time
from typing import Any, Optional
from urllib.parse import urlencode

import requests

from lib.oauth_ssh.callback import wait_for_code

_TOKEN_DIR = pathlib.Path.home() / ".config" / "oauth-ssh" / "tokens"
_EXPIRY_MARGIN_SECONDS = 60
_DEFAULT_CALLBACK_PORT = 8090


@dataclasses.dataclass
class PlatformConfig:
    """OAuth platform configuration."""

    name: str  # e.g. "google", "confluence"
    auth_url: str  # Authorization endpoint
    token_url: str  # Token exchange endpoint
    scopes: list[str]
    client_id: str
    client_secret: str
    redirect_uri: str = "http://localhost:8090/callback"
    extra_auth_params: Optional[dict[str, str]] = None
    token_content_type: str = "form"  # "form" or "json"


def authenticate(config: PlatformConfig) -> str:
    """Return a valid access token, using cache/refresh/consent as needed."""
    token_data = _load_token(config)

    if token_data and not _is_expired(token_data):
        return token_data["access_token"]

    # Try refresh if we have a refresh token.
    if token_data and token_data.get("refresh_token"):
        refreshed = _refresh_token(config, token_data["refresh_token"])
        if refreshed is not None:
            return refreshed["access_token"]

    # Full consent flow.
    return _consent_flow(config)


# ---------------------------------------------------------------------------
# Token persistence
# ---------------------------------------------------------------------------


def _token_path(config: PlatformConfig) -> pathlib.Path:
    return _TOKEN_DIR / f"{config.name}.json"


def _load_token(config: PlatformConfig) -> Optional[dict[str, Any]]:
    path = _token_path(config)
    if not path.exists():
        return None
    try:
        data = json.loads(path.read_text())
        if not isinstance(data, dict) or "access_token" not in data:
            # Corrupt — remove and treat as absent.
            path.unlink(missing_ok=True)
            return None
        return data
    except (json.JSONDecodeError, OSError):
        # Corrupt file — delete and treat as missing.
        try:
            path.unlink(missing_ok=True)
        except OSError:
            pass
        return None


def _save_token(config: PlatformConfig, data: dict[str, Any]) -> None:
    path = _token_path(config)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2))


def _is_expired(token_data: dict[str, Any]) -> bool:
    obtained_at: float = token_data.get("obtained_at", 0.0)
    expires_in: float = token_data.get("expires_in", 0)
    return time.time() >= obtained_at + expires_in - _EXPIRY_MARGIN_SECONDS


# ---------------------------------------------------------------------------
# Token exchange helpers
# ---------------------------------------------------------------------------


def _post_token(
    config: PlatformConfig,
    payload: dict[str, str],
) -> requests.Response:
    """POST to the token endpoint, choosing form vs JSON encoding."""
    if config.token_content_type == "json":
        return requests.post(config.token_url, json=payload)
    return requests.post(config.token_url, data=payload)


def _refresh_token(
    config: PlatformConfig,
    refresh_token: str,
) -> Optional[dict[str, Any]]:
    """Attempt to refresh the access token. Returns saved token data or None."""
    payload = {
        "grant_type": "refresh_token",
        "client_id": config.client_id,
        "client_secret": config.client_secret,
        "refresh_token": refresh_token,
    }
    try:
        resp = _post_token(config, payload)
        if resp.status_code == 403:
            print(
                f"Refresh failed (403 Forbidden) for {config.name}.",
                file=sys.stderr,
            )
            return None
        if not resp.ok:
            print(
                f"Refresh failed ({resp.status_code}) for {config.name}, "
                "re-authenticating.",
                file=sys.stderr,
            )
            return None
    except requests.RequestException as exc:
        print(f"Refresh request failed: {exc}", file=sys.stderr)
        return None

    data = resp.json()
    # Preserve old refresh_token if the response doesn't supply a new one
    # (Google never rotates; Atlassian does).
    if "refresh_token" not in data:
        data["refresh_token"] = refresh_token
    data["obtained_at"] = time.time()
    _save_token(config, data)
    return data


def _exchange_code(
    config: PlatformConfig,
    code: str,
) -> dict[str, Any]:
    """Exchange authorization code for tokens."""
    payload = {
        "grant_type": "authorization_code",
        "client_id": config.client_id,
        "client_secret": config.client_secret,
        "code": code,
        "redirect_uri": config.redirect_uri,
    }
    resp = _post_token(config, payload)
    if not resp.ok:
        raise RuntimeError(
            f"Token exchange failed ({resp.status_code}): {resp.text}"
        )
    data = resp.json()
    data["obtained_at"] = time.time()
    _save_token(config, data)
    return data


# ---------------------------------------------------------------------------
# Consent flow
# ---------------------------------------------------------------------------


def _build_auth_url(config: PlatformConfig) -> str:
    params: dict[str, str] = {
        "client_id": config.client_id,
        "redirect_uri": config.redirect_uri,
        "response_type": "code",
        "scope": " ".join(config.scopes),
    }
    if config.extra_auth_params:
        params.update(config.extra_auth_params)
    return f"{config.auth_url}?{urlencode(params)}"


def _consent_flow(config: PlatformConfig) -> str:
    """Run the full browser-based consent flow and return the access token."""
    auth_url = _build_auth_url(config)
    print(
        f"\n[{config.name}] Visit this URL to authorize:",
        file=sys.stderr,
    )
    print(f"  {auth_url}\n", file=sys.stderr)

    # Extract port from redirect_uri (default 8090).
    port = _DEFAULT_CALLBACK_PORT
    try:
        from urllib.parse import urlparse as _urlparse

        parsed = _urlparse(config.redirect_uri)
        if parsed.port:
            port = parsed.port
    except Exception:
        pass

    code, via_paste = wait_for_code(port)
    token_data = _exchange_code(config, code)

    if via_paste:
        print(
            "\nTip: to skip the paste step in the future, add this to "
            "~/.ssh/config\nfor your connection to this host:\n\n"
            f"    RemoteForward {port} localhost:{port}\n",
            file=sys.stderr,
        )

    return token_data["access_token"]
