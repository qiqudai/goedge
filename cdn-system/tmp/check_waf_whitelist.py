import json
from pathlib import Path

path = Path("/usr/local/goedge/nodes/edge-node/conf/cdn_config.json")
print("exists", path.exists())
if path.exists():
    data = json.loads(path.read_text())
    waf = data.get("waf") or {}
    print("whitelist_ips", waf.get("whitelist_ips"))
