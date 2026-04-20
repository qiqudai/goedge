import json
import urllib.request
import time

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
    with urllib.request.urlopen(req, timeout=20) as resp:
        raw = resp.read()
    return json.loads(raw.decode('utf-8')) if raw else None

login = request('POST', '/login', {'username': 'admin', 'password': '123456'})
token = login.get('token') if 'token' in login else login.get('data', {}).get('token')

cfg = request('GET', '/global_config', token=token)['data']

waf = cfg.setdefault('waf', {})
waf['enable'] = True
waf['policy'] = ''
waf['mode'] = 'page'
waf['default_block_action'] = ''
waf['temp_whitelist_timeout'] = 0
waf['temp_whitelist_limit_total'] = 0
waf['temp_whitelist_limit_url'] = 0

resp = request('POST', '/global_config', cfg, token)
print(resp)
