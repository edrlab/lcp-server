#!/usr/bin/env python3
import email.utils
import json
import os
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote, urlparse


HOST = os.environ.get("LICENSE_SERVER_HOST", "127.0.0.1")
PORT = int(os.environ.get("LICENSE_SERVER_PORT", "8991"))

LICENSE_DIR = Path(
    os.environ.get(
        "LICENSE_SERVER_DIR",
        ".local/lcp-update-test/served-licenses",
    )
).resolve()

LCPL_CONTENT_TYPE = "application/vnd.readium.lcp.license.v1.0+json"


def validate_license_id(value: str) -> str:
    value = unquote(value.strip())

    if not value:
        raise ValueError("missing license id")

    if "/" in value or "\\" in value or ".." in value:
        raise ValueError("unsafe license id")

    if not re.fullmatch(r"[A-Za-z0-9._:-]+", value):
        raise ValueError(f"invalid license id: {value!r}")

    return value


def license_path(license_id: str) -> Path:
    return LICENSE_DIR / f"{license_id}.lcpl"


def http_date(path: Path) -> str:
    return email.utils.formatdate(path.stat().st_mtime, usegmt=True)


class Handler(BaseHTTPRequestHandler):
    server_version = "LicenseFileServer/0.1"

    def send_bytes(self, status: int, body: bytes, content_type: str):
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store, max-age=0")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def send_json(self, status: int, payload: dict):
        body = json.dumps(payload, indent=2).encode("utf-8")
        self.send_bytes(status, body, "application/json; charset=utf-8")

    def do_GET(self):
        parsed = urlparse(self.path)
        parts = parsed.path.strip("/").split("/")

        if parsed.path == "/health":
            return self.send_json(
                200,
                {
                    "status": "ok",
                    "license_dir": str(LICENSE_DIR),
                    "route": "/licenses/{license_id}",
                },
            )

        if len(parts) != 2 or parts[0] != "licenses":
            return self.send_json(404, {"error": "not found"})

        try:
            license_id = validate_license_id(parts[1])
        except ValueError as exc:
            return self.send_json(400, {"error": str(exc)})

        path = license_path(license_id)

        if not path.is_file():
            return self.send_json(
                404,
                {
                    "error": "license not found",
                    "expected_file": str(path),
                },
            )

        body = path.read_bytes()

        self.send_response(200)
        self.send_header("Content-Type", LCPL_CONTENT_TYPE)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store, max-age=0")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Last-Modified", http_date(path))
        self.send_header("Content-Disposition", f'inline; filename="{license_id}.lcpl"')
        self.end_headers()
        self.wfile.write(body)


def main():
    LICENSE_DIR.mkdir(parents=True, exist_ok=True)

    print("License file server")
    print(f"  URL:         http://{HOST}:{PORT}")
    print(f"  License dir: {LICENSE_DIR}")
    print(f"  Route:       /licenses/{{license_id}}")

    ThreadingHTTPServer((HOST, PORT), Handler).serve_forever()


if __name__ == "__main__":
    main()
