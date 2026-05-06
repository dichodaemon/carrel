from __future__ import annotations


class AtlError(Exception):
    """Base error for atl CLI."""

    exit_code: int = 1


class InputError(AtlError):
    """User input or configuration error."""

    exit_code = 1


class AuthError(AtlError):
    """Authentication or authorization failure (401/403)."""

    exit_code = 2


class ApiError(AtlError):
    """Remote API error (non-2xx other than auth)."""

    exit_code = 3
