import json
import urllib.request

base = 'http://127.0.0.1:18081/api/v1/admin'

def request(method, path, data=None, token=None):
    url = base + path
    headers = {}
    if token:
        headers['Authorization'] = f'Bearer {token}'
    body = None
    if data is not None:
        body = json.dumps(data).encode('utf-8')
        headers['Content-Type'] = 'application/json'
    req = urllib.request.Request(url, data=body, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=15) as resp:
        raw = resp.read()
    if not raw:
        return None
    return json.loads(raw.decode('utf-8'))

login = request('POST', '/login', {'username':'admin','password':'123456'})
if not login:
    raise SystemExit('login failed')

token = None
if 'token' in login:
    token = login['token']
elif isinstance(login.get('data'), dict) and 'token' in login['data']:
    token = login['data']['token']

cfg = request('GET', '/global_config', token=token)
print(cfg['data']['waf'].get('enable'), cfg['data']['waf'].get('policy'), cfg['data']['waf'].get('mode'))
