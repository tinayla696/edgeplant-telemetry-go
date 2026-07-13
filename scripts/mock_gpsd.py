#!/usr/bin/env python3
import json
import socket
import threading
import time

HOST = "0.0.0.0"
PORT = 2947
TPV = {
    "class": "TPV",
    "device": "/dev/ttyMOCK0",
    "mode": 3,
    "time": "2024-06-10T15:30:30Z",
    "lat": 35.6812,
    "lon": 139.7671,
    "alt": 24.5,
    "speed": 10.0,
    "climb": 0.0,
    "track": 180.0,
}
VERSION = {"class": "VERSION", "release": "mock-gpsd", "rev": "1.0", "proto_major": 3, "proto_minor": 25}

def handle_client(conn: socket.socket) -> None:
    with conn:
        conn.sendall((json.dumps(VERSION) + "\n").encode())
        buffer = b""
        watch_enabled = False
        while True:
            data = conn.recv(4096)
            if not data:
                return
            buffer += data
            text = buffer.decode(errors="ignore")
            if "?WATCH={\"enable\":true,\"json\":true}" in text:
                watch_enabled = True
                buffer = b""
            if watch_enabled:
                for _ in range(20):
                    conn.sendall((json.dumps(TPV) + "\n").encode())
                    time.sleep(0.2)
                return

def main() -> None:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind((HOST, PORT))
    sock.listen()
    while True:
        conn, _ = sock.accept()
        threading.Thread(target=handle_client, args=(conn,), daemon=True).start()

if __name__ == "__main__":
    main()
