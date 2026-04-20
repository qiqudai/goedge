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

login = request('POST', '/login', {'username': 'admin', 'password': '123456'})
if not login:
    raise SystemExit('login failed: empty')

token = None
if 'token' in login:
    token = login['token']
elif isinstance(login.get('data'), dict) and 'token' in login['data']:
    token = login['data']['token']

if not token:
    raise SystemExit(f'login failed: {login}')

site_id = 115
site_resp = request('GET', f'/sites/{site_id}', token=token)
if not site_resp or site_resp.get('data') is None:
    raise SystemExit(f'get site failed: {site_resp}')
site = site_resp['data']['site']
settings = site.get('settings') or {}

origin = settings.get('origin') or {}
origin['list'] = [{'address': '127.0.0.1:8081', 'weight': 10, 'enable': True}]
origin['protocol'] = origin.get('protocol') or 'http'
settings['origin'] = origin

rewrite_rule = {'match': '^/r/(.*)', 'replace': '/real/$1', 'code': 'internal'}
settings['url_rewrites'] = [rewrite_rule]

advanced = settings.get('advanced') or {}
advanced['url_rewrites'] = [rewrite_rule]
settings['advanced'] = advanced

update_payload = {'settings': settings}
update_resp = request('PUT', f'/sites/{site_id}', update_payload, token)
print(update_resp)
