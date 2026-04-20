import json
from pathlib import Path

path = Path("/usr/local/goedge/nodes/edge-node/conf/cdn_config.json")
if not path.exists():
    print("missing")
    raise SystemExit(1)

data = json.loads(path.read_text())
for d in data.get("domains", []):
    if d.get("name") == "wsl-test.example.com":
        print(json.dumps({"waf_enable": d.get("waf_enable")}, ensure_ascii=False))
        break
else:
    print("not found")
