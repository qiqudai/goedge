import json
import time
import urllib.request
from pathlib import Path

EDGE = "http://127.0.0.1:18080"
HOST = "wsl-test.example.com"
LOG_PATH = Path("/usr/local/goedge/nodes/edge-node/logs/access.json")


def request(path, method="GET", body=None, ua=None):
    url = EDGE + path
    data = None
    if body is not None:
        if isinstance(body, (bytes, bytearray)):
            data = body
        else:
            data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Host", HOST)
    if ua is not None:
        req.add_header("User-Agent", ua)
    if data is not None:
        req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            return resp.status, resp.read().decode("utf-8")
    except Exception as e:
        return "error", str(e)


def tail_entries(keyword, max_items=3):
    if not LOG_PATH.exists():
        return []
    entries = []
    with LOG_PATH.open("r", encoding="utf-8", errors="ignore") as f:
        for line in f:
            if keyword in line:
                try:
                    entries.append(json.loads(line))
                except Exception:
                    continue
    return entries[-max_items:]

# 1) WAF disabled check: empty UA should not be 418
status, _ = request("/rt-test", method="GET", ua="")

# 2) request/response header + body logging
status_post, _ = request("/rt-test", method="POST", body={"msg": "rt-test"}, ua="rt-test")

# 3) cache rule test (.jpg)
request("/test.jpg", method="GET", ua="cache-test")
request("/test.jpg", method="GET", ua="cache-test")

# 4) body log size limit (80KB)
body_80k = {"data": "a" * (80 * 1024)}
request("/body-test", method="POST", body=body_80k, ua="body-test")

# 5) upload limit (350KB -> expect 413)
large_body = json.dumps({"data": "b" * (350 * 1024)}).encode("utf-8")
status_large, _ = request("/upload-limit", method="POST", body=large_body, ua="limit-test")

# wait for logs flush
time.sleep(0.5)

rt_entries = tail_entries("/rt-test", 1)
cache_entries = tail_entries("/test.jpg", 2)
body_entries = tail_entries("/body-test", 1)

out = {
    "waf_disable_status": status,
    "rt_post_status": status_post,
    "upload_limit_status": status_large,
    "rt_log": rt_entries[-1] if rt_entries else None,
    "cache_logs": cache_entries,
    "body_log_len": len((body_entries[-1] or {}).get("cdn_req_body", "")) if body_entries else None,
}
print(json.dumps(out, ensure_ascii=False))
