import json
import urllib.request

TOKEN = "1e53b4aaf19c4feda09cc353f28e4c7e"
URL = "http://127.0.0.1:18081/api/v1/agent/config?node_id=1"

req = urllib.request.Request(URL, headers={"Authorization": "Bearer " + TOKEN})
with urllib.request.urlopen(req) as resp:
    data = json.loads(resp.read().decode("utf-8"))

domain = None
for item in data.get("domains", []):
    if item.get("name") == "wsl-test.example.com":
        domain = item
        break

if not domain:
    print("domain not found")
else:
    print(json.dumps({
        "realtime_send": domain.get("realtime_send"),
        "realtime_return": domain.get("realtime_return"),
        "realtime_identify": domain.get("realtime_identify"),
    }, ensure_ascii=False))
