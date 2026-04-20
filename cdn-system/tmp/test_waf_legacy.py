import json
import time
import urllib.request
import urllib.error
import subprocess
import copy
from pathlib import Path

API_BASE = 'http://127.0.0.1:18081/api/v1/admin'
EDGE_BASE = 'http://127.0.0.1:18080'
BACKUP_PATH = Path('/mnt/e/cdn/goedge/cdn-system/tmp/global_config_backup.json')


def api_request(method, path, data=None, token=None):
    url = API_BASE + path
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
    if not raw:
        return None
    return json.loads(raw.decode('utf-8'))


def edge_request(path, headers=None):
    url = EDGE_BASE + path
    req = urllib.request.Request(url, headers=headers or {}, method='GET')
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, resp.read().decode('utf-8', errors='replace')
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8', errors='replace')
    except Exception as e:
        return None, str(e)


def login():
    login_resp = api_request('POST', '/login', {'username': 'admin', 'password': '123456'})
    if not login_resp:
        raise RuntimeError('login failed: empty')
    if 'token' in login_resp:
        return login_resp['token']
    if isinstance(login_resp.get('data'), dict) and 'token' in login_resp['data']:
        return login_resp['data']['token']
    raise RuntimeError(f'login failed: {login_resp}')


def update_global_config(token, cfg):
    resp = api_request('POST', '/global_config', cfg, token)
    return resp


def start_echo_server():
    return subprocess.Popen(['python3', '/tmp/echo_server.py'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def stop_process(proc):
    if not proc:
        return
    try:
        proc.terminate()
        proc.wait(timeout=2)
    except Exception:
        try:
            proc.kill()
        except Exception:
            pass


def main():
    token = login()
    cfg_resp = api_request('GET', '/global_config', token=token)
    if not cfg_resp or 'data' not in cfg_resp:
        raise RuntimeError(f'get global_config failed: {cfg_resp}')
    backup = cfg_resp['data']
    BACKUP_PATH.write_text(json.dumps(backup, ensure_ascii=False), encoding='utf-8')

    # Test 1: mode -> default_block_action (page)
    cfg1 = copy.deepcopy(backup)
    waf1 = cfg1.setdefault('waf', {})
    waf1['enable'] = True
    waf1['policy'] = ''
    waf1['mode'] = 'page'
    waf1['default_block_action'] = ''
    waf1['temp_whitelist_timeout'] = 0
    waf1['temp_whitelist_limit_total'] = 0
    waf1['temp_whitelist_limit_url'] = 0
    update_global_config(token, cfg1)
    time.sleep(1.5)

    proc = start_echo_server()
    time.sleep(0.5)
    status, body = edge_request('/index.html', headers={'Host': 'wsl-test.example.com', 'User-Agent': 'sqlmap'})
    stop_process(proc)
    print('mode_map_status', status)

    # Test 2: log_only (should pass through)
    cfg2 = copy.deepcopy(cfg1)
    cfg2['waf']['policy'] = 'log_only'
    update_global_config(token, cfg2)
    time.sleep(1.5)

    proc = start_echo_server()
    time.sleep(0.5)
    status2, body2 = edge_request('/index.html', headers={'Host': 'wsl-test.example.com', 'User-Agent': 'sqlmap'})
    stop_process(proc)
    print('log_only_status', status2, 'body_prefix', body2.strip()[:20])

    # Test 3: cc legacy mapping -> default page protection
    cfg3 = copy.deepcopy(backup)
    waf3 = cfg3.setdefault('waf', {})
    waf3['enable'] = True
    waf3['policy'] = ''
    waf3['mode'] = ''
    waf3['default_block_action'] = 'disconnect'
    waf3['default_page_protection'] = ''
    waf3['default_page_protection_threshold'] = 0
    waf3['anti_cc_type'] = ''
    waf3['cc'] = {
        'enable': True,
        'threshold': 1,
        'action': '5s',
        'block_timeout': 60,
        'emergency_mode': False,
        'slide_count': 0,
    }
    update_global_config(token, cfg3)
    time.sleep(1.5)

    proc = start_echo_server()
    time.sleep(0.5)
    status3, body3 = edge_request('/index.html', headers={'Host': 'wsl-test.example.com'})
    stop_process(proc)
    print('cc_map_status', status3, 'body_prefix', body3.strip()[:20])

    # Restore backup
    update_global_config(token, backup)
    time.sleep(1.5)

if __name__ == '__main__':
    main()
