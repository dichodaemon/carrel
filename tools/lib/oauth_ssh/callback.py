from __future__ import annotations

import select
import sys
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Optional
from urllib.parse import parse_qs, urlparse


def _extract_code_from_input(user_input: str) -> str:
    """Extract auth code from a pasted URL or return raw input as code."""
    stripped = user_input.strip()
    if not stripped:
        return ""
    # Try parsing as URL with ?code= parameter.
    try:
        parsed = urlparse(stripped)
        params = parse_qs(parsed.query)
        codes = params.get("code")
        if codes:
            return codes[0]
    except Exception:
        pass
    # Fall back: treat entire input as raw code.
    return stripped


def wait_for_code(port: int) -> tuple[str, bool]:
    """Wait for OAuth authorization code via HTTP callback or stdin paste.

    Returns (auth_code, via_paste) where via_paste is True if the code was
    obtained from user pasting rather than the HTTP redirect.
    """
    code_event = threading.Event()
    result: dict[str, Optional[str]] = {"code": None, "via_paste": None}

    class _CallbackHandler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            params = parse_qs(urlparse(self.path).query)
            codes = params.get("code")
            if codes:
                result["code"] = codes[0]
                result["via_paste"] = "false"
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(
                    b"Authorization received. You can close this window."
                )
                code_event.set()
            else:
                self.send_response(400)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(b"Missing authorization code.")

        def log_message(self, format: str, *args: object) -> None:
            # Suppress HTTP server logging.
            pass

    server = HTTPServer(("localhost", port), _CallbackHandler)
    server.timeout = 1

    def _serve() -> None:
        while not code_event.is_set():
            server.handle_request()

    thread = threading.Thread(target=_serve, daemon=True)
    thread.start()

    try:
        print(
            f"Listening for redirect on localhost:{port}.",
            file=sys.stderr,
        )
        print(
            "If your browser shows a connection error, paste the full URL here:",
            file=sys.stderr,
        )

        while not code_event.is_set():
            # Non-blocking stdin check with 1s timeout so we also poll
            # the HTTP callback state each iteration.
            ready, _, _ = select.select([sys.stdin], [], [], 1.0)
            if ready:
                line = sys.stdin.readline()
                if not line:
                    # EOF on stdin — keep waiting for HTTP callback.
                    continue
                code = _extract_code_from_input(line)
                if code:
                    result["code"] = code
                    result["via_paste"] = "true"
                    code_event.set()
    finally:
        server.server_close()

    assert result["code"] is not None, "code_event set but no code captured"
    return result["code"], result["via_paste"] == "true"
