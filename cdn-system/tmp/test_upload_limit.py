import json
import urllib.request
import urllib.error

url = "http://127.0.0.1:18080/upload-limit"
body = json.dumps({"data": "b" * (350 * 1024)}).encode("utf-8")
req = urllib.request.Request(url, data=body, method="POST")
req.add_header("Host", "wsl-test.example.com")
req.add_header("User-Agent", "limit-test")
req.add_header("Content-Type", "application/json")
try:
    with urllib.request.urlopen(req, timeout=5) as resp:
        print(resp.status)
except urllib.error.HTTPError as e:
    print(e.code)
