import urllib.request

url = "http://127.0.0.1:18080/rt-test"
req = urllib.request.Request(url, method="GET")
req.add_header("Host", "wsl-test.example.com")
req.add_header("User-Agent", "")
try:
    with urllib.request.urlopen(req, timeout=5) as resp:
        print(resp.status)
except Exception as e:
    print("error", e)
