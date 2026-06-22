-- lua/guard.lua
-- NOTE: This file should stay consistent with `edge-node/lua/guard.lua`.
local cjson = require "cjson.safe"

-- Ensure bundled libs under lua/lib are discoverable (agent assets layout).
if not string.find(package.path, "lua/lib/?.lua", 1, true) then
    package.path = package.path .. ";lua/lib/?.lua"
end

local aes = require "resty.aes"
local md5 = require "resty.md5"
local random = require "resty.random"
local str = require "resty.string"
local bit = require "bit"

local _M = {}

local COOKIE_GUARD = "guard"
local COOKIE_GUARD_RET = "guardret"
local COOKIE_GUARD_PASS = "__cdn_guard_pass"
local COOKIE_GUARD_STATE = "__cdn_guard_state"
local COOKIE_GUARD_BROWSER_ID = "__cdn_guard_bid"
local COOKIE_GUARD_FINGERPRINT = "__cdn_guard_fp"

local STATE_TTL = 10 * 60
local PASS_TTL = 30 * 60
local FIVE_SECONDS_DELAY = 5
local MAX_ATTEMPTS = 5
local ROTATE_TOLERANCE_DEG = 10

local function now()
    return ngx.time()
end

local function is_https()
    return ngx.var.https == "on" or ngx.var.scheme == "https"
end

local function guard_pass_ttl()
    local ctx = ngx.ctx
    if ctx and ctx.guard_pass_ttl then
        local ttl = tonumber(ctx.guard_pass_ttl)
        if ttl and ttl > 0 then
            return ttl
        end
    end
    return PASS_TTL
end

local function guard_block_ttl()
    local ctx = ngx.ctx
    if ctx and ctx.guard_block_ttl then
        local ttl = tonumber(ctx.guard_block_ttl)
        if ttl and ttl > 0 then
            return ttl
        end
    end
    return 0
end

local function guard_cookie_domain()
    local ctx = ngx.ctx
    if ctx and ctx.guard_cookie_domain and ctx.guard_cookie_domain ~= "" then
        return ctx.guard_cookie_domain
    end
    return nil
end

local function waf_config()
    local cfg = _G.cdn_config
    if cfg and cfg.waf then
        return cfg.waf
    end
    return nil
end

local function guard_debug_enabled()
    local waf = waf_config()
    return waf and waf.anti_cc_debug == true
end

local function guard_debug_log(...)
    if guard_debug_enabled() then
        ngx.log(ngx.INFO, ...)
    end
end

local function guard_custom_image_url()
    local waf = waf_config()
    if not waf then
        return ""
    end
    if waf.anti_cc_image_source ~= "custom" then
        return ""
    end
    local url = tostring(waf.anti_cc_image_custom_url or "")
    if url == "" then
        return ""
    end
    return url
end

local function cookie(name)
    return ngx.var["cookie_" .. name]
end

local function normalize_cookie_token(value, max_len)
    if type(value) ~= "string" then
        return ""
    end
    local token = string.match(value, "^([%w_%-]+)") or ""
    if max_len and #token > max_len then
        token = string.sub(token, 1, max_len)
    end
    return token
end

local function cookie_values(name)
    local header = ngx.var.http_cookie
    local values = {}
    if type(header) ~= "string" or header == "" then
        return values
    end
    for item in string.gmatch(header, "([^;]+)") do
        local k, v = string.match(item, "^%s*([^=]+)=([^;]*)")
        if k == name then
            table.insert(values, v)
        end
    end
    return values
end

local function add_set_cookie(value)
    local existing = ngx.header["Set-Cookie"]
    if not existing then
        ngx.header["Set-Cookie"] = { value }
        return
    end
    if type(existing) ~= "table" then
        ngx.header["Set-Cookie"] = { existing, value }
        return
    end
    table.insert(existing, value)
    ngx.header["Set-Cookie"] = existing
end

local function set_cookie(name, value, opts)
    opts = opts or {}
    if not opts.domain then
        local domain = guard_cookie_domain()
        if domain then
            opts.domain = domain
        end
    end
    local parts = { name .. "=" .. (value or ""), "Path=" .. (opts.path or "/") }
    if opts.domain then
        table.insert(parts, "Domain=" .. opts.domain)
    end
    if opts.max_age ~= nil then
        table.insert(parts, "Max-Age=" .. tostring(opts.max_age))
    end
    if opts.http_only then
        table.insert(parts, "HttpOnly")
    end
    if opts.secure then
        table.insert(parts, "Secure")
    end
    if opts.same_site then
        table.insert(parts, "SameSite=" .. opts.same_site)
    end
    add_set_cookie(table.concat(parts, "; "))
end

local function clear_cookie(name, http_only)
    set_cookie(name, "", {
        path = "/",
        max_age = 0,
        http_only = http_only == true,
        secure = is_https(),
        same_site = "Lax",
    })
end

