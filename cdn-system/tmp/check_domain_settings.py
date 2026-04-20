import json
from pathlib import Path

path = Path("/usr/local/goedge/nodes/edge-node/conf/cdn_config.json")
if not path.exists():
    print("missing")
    raise SystemExit(1)

data = json.loads(path.read_text())
for d in data.get("domains", []):
    if d.get("name") == "wsl-test.example.com":
        out = {
            "log_request_header": d.get("log_request_header"),
            "log_response_header": d.get("log_response_header"),
            "log_request_body": d.get("log_request_body"),
            "log_request_body_size_limit": d.get("log_request_body_size_limit"),
            "realtime_send": d.get("realtime_send"),
            "realtime_return": d.get("realtime_return"),
            "realtime_identify": d.get("realtime_identify"),
            "default_site": d.get("default_site"),
            "body_limit": d.get("body_limit"),
            "cache_rules": d.get("cache", {}).get("rules"),
        }
        print(json.dumps(out, ensure_ascii=False))
        break
else:
    print("not found")
