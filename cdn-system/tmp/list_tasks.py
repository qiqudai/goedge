import json
import urllib.parse
import urllib.request

BASE = "http://127.0.0.1:18081/api/v1/admin"


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

params = urllib.parse.urlencode({"page": 1, "pageSize": 50, "type": "config_sync"})
resp = request(f"/tasks?{params}", token=token)
print(json.dumps(resp, ensure_ascii=False))
