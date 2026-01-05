import json
import re
import sys
import urllib.request


def request_json(url: str, method: str = "GET", headers=None, body_obj=None):
    headers = dict(headers or {})
    data = None
    if body_obj is not None:
        data = json.dumps(body_obj, ensure_ascii=False).encode("utf-8")
        headers.setdefault("Content-Type", "application/json; charset=utf-8")
    req = urllib.request.Request(url, method=method, headers=headers, data=data)
    with urllib.request.urlopen(req) as resp:
        raw = resp.read()
    return json.loads(raw.decode("utf-8"))


def normalize_error_pages(pages: dict) -> dict:
    out = {}

    def copy_if_present(dst_key: str, src_key: str):
        v = pages.get(src_key)
        if isinstance(v, str) and v:
            out[dst_key] = v

    for k in [
        "400",
        "403",
        "502",
        "504",
        "traffic_limit",
        "site_locked",
        "domain_invalid",
        "conn_limit",
        "timeout",
        "ip",
    ]:
        copy_if_present(k, k)

    legacy_map = {
        "p400": "400",
        "p403": "403",
        "p502": "502",
        "p504": "504",
        "p512": "timeout",
        "p513": "traffic_limit",
        "p514": "site_locked",
        "p515": "conn_limit",
        "access_ip_not_allow": "ip",
        "host_not_found": "domain_invalid",
    }
    for legacy_key, dst_key in legacy_map.items():
        if dst_key in out:
            continue
        copy_if_present(dst_key, legacy_key)

    return out


def has_han(s: str) -> bool:
    return bool(re.search(r"[\u4e00-\u9fff]", s or ""))


def main():
    base = "http://127.0.0.1:8080/api/v1/admin"
    user = "admin"
    password = "123456"

    login = request_json(
        base + "/login", method="POST", body_obj={"username": user, "password": password}
    )
    token = login["token"]
    headers = {"Authorization": "Bearer " + token}

    cfg_items = request_json(base + "/config_items?type=error_page", headers=headers)
    legacy_item = None
    for item in cfg_items.get("list", []):
        if item.get("name") == "error-page":
            legacy_item = item
            break
    if not legacy_item:
        print("error: config_items(type=error_page) missing name=error-page", file=sys.stderr)
        return 2

    legacy_pages = json.loads(legacy_item.get("value", "") or "{}")
    normalized = normalize_error_pages(legacy_pages)
    if not normalized:
        print("error: normalized error_pages empty", file=sys.stderr)
        return 2

    # sanity: ensure we actually have Chinese somewhere; otherwise we might just overwrite with junk
    if not any(has_han(v) for v in normalized.values() if isinstance(v, str)):
        print(
            "error: normalized pages contain no Chinese (Han) characters; aborting",
            file=sys.stderr,
        )
        return 2

    current = request_json(base + "/global_config", headers=headers)
    data = current.get("data") or {}
    data["error_pages"] = normalized

    request_json(base + "/global_config", method="POST", headers=headers, body_obj=data)

    # Verify: refetch and print a couple of titles.
    after = request_json(base + "/global_config", headers=headers)
    ep = (after.get("data") or {}).get("error_pages") or {}
    for k in ["403", "traffic_limit", "timeout", "ip"]:
        v = ep.get(k, "")
        m = re.search(r"(?is)<title>(.*?)</title>", v or "")
        title = (m.group(1).strip() if m else "")[:80]
        print(f"{k}: title={title!r} has_han={has_han(v)}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())

