#!/usr/bin/env python3
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

_lock = threading.Lock()
_total = 0
_by_path = {}


def record(path: str) -> None:
    global _total
    with _lock:
        _total += 1
        _by_path[path] = _by_path.get(path, 0) + 1


def snapshot() -> dict:
    with _lock:
        return {"total": _total, "paths": dict(_by_path)}


class Handler(BaseHTTPRequestHandler):
    def _ok(self):
        if self.path == "/_origin/stats":
            body = json.dumps(snapshot()).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return True
        record(self.path)
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"origin-ok\n")
        return True

    def do_GET(self):
        self._ok()

    def do_POST(self):
        self._ok()

    def log_message(self, fmt, *args):
        return


if __name__ == "__main__":
    server = ThreadingHTTPServer(("0.0.0.0", 8080), Handler)
    server.serve_forever()
