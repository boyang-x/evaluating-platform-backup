#!/usr/bin/env python3
"""Local development reverse proxy for Docker containers.

Docker Desktop containers on Windows can fail to reach some corporate VPN
targets directly even when the host can. This helper lets containers call
http://host.docker.internal:<port> while the host process forwards to the real
HTTPS target without using the host proxy environment.
"""

from __future__ import annotations

import argparse
import http.server
import socketserver
import sys
import urllib.error
import urllib.request


class ReverseProxyHandler(http.server.BaseHTTPRequestHandler):
    target_base = ""

    def log_message(self, fmt: str, *args: object) -> None:
        sys.stderr.write("[local-target-proxy] " + fmt % args + "\n")

    def _handle(self) -> None:
        body = None
        if self.command in {"POST", "PUT", "PATCH"}:
            body = self.rfile.read(int(self.headers.get("Content-Length", "0") or "0"))
        target_url = self.target_base.rstrip("/") + self.path
        headers = {
            key: value
            for key, value in self.headers.items()
            if key.lower() not in {"host", "connection", "proxy-connection", "accept-encoding"}
        }
        request = urllib.request.Request(target_url, data=body, headers=headers, method=self.command)
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        try:
            response = opener.open(request, timeout=180)
            self._write_response(response.status, response.headers, response.read())
        except urllib.error.HTTPError as exc:
            self._write_response(exc.code, exc.headers, exc.read())
        except Exception as exc:
            self.send_response(502)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.end_headers()
            self.wfile.write(str(exc).encode("utf-8", "replace"))

    def _write_response(self, status: int, headers: object, body: bytes) -> None:
        self.send_response(status)
        for key, value in headers.items():
            if key.lower() not in {"transfer-encoding", "connection"}:
                self.send_header(key, value)
        self.end_headers()
        self.wfile.write(body)

    do_DELETE = _handle
    do_GET = _handle
    do_PATCH = _handle
    do_POST = _handle
    do_PUT = _handle


class ThreadingHTTPServer(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--listen", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=48761)
    parser.add_argument("--target", default="https://aip.b.qianxin-inc.cn")
    args = parser.parse_args()
    ReverseProxyHandler.target_base = args.target
    server = ThreadingHTTPServer((args.listen, args.port), ReverseProxyHandler)
    print(f"forwarding http://{args.listen}:{args.port} -> {args.target}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
