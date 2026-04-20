import json
import urllib.request

BASE = "http://127.0.0.1:18081/api/v1/admin"
SITE_ID = 115


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

site_resp = request(f"/sites/{SITE_ID}", token=token)
site_data = site_resp.get("data", {})
site = site_data.get("site") or site_resp.get("site") or site_resp
settings = site.get("settings") or {}

settings["realtime_send"] = False
settings["realtime_return"] = False
settings["realtime_identify"] = False

adv = settings.get("advanced") or {}
adv["realtime_send"] = False
adv["realtime_return"] = False
adv["realtime_identify"] = False
settings["advanced"] = adv

update_resp = request(f"/sites/{SITE_ID}", method="PUT", data={"settings": settings}, token=token)
print(json.dumps(update_resp, ensure_ascii=False))
