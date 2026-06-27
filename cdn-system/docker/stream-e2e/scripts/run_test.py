#!/usr/bin/env python3
import socket
import sys
import time

HOST = "edge"
TCP_PORT = 19001
UDP_PORT = 19002


def wait_tcp():
    for _ in range(30):
        try:
            sock = socket.create_connection((HOST, TCP_PORT), timeout=1)
            sock.close()
            return
        except OSError:
            time.sleep(0.5)
    raise RuntimeError("edge tcp port not ready")


def test_tcp():
    payload = b"hello-tcp"
    with socket.create_connection((HOST, TCP_PORT), timeout=5) as sock:
        sock.sendall(payload)
        data = sock.recv(4096)
    if data != payload:
        raise RuntimeError(f"tcp echo mismatch: {data!r}")


def test_udp():
    payload = b"hello-udp"
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(5)
    try:
        sock.sendto(payload, (HOST, UDP_PORT))
        data, _ = sock.recvfrom(4096)
    finally:
        sock.close()
    if data != payload:
        raise RuntimeError(f"udp echo mismatch: {data!r}")


def main():
    print("[stream-e2e] waiting for edge...")
    wait_tcp()
    print("[stream-e2e] TCP forward test")
    test_tcp()
    print("[stream-e2e] UDP forward test")
    test_udp()
    print("[stream-e2e] all tests passed")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"[stream-e2e] failed: {exc}", file=sys.stderr)
        sys.exit(1)
