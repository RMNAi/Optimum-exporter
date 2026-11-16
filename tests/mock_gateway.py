#!/usr/bin/env python3
"""Test-only mock of the gateway /health endpoint. Not part of the monitoring path.
Run from repo root: python3 tests/mock_gateway.py [port]"""
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

FIXTURES = {
    "/stream-only": (200, "tests/fixtures/health_stream_only.json"),
    "/cl-healthy":  (200, "tests/fixtures/health_cl_healthy.json"),
    "/stall":       (503, "tests/fixtures/health_stall.json"),
    "/cl-fail":     (503, "tests/fixtures/health_cl_fail.json"),
}

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        status, path = FIXTURES.get(self.path, (404, None))
        body = open(path, "rb").read() if path else b"{}"
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 8000), H).serve_forever()