local function config_prefix()
    local prefix = ngx.config.prefix() or ""
    if prefix ~= "" and prefix:sub(-1) ~= "/" then
        prefix = prefix .. "/"
    end
    return prefix
end

local function guard_dir()
    return config_prefix() .. "conf/guard/"
end

local function conf_dir()
    return config_prefix() .. "conf/"
end

local guard_i18n_cache
local error_page_i18n_cache
local file_cache = {}
-- Forward declaration: read_file is defined later in this file, but
-- read_json/preload_assets reference it earlier. Without this local
-- declaration the earlier references bind to a nil global and abort.
local read_file
local secret
local GUARD_TEMPLATE_FILES = {
    "browser_verify_auto.html",
    "delay_jump.html",
    "click.html",
    "slide.html",
    "captcha.html",
    "rotate.html",
}

function _M.preload_assets()
    if file_cache._ready then
        return
    end
    file_cache._ready = true
    for _, tpl in ipairs(GUARD_TEMPLATE_FILES) do
        local path = guard_dir() .. tpl
        local f = io.open(path, "rb")
        if f then
            file_cache[path] = f:read("*a")
            f:close()
        else
            ngx.log(ngx.WARN, "guard preload missing template: ", path)
        end
    end
    for _, rel in ipairs({ "guard_i18n.json", "error_page_i18n.json" }) do
        local path = conf_dir() .. rel
        local f = io.open(path, "rb")
        if f then
            file_cache[path] = f:read("*a")
            f:close()
        else
            ngx.log(ngx.WARN, "guard preload missing config: ", path)
        end
    end
    local captcha_list = guard_dir() .. "captcha_list.txt"
    local f = io.open(captcha_list, "rb")
    if f then
        file_cache[captcha_list] = f:read("*a")
        f:close()
    end
end

local function read_json(path)
    local content = read_file(path)
    if not content then
        return nil
    end
    return cjson.decode(content)
end

