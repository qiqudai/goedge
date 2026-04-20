import json
import urllib.request
from pathlib import Path

BASE = "http://127.0.0.1:18081/api/v1/admin"
BACKUP_PATH = Path("/mnt/e/cdn/goedge/cdn-system/tmp/waf_whitelist_backup.json")


def request(path, method="GET", data=None, token=None):
    url = BASE + path
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    payload = None
    if data is not None:
        payload = json.dumps(data).encode("utf-8")
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode("utf-8"))


login = request("/login", method="POST", data={"username": "admin", "password": "123456"})
login_data = login.get("data") or login

token = login_data.get("token")
if not token:
    raise SystemExit("login failed: " + json.dumps(login, ensure_ascii=False))

cfg_resp = request("/global_config", token=token)
config = cfg_resp.get("data") or cfg_resp
if not isinstance(config, dict):
    raise SystemExit("unexpected config response: " + json.dumps(cfg_resp, ensure_ascii=False))

waf = config.get("waf") or {}
whitelist_raw = waf.get("whitelist_ips") or ""

backup = {
    "whitelist_ips": whitelist_raw,
}
BACKUP_PATH.write_text(json.dumps(backup, ensure_ascii=False), encoding="utf-8")

lines = [line.strip() for line in str(whitelist_raw).splitlines() if line.strip()]
if "127.0.0.1" not in lines:
    lines.append("127.0.0.1")

waf["whitelist_ips"] = "\n".join(lines)
config["waf"] = waf

save_resp = request("/global_config", method="POST", data=config, token=token)
print(json.dumps(save_resp, ensure_ascii=False))
