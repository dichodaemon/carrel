"""doc-fetch: fetch Google Docs and Confluence pages as markdown."""
from __future__ import annotations


class DocFetchError(Exception):
    """Base error for doc-fetch operations."""

    exit_code: int = 1


class InputError(DocFetchError):
    """Invalid input or missing configuration."""

    exit_code = 1


class AuthError(DocFetchError):
    """Authentication or authorization failure."""

    exit_code = 2


class ApiError(DocFetchError):
    """Remote API or network error."""

    exit_code = 3
