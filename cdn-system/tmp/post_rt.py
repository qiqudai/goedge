import json
import urllib.request

url = "http://127.0.0.1:18080/rt-test"
body = json.dumps({"msg": "rt-test"}).encode("utf-8")
req = urllib.request.Request(url, data=body, method="POST")
req.add_header("Host", "wsl-test.example.com")
req.add_header("User-Agent", "rt-test")
req.add_header("Content-Type", "application/json")
try:
    with urllib.request.urlopen(req, timeout=5) as resp:
        print(resp.status)
except Exception as e:
    print("error", e)
