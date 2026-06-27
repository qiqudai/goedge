#!/usr/bin/env python3
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    def _echo_request(self):
        names = ["Upgrade", "Connection", "Expect", "TE", "Trailer", "Proxy-Connection", "Keep-Alive"]
        lines = []
        for name in names:
            lines.append(f"{name}: {self.headers.get(name, '')}")
        body = ("\n".join(lines) + "\n").encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(body)

    def _normal(self):
        body = b"origin-ok\n"
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Upgrade", "h2,h2c")
        self.send_header("Connection", "Upgrade")
        self.send_header("Keep-Alive", "timeout=5")
        self.send_header("Proxy-Connection", "keep-alive")
        self.send_header("TE", "trailers")
        self.send_header("Trailer", "Expires")
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(body)

    def _websocket(self):
        self.send_response(101)
        self.send_header("Upgrade", "websocket")
        self.send_header("Connection", "Upgrade")
        self.send_header("Sec-WebSocket-Accept", "test")
        self.end_headers()

    def do_HEAD(self):
        if self.path == "/ws":
            self._websocket()
            return
        if self.path == "/echo-request":
            self._echo_request()
            return
        self._normal()

    def do_GET(self):
        if self.path == "/ws":
            self._websocket()
            return
        if self.path == "/echo-request":
            self._echo_request()
            return
        self._normal()

    def log_message(self, fmt, *args):
        return


if __name__ == "__main__":
    server = ThreadingHTTPServer(("0.0.0.0", 8080), Handler)
    server.serve_forever()
