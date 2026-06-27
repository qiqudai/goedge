#!/usr/bin/env python3
import socket
import threading

TCP_PORT = 9000
UDP_PORT = 9001


def tcp_echo():
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind(("0.0.0.0", TCP_PORT))
    sock.listen(32)
    while True:
        conn, _ = sock.accept()
        threading.Thread(target=handle_tcp, args=(conn,), daemon=True).start()


def handle_tcp(conn):
    with conn:
        data = conn.recv(4096)
        conn.sendall(data or b"tcp-ok")


def udp_echo():
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind(("0.0.0.0", UDP_PORT))
    while True:
        data, addr = sock.recvfrom(4096)
        sock.sendto(data or b"udp-ok", addr)


if __name__ == "__main__":
    threading.Thread(target=udp_echo, daemon=True).start()
    tcp_echo()