local function normalize_locale(lang)
    if not lang or lang == "" then
        return ""
    end
    lang = string.gsub(lang, "_", "-")
    local parts = {}
    for part in string.gmatch(lang, "[^-]+") do
        parts[#parts + 1] = part
    end
    if #parts == 0 then
        return ""
    end
    parts[1] = string.lower(parts[1])
    for i = 2, #parts do
        if #parts[i] == 2 then
            parts[i] = string.upper(parts[i])
        else
            parts[i] = string.lower(parts[i])
        end
    end
    return table.concat(parts, "-")
end

local function parse_accept_language(header, enabled_langs, default_lang)
    if not header or header == "" then
        return default_lang
    end
    local enabled = {}
    for _, lang in ipairs(enabled_langs or {}) do
        enabled[normalize_locale(lang)] = true
        local base = string.match(lang, "^([^-]+)")
        if base then
            enabled[normalize_locale(base)] = true
        end
    end
    local best_lang = default_lang
    local best_q = -1
    for chunk in string.gmatch(header, "[^,]+") do
        local lang_part = string.match(chunk, "^%s*([^;]+)")
        local q = tonumber(string.match(chunk, "q=([0-9%.]+)")) or 1
        if lang_part then
            lang_part = normalize_locale(string.gsub(lang_part, "^%s*(.-)%s*$", "%1"))
            if enabled[lang_part] and q > best_q then
                best_lang = lang_part
                best_q = q
            else
                local base = string.match(lang_part, "^([^-]+)")
                base = normalize_locale(base)
                if base ~= "" and enabled[base] and q > best_q then
                    best_lang = base
                    best_q = q
                end
            end
        end
    end
    return best_lang
end

local function load_error_page_i18n()
    if error_page_i18n_cache then
        return error_page_i18n_cache
    end
    local i18n = read_json(conf_dir() .. "error_page_i18n.json")
    if not i18n then
        i18n = { default_lang = "zh-CN", lang_mode = "browser", enabled_langs = { "zh-CN", "en" } }
    end
    error_page_i18n_cache = i18n
    return i18n
end

local function resolve_guard_lang()
    local i18n = load_error_page_i18n()
    local default_lang = normalize_locale(i18n.default_lang or "zh-CN")
    local enabled_langs = i18n.enabled_langs or { default_lang }
    local site_lang = normalize_locale(ngx.var.cdn_error_lang or "")
    if site_lang ~= "" and site_lang ~= "browser" then
        return site_lang
    end
    if site_lang == "browser" then
        return parse_accept_language(ngx.var.http_accept_language or "", enabled_langs, default_lang)
    end
    local mode = i18n.lang_mode or "browser"
    if mode == "browser" then
        return parse_accept_language(ngx.var.http_accept_language or "", enabled_langs, default_lang)
    end
    return default_lang
end

local function load_guard_i18n()
    if guard_i18n_cache then
        return guard_i18n_cache
    end
    local data = read_json(conf_dir() .. "guard_i18n.json")
    guard_i18n_cache = data or {}
    return guard_i18n_cache
end

local function strings_key_for_type(filter_type)
    if filter_type == "five_seconds" then
        return "delay_jump"
    end
    if filter_type == "click_captcha" or filter_type == "click_captcha_simple" then
        return "click"
    end
    if filter_type == "slide_captcha" or filter_type == "slide_captcha_simple" then
        return "slide"
    end
    if filter_type == "captcha" then
        return "captcha"
    end
    if filter_type == "rotate_captcha" then
        return "rotate"
    end
    return "click"
end

local function resolve_guard_strings(filter_type)
    local lang = resolve_guard_lang()
    local i18n = load_error_page_i18n()
    local default_lang = normalize_locale(i18n.default_lang or "zh-CN")
    if lang == "" then
        lang = default_lang
    end
    local data = load_guard_i18n()
    local key = strings_key_for_type(filter_type)
    local by_lang = (data.strings or {})[key] or {}
    local candidates = { lang }
    local base = string.match(lang, "^([^-]+)")
    if base and base ~= lang then
        candidates[#candidates + 1] = base
    end
    candidates[#candidates + 1] = default_lang
    candidates[#candidates + 1] = "zh-CN"
    candidates[#candidates + 1] = "en"
    local strings
    for _, candidate in ipairs(candidates) do
        strings = by_lang[candidate]
        if strings then
            lang = candidate
            break
        end
    end
    strings = strings or by_lang["zh-CN"] or by_lang["en"] or {}
    local out = {}
    for k, v in pairs(strings) do
        out[k] = v
    end
    out.html_lang = lang
    return out
end

local PLACEHOLDER_PATTERN = "{{([a-zA-Z0-9_]+)}}"

local function render_guard_template(content, strings)
    if not content then
        return content
    end
    return string.gsub(content, PLACEHOLDER_PATTERN, function(key)
        local value = strings[key]
        if value == nil then
            return "{{" .. key .. "}}"
        end
        return value
    end)
end

read_file = function(path)
    if not path or path == "" then
        return nil
    end
    if file_cache[path] then
        return file_cache[path]
    end
    local f = io.open(path, "rb")
    if f then
        local content = f:read("*a")
        f:close()
        if content and content ~= "" then
            file_cache[path] = content
            return content
        end
    end
    local prefix = config_prefix()
    local guard_base = prefix .. "conf/guard/"
    if path:sub(1, #guard_base) == guard_base then
        local rel = path:sub(#guard_base + 1)
        local res = ngx.location.capture("/_guard/" .. rel)
        if res and res.status == 200 and res.body and res.body ~= "" then
            file_cache[path] = res.body
            return res.body
        end
        return nil
    end
    local conf_base = prefix .. "conf/"
    if path:sub(1, #conf_base) == conf_base then
        local rel = path:sub(#conf_base + 1)
        local res = ngx.location.capture("/_cdn_conf/" .. rel)
        if res and res.status == 200 and res.body and res.body ~= "" then
            file_cache[path] = res.body
            return res.body
        end
        return nil
    end
    return nil
end

local function md5_bin(s)
    local m = md5:new()
    if not m then
        return nil
    end
    m:update(s)
    return m:final()
end

local function aes_for_nonce(nonce8)
    local key = md5_bin(secret() .. "|" .. tostring(nonce8 or ""))
    if not key then
        return nil, "md5 failed"
    end
    return aes:new(key, nil, aes.cipher(128, "cbc"), { iv = key })
end

local function rand_hex(nbytes)
    local bytes = random.bytes(nbytes, true)
    if not bytes then
        return str.to_hex(tostring(math.random()) .. tostring(now()))
    end
    return str.to_hex(bytes)
end

local function store()
    return ngx.shared.guard_store or ngx.shared.waf_cache
end

local function state_key(nonce8)
    return "guard:st:" .. nonce8
end

local function load_state(nonce8)
    local s = store()
    if not s then
        return nil
    end
    local raw = s:get(state_key(nonce8))
    if not raw or type(raw) ~= "string" then
        return nil
    end
    return cjson.decode(raw)
end

local function save_state(nonce8, st, ttl)
    local s = store()
    if not s then
        return false
    end
    local raw = cjson.encode(st)
    if not raw then
        return false
    end
    return s:set(state_key(nonce8), raw, ttl or STATE_TTL)
end

local function delete_state(nonce8)
    local s = store()
    if not s then
        return
    end
    s:delete(state_key(nonce8))
end

local function normalize_type(filter_type)
    local t = string.lower(tostring(filter_type or ""))
    if t == "5s" or t == "5s_shield" or t == "shield_5s" or t == "five_seconds" or t == "delay_jump" or t == "delay_jump_filter" then
        return "five_seconds"
    end
    if t == "invisible" or t == "silent_captcha" or t == "browser_verify_auto" or t == "302" or t == "302_challenge" then
        return "silent_captcha"
    end
    if t == "click" or t == "click_captcha" or t == "click_filter" then
        return "click_captcha"
    end
    if t == "click_simple" or t == "click_captcha_simple" then
        return "click_captcha_simple"
    end
    if t == "slide" or t == "slide_captcha" or t == "slide_filter" then
        return "slide_captcha"
    end
    if t == "slide_simple" or t == "slide_captcha_simple" then
        return "slide_captcha_simple"
    end
    if t == "captcha_filter" then
        return "captcha"
    end
    if t == "rotate" or t == "rotate_captcha" or t == "rotate_filter" then
        return "rotate_captcha"
    end
    return t
end

function _M.is_guard_filter(filter_type)
    local t = normalize_type(filter_type)
    return t == "silent_captcha"
        or t == "five_seconds"
        or t == "click_captcha"
        or t == "click_captcha_simple"
        or t == "slide_captcha"
        or t == "slide_captcha_simple"
        or t == "captcha"
        or t == "rotate_captcha"
end

function _M.is_guard_request(uri)
    if not uri or uri == "" then
        return false
    end
    if string.sub(uri, 1, 7) == "/_guard" then
        return true
    end
    if string.sub(uri, 1, 11) == "/_cdn_conf/" then
        return true
    end
    if string.sub(uri, 1, 19) == "/@cdn_guard_render/" then
        return true
    end
    return false
end

local COMMON_NON_BROWSER_UA = {
    "curl",
    "wget",
    "python-requests",
    "python-urllib",
    "go-http-client",
    "httpie",
    "postmanruntime",
    "okhttp",
    "apache-httpclient",
    "libwww-perl",
    "scrapy",
    "aiohttp",
    "node-fetch",
    "undici",
    "axios/",
    "java/",
    "ruby",
    "php/",
    "powershell",
}

function _M.is_common_non_browser_request()
    local ua = string.lower(tostring(ngx.var.http_user_agent or ""))
    if ua == "" then
        return true, "ua=empty"
    end
    for _, token in ipairs(COMMON_NON_BROWSER_UA) do
        if string.find(ua, token, 1, true) then
            return true, "ua=" .. token
        end
    end
    return false, ""
end

local cached_secret

secret = function()
    local cfg = _G.cdn_config
    if cfg and cfg.waf and type(cfg.waf.secret_key) == "string" and cfg.waf.secret_key ~= "" then
        cached_secret = cfg.waf.secret_key
        return cached_secret
    end
    if cached_secret then
        return cached_secret
    end

    local s = store()
    if s then
        local v = s:get("guard:secret")
        if type(v) == "string" and v ~= "" then
            cached_secret = v
            return v
        end
        v = rand_hex(32)
        s:set("guard:secret", v)
        cached_secret = v
        return v
    end

    cached_secret = rand_hex(32)
    return cached_secret
end

local function hmac_hex(data)
    local sig = ngx.hmac_sha1(secret(), data)
    if not sig then
        return ""
    end
    return str.to_hex(sig)
end

local function constant_time_eq(a, b)
    if type(a) ~= "string" or type(b) ~= "string" then
        return false
    end
    if #a ~= #b then
        return false
    end
    local diff = 0
    for i = 1, #a do
        diff = bit.bxor(diff, bit.bxor(a:byte(i), b:byte(i)))
    end
    return diff == 0
end

local function valid_browser_id(value)
    return type(value) == "string" and value:match("^[a-f0-9]{32}$") ~= nil
end

local function valid_fingerprint(value)
    return type(value) == "string" and value:match("^[a-f0-9]{16}$") ~= nil
end

local function browser_binding()
    local browser_id = normalize_cookie_token(cookie(COOKIE_GUARD_BROWSER_ID), 64)
    local fingerprint = normalize_cookie_token(cookie(COOKIE_GUARD_FINGERPRINT), 128)
    local ua_sig = string.sub(hmac_hex("ua|" .. tostring(ngx.var.http_user_agent or "")), 1, 24)
    return browser_id, fingerprint, ua_sig
end

local function pass_state_key(host, ip, filter_type, filter_id, browser_id, fingerprint, ua_sig)
    local raw = table.concat({
        host or "",
        ip or "",
        normalize_type(filter_type),
        tostring(filter_id or 0),
        browser_id or "",
        fingerprint or "",
        ua_sig or "",
    }, "|")
    local digest = md5_bin(raw)
    if digest then
        return "guard:pass:" .. str.to_hex(digest)
    end
    return "guard:pass:" .. hmac_hex(raw)
end

local function b64url_encode(value)
    local encoded = ngx.encode_base64(value or "") or ""
    encoded = string.gsub(encoded, "+", "-")
    encoded = string.gsub(encoded, "/", "_")
    encoded = string.gsub(encoded, "=+$", "")
    return encoded
end

local function b64url_decode(value)
    if type(value) ~= "string" or value == "" then
        return nil
    end
    value = string.gsub(value, "-", "+")
    value = string.gsub(value, "_", "/")
    local mod = #value % 4
    if mod > 0 then
        value = value .. string.rep("=", 4 - mod)
    end
    return ngx.decode_base64(value)
end

local function state_payload(st)
    return cjson.encode(st)
end

local function sign_state_payload(payload)
    return hmac_hex("state|" .. tostring(payload or ""))
end

local function state_cookie_value(st)
    local payload = state_payload(st)
    if not payload then
        return nil
    end
    return "v1." .. b64url_encode(payload) .. "." .. sign_state_payload(payload)
end

local function validate_state(st, host, ip, filter_type, filter_id)
    if type(st) ~= "table" then
        return false
    end
    if st.host ~= host or st.ip ~= ip then
        return false
    end
    if st.type ~= filter_type or tostring(st.filter_id or 0) ~= tostring(filter_id or 0) then
        return false
    end
    local issued = tonumber(st.issued) or 0
    if issued <= 0 or issued + STATE_TTL < now() then
        return false
    end
    return true
end

local function load_state_cookie(host, ip, filter_type, filter_id)
    local value = cookie(COOKIE_GUARD_STATE)
    if type(value) ~= "string" or value == "" then
        return nil
    end
    local version, payload64, sig = string.match(value, "^(v1)%.([^%.]+)%.([0-9a-f]+)$")
    if version ~= "v1" or not payload64 or not sig then
        return nil
    end
    local payload = b64url_decode(payload64)
    if type(payload) ~= "string" or payload == "" then
        return nil
    end
    if not constant_time_eq(sig, sign_state_payload(payload)) then
        return nil
    end
    local st = cjson.decode(payload)
    if validate_state(st, host, ip, filter_type, filter_id) then
        return st
    end
    return nil
end

local function load_state_cookie_by_nonce(nonce8)
    local value = cookie(COOKIE_GUARD_STATE)
    if type(value) ~= "string" or value == "" then
        return nil
    end
    local version, payload64, sig = string.match(value, "^(v1)%.([^%.]+)%.([0-9a-f]+)$")
    if version ~= "v1" or not payload64 or not sig then
        return nil
    end
    local payload = b64url_decode(payload64)
    if type(payload) ~= "string" or payload == "" then
        return nil
    end
    if not constant_time_eq(sig, sign_state_payload(payload)) then
        return nil
    end
    local st = cjson.decode(payload)
    if type(st) ~= "table" or st.nonce ~= nonce8 then
        return nil
    end
    local issued = tonumber(st.issued) or 0
    if issued <= 0 or issued + STATE_TTL < now() then
        return nil
    end
    return st
end

local function save_state_cookie(st)
    local value = state_cookie_value(st)
    if not value then
        return false
    end
    set_cookie(COOKIE_GUARD_STATE, value, {
        path = "/",
        max_age = STATE_TTL,
        http_only = true,
        secure = is_https(),
        same_site = "Lax",
    })
    return true
end

local function clear_state_cookie()
    clear_cookie(COOKIE_GUARD_STATE, true)
end

local function verify_pass_cookie_value(value, host, ip, filter_type, filter_id)
    if type(value) ~= "string" or value == "" then
        return false
    end
    local parts = {}
    for item in string.gmatch(value, "([^|]+)") do
        table.insert(parts, item)
        if #parts > 12 then
            break
        end
    end
    if #parts < 11 or parts[1] ~= "v3" then
        return false
    end
    local exp = tonumber(parts[2]) or 0
    if exp <= now() then
        return false
    end
    if parts[3] ~= (host or "") or parts[4] ~= (ip or "") then
        return false
    end
    if parts[5] ~= normalize_type(filter_type) then
        return false
    end
    if parts[6] ~= tostring(filter_id or 0) then
        return false
    end
    local browser_id, fingerprint, ua_sig = browser_binding()
    if not valid_browser_id(browser_id) or not valid_fingerprint(fingerprint) then
        return false
    end
    if parts[7] ~= browser_id or parts[8] ~= fingerprint or parts[9] ~= ua_sig then
        return false
    end
    local token_id = parts[10]
    if token_id == "" then
        return false
    end
    local payload = table.concat({ parts[1], parts[2], parts[3], parts[4], parts[5], parts[6], parts[7], parts[8], parts[9], parts[10] }, "|")
    local expected = hmac_hex(payload)
    if not constant_time_eq(parts[11], expected) then
        return false
    end
    return true
end

local function verify_pass_cookie(host, ip, filter_type, filter_id)
    local values = cookie_values(COOKIE_GUARD_PASS)
    if #values == 0 then
        local value = cookie(COOKIE_GUARD_PASS)
        if value then
            table.insert(values, value)
        end
    end
    for _, value in ipairs(values) do
        if verify_pass_cookie_value(value, host, ip, filter_type, filter_id) then
            return true
        end
    end
    return false
end

local function build_pass_cookie(host, ip, filter_type, filter_id, ttl)
    local browser_id, fingerprint, ua_sig = browser_binding()
    if not valid_browser_id(browser_id) or not valid_fingerprint(fingerprint) then
        return nil, "browser binding missing"
    end
    local pass_ttl = ttl or PASS_TTL
    local exp = now() + pass_ttl
    local token_id = rand_hex(16)
    local payload = table.concat(
        {
            "v3",
            tostring(exp),
            host or "",
            ip or "",
            normalize_type(filter_type),
            tostring(filter_id or 0),
            browser_id,
            fingerprint,
            ua_sig,
            token_id,
        },
        "|"
    )
    local sig = hmac_hex(payload)
    local record = cjson.encode({
        token = token_id,
        browser_id = browser_id,
        fingerprint = fingerprint,
        ua_sig = ua_sig,
        host = host or "",
        ip = ip or "",
        type = normalize_type(filter_type),
        filter_id = tostring(filter_id or 0),
        issued = now(),
    })
    local s = store()
    if s then
        s:set(pass_state_key(host, ip, filter_type, filter_id, browser_id, fingerprint, ua_sig), record or token_id, pass_ttl)
    end
    return payload .. "|" .. sig
end

local function parse_guard_nonce(value)
    if type(value) ~= "string" or #value < 8 then
        return nil
    end
    return string.sub(value, 1, 8)
end

local function build_guard_cookie_value(nonce8, filter_type, issued_at)
    local a, err = aes_for_nonce(nonce8)
    if not a then
        return nonce8
    end
    local plaintext = tostring(issued_at or now())
    if normalize_type(filter_type) == "five_seconds" then
        plaintext = rand_hex(5) .. plaintext -- delay_jump.js reads substr(10)
    end
    local enc, enc_err = a:encrypt(plaintext)
    if not enc then
        ngx.log(ngx.WARN, "guard encrypt failed: ", enc_err or "")
        return nonce8
    end
    return nonce8 .. rand_hex(2) .. (ngx.encode_base64(enc) or "")
end

local function ensure_state(filter, host, ip)
    local filter_type = normalize_type(filter.type or filter.Type or "")
    local filter_id = filter.id or filter.ID or 0

    local guard_value = cookie(COOKIE_GUARD)
    local nonce8 = parse_guard_nonce(guard_value)
    if nonce8 then
        local cookie_state = load_state_cookie(host, ip, filter_type, filter_id)
        if cookie_state and cookie_state.nonce == nonce8 then
            return nonce8, cookie_state, guard_value
        end
        local st = load_state(nonce8)
        if st and st.host == host and st.ip == ip and st.type == filter_type and st.filter_id == filter_id then
            return nonce8, st, guard_value
        end
    end

    nonce8 = rand_hex(4)
    local issued_at = now()
    local st = { nonce = nonce8, host = host, ip = ip, type = filter_type, filter_id = filter_id, issued = issued_at, attempts = 0 }
    if filter_type == "rotate_captcha" then
        local group = (tonumber(rand_hex(1), 16) or 1) % 30 + 1
        local degree = (tonumber(rand_hex(2), 16) or 15) % 331 + 15
        st.rotate = { file = string.format("%d-%d.jpeg", group, degree), degree = degree, answer = (360 - degree) % 360 }
    end
    save_state(nonce8, st, STATE_TTL)
    save_state_cookie(st)

    guard_value = build_guard_cookie_value(nonce8, filter_type, issued_at)
    set_cookie(COOKIE_GUARD, guard_value, { path = "/", max_age = STATE_TTL, secure = is_https(), same_site = "Lax" })
    clear_cookie(COOKIE_GUARD_RET, false)
    return nonce8, st, guard_value
end

local function decrypt_guardret(nonce8, guardret_value)
    local a, err = aes_for_nonce(nonce8)
    if not a then
        return nil, err
    end
    local cipher_bin = ngx.decode_base64(guardret_value or "")
    if not cipher_bin then
        return nil, "base64 decode failed"
    end
    return a:decrypt(cipher_bin)
end

local function abs_diff_mod360(a, b)
    local d = math.abs((a or 0) - (b or 0)) % 360
    if d > 180 then
        d = 360 - d
    end
    return d
end

local function validate_guardret(filter_type, nonce8, st, guardret_value)
    if filter_type == "captcha" then
        local expected = st and st.captcha and st.captcha.code or ""
        local got = string.upper(string.gsub(tostring(guardret_value or ""), "%s+", ""))
        expected = string.upper(string.gsub(tostring(expected or ""), "%s+", ""))
        return expected ~= "" and got ~= "" and got == expected, "captcha mismatch"
    end

    local plain, err = decrypt_guardret(nonce8, guardret_value)
    if not plain then
        return false, err
    end
    local text = tostring(plain or "")

    if filter_type == "silent_captcha" or filter_type == "five_seconds" then
        if now() - (tonumber(st.issued) or 0) < 300 then
            return false, "auto verify too fast"
        end
        local n = tonumber(text) or 0
        local expected = (tonumber(st.issued) or 0) + 10
        if n ~= expected then
            return false, "ret mismatch"
        end
        if filter_type == "five_seconds" and now()-(tonumber(st.issued) or 0) < FIVE_SECONDS_DELAY then
            return false, "too fast"
        end
        return true
    end

    local payload = cjson.decode(text)
    if type(payload) ~= "table" then
        return false, "invalid payload"
    end

    if filter_type == "click_captcha" or filter_type == "click_captcha_simple" then
        if now() - (tonumber(st.issued) or 0) < 800 then
            return false, "click too fast"
        end
        local x, y, a = tonumber(payload.x), tonumber(payload.y), tonumber(payload.a)
        if not x or not y or not a or a ~= x + y then
            return false, "click invalid"
        end
        if x < 0 or y < 0 or x > 4096 or y > 4096 then
            return false, "click out of range"
        end
        return true
    end

    if filter_type == "slide_captcha" or filter_type == "slide_captcha_simple" then
        local move = payload.move
        if type(move) ~= "table" or #move < 3 then
            return false, "slide invalid"
        end
        local first_ts = tonumber(move[1].timestamp) or 0
        local last_ts = tonumber(move[#move].timestamp) or 0
        if first_ts <= 0 or last_ts <= 0 or last_ts-first_ts < 500 then
            return false, "slide too fast"
        end
        local slider = tonumber(payload.slider) or 0
        local btn = tonumber(payload.btn) or 0
        local start_x = tonumber(move[1].x) or 0
        local end_x = tonumber(move[#move].x) or 0
        local expected = slider - btn
        if slider <= 0 or btn <= 0 or expected <= 0 or end_x - start_x < expected - 3 then
            return false, "slide distance"
        end
        local prev_x = start_x
        for i = 2, #move do
            local x = tonumber(move[i].x) or 0
            if x < prev_x - 2 then
                return false, "slide non-monotonic"
            end
            prev_x = x
        end
        return true
    end

    if filter_type == "rotate_captcha" then
        local deg = tonumber(payload.deg)
        if not deg or not st.rotate then
            return false, "rotate invalid"
        end
        deg = deg % 360
        local answer = tonumber(st.rotate.answer) or 0
        if abs_diff_mod360(deg, answer) <= ROTATE_TOLERANCE_DEG then
            return true
        end
        return false, "rotate mismatch"
    end

    return false, "unsupported type"
end

function _M.ensure_passed(filter, host, ip)
    local filter_type = normalize_type(filter.type or filter.Type or "")
    local filter_id = filter.id or filter.ID or 0

    if verify_pass_cookie(host, ip, filter_type, filter_id) then
        return true
    end

    local guardret_value = cookie(COOKIE_GUARD_RET)
    if not guardret_value or guardret_value == "" then
        return false
    end

    local nonce8, st = ensure_state(filter, host, ip)
    if not nonce8 or not st then
        return false
    end

    local ok = false
    ok = validate_guardret(filter_type, nonce8, st, guardret_value)
    if ok == true then
        guard_debug_log("guard passed host=", host or "", " ip=", ip or "", " type=", filter_type)
        local pass_ttl = guard_pass_ttl()
        local pass_value, pass_err = build_pass_cookie(host, ip, filter_type, filter_id, pass_ttl)
        if not pass_value then
            guard_debug_log("guard pass build failed host=", host or "", " ip=", ip or "", " type=", filter_type, " err=", pass_err or "")
            clear_cookie(COOKIE_GUARD_RET, false)
            return false
        end
        set_cookie(COOKIE_GUARD_PASS, pass_value, {
            path = "/",
            max_age = pass_ttl,
            http_only = true,
            secure = is_https(),
            same_site = "Lax",
        })
        clear_cookie(COOKIE_GUARD_RET, false)
        clear_cookie(COOKIE_GUARD, false)
        clear_state_cookie()
        delete_state(nonce8)
        return true
    end
    guard_debug_log("guard failed host=", host or "", " ip=", ip or "", " type=", filter_type)

    st.attempts = (tonumber(st.attempts) or 0) + 1
    if st.attempts >= MAX_ATTEMPTS then
        local block_ttl = guard_block_ttl()
        if block_ttl > 0 then
            local block_list = ngx.shared.ip_blacklist
            if block_list then
                block_list:set(ip, true, block_ttl)
            end
        end
        delete_state(nonce8)
        clear_cookie(COOKIE_GUARD, false)
        clear_state_cookie()
    else
        save_state(nonce8, st, STATE_TTL)
        save_state_cookie(st)
    end
    clear_cookie(COOKIE_GUARD_RET, false)
    return false
end

local function template_for_type(filter_type)
    if filter_type == "silent_captcha" then
        return "browser_verify_auto.html"
    end
    if filter_type == "five_seconds" then
        return "delay_jump.html"
    end
    if filter_type == "click_captcha" or filter_type == "click_captcha_simple" then
        return "click.html"
    end
    if filter_type == "slide_captcha" or filter_type == "slide_captcha_simple" then
        return "slide.html"
    end
    if filter_type == "captcha" then
        return "captcha.html"
    end
    if filter_type == "rotate_captcha" then
        return "rotate.html"
    end
    return "click.html"
end

function _M.serve_challenge_by_nonce(nonce8)
    if not nonce8 or nonce8 == "" then
        ngx.status = ngx.HTTP_INTERNAL_SERVER_ERROR
        ngx.say("<html><body>Guard challenge missing</body></html>")
        return
    end
    local st = load_state_cookie_by_nonce(nonce8) or load_state(nonce8)
    if not st then
        ngx.status = ngx.HTTP_INTERNAL_SERVER_ERROR
        ngx.say("<html><body>Guard challenge missing</body></html>")
        return
    end
    local filter = { type = st.type, id = st.filter_id }
    local filter_type = normalize_type(st.type or "")
    local tpl = template_for_type(filter_type)
    local content = read_file(guard_dir() .. tpl)
    if not content then
        ngx.say("<html><body>Guard template missing</body></html>")
        return
    end
    local custom_url = guard_custom_image_url()
    if custom_url ~= "" then
        if filter_type == "rotate_captcha" then
            content = string.gsub(content, "/_guard/rotate_image", custom_url)
        elseif filter_type == "captcha" then
            content = string.gsub(content, "/_guard/captcha%.png", custom_url)
        end
    end
    content = render_guard_template(content, resolve_guard_strings(filter_type))
    ngx.header["Content-Type"] = "text/html; charset=utf-8"
    ngx.header["Cache-Control"] = "no-store"
    ngx.print(content)
end

function _M.challenge(filter, host, ip)
    local filter_type = normalize_type(filter.type or filter.Type or "")
    local nonce8 = ensure_state(filter, host, ip)
    guard_debug_log("guard challenge host=", host or "", " ip=", ip or "", " type=", filter_type)

    if not nonce8 or nonce8 == "" then
        ngx.status = ngx.HTTP_INTERNAL_SERVER_ERROR
        ngx.say("<html><body>Guard challenge missing</body></html>")
        return
    end
    _M.serve_challenge_by_nonce(nonce8)
end

local captcha_codes
local function load_captcha_codes()
    if captcha_codes then
        return captcha_codes
    end
    local content = read_file(guard_dir() .. "captcha_list.txt")
    local list = {}
    if content then
        for line in string.gmatch(content, "([^\r\n]+)") do
            line = string.match(line, "^%s*(.-)%s*$")
            if line ~= "" then
                table.insert(list, line)
            end
        end
    end
    captcha_codes = list
    return captcha_codes
end

local function rand_index(max)
    if max <= 0 then
        return 1
    end
    local bytes = random.bytes(4, true)
    local n = 0
    if bytes and #bytes >= 4 then
        n = bytes:byte(1) * 16777216 + bytes:byte(2) * 65536 + bytes:byte(3) * 256 + bytes:byte(4)
    else
        n = math.random(0, 0x7fffffff)
    end
    return (n % max) + 1
end

function _M.serve_captcha_png()
    local guard_value = cookie(COOKIE_GUARD)
    local nonce8 = parse_guard_nonce(guard_value)
    if not nonce8 then
        ngx.status = 404
        return
    end
    local st = load_state(nonce8)
    if not st or st.type ~= "captcha" then
        ngx.status = 404
        return
    end
    local list = load_captcha_codes()
    if not list or #list == 0 then
        ngx.status = 404
        return
    end
    local code = list[rand_index(#list)]
    st.captcha = { code = code, at = now() }
    save_state(nonce8, st, STATE_TTL)
    save_state_cookie(st)

    local img = read_file(guard_dir() .. "captcha/" .. code .. ".png")
    if not img then
        ngx.status = 404
        return
    end
    ngx.header["Content-Type"] = "image/png"
    ngx.header["Cache-Control"] = "no-store"
    ngx.print(img)
end

function _M.serve_rotate_image()
    local guard_value = cookie(COOKIE_GUARD)
    local nonce8 = parse_guard_nonce(guard_value)
    if not nonce8 then
        ngx.status = 404
        return
    end
    local st = load_state_cookie_by_nonce(nonce8) or load_state(nonce8)
    if not st or st.type ~= "rotate_captcha" or not st.rotate or not st.rotate.file then
        ngx.status = 404
        return
    end
    if ngx.var.arg_r and ngx.var.arg_r ~= "" then
        local group = (tonumber(rand_hex(1), 16) or 1) % 30 + 1
        local degree = (tonumber(rand_hex(2), 16) or 15) % 331 + 15
        st.rotate = { file = string.format("%d-%d.jpeg", group, degree), degree = degree, answer = (360 - degree) % 360 }
        st.attempts = 0
        save_state(nonce8, st, STATE_TTL)
        save_state_cookie(st)
    end

    local img = read_file(guard_dir() .. "rotate/" .. st.rotate.file)
    if not img then
        ngx.status = 404
        return
    end
    ngx.header["Content-Type"] = "image/jpeg"
    ngx.header["Cache-Control"] = "no-store"
    ngx.print(img)
end

return _M
