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

local function read_file(path)
    local f = io.open(path, "rb")
    if not f then
        return nil
    end
    local content = f:read("*a")
    f:close()
    return content
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
    local key = md5_bin(nonce8)
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
    if t == "5s" or t == "5s_shield" or t == "shield_5s" or t == "five_seconds" then
        return "five_seconds"
    end
    if t == "invisible" or t == "silent_captcha" then
        return "silent_captcha"
    end
    if t == "click" or t == "click_captcha" then
        return "click_captcha"
    end
    if t == "click_simple" or t == "click_captcha_simple" then
        return "click_captcha_simple"
    end
    if t == "slide" or t == "slide_captcha" then
        return "slide_captcha"
    end
    if t == "slide_simple" or t == "slide_captcha_simple" then
        return "slide_captcha_simple"
    end
    if t == "rotate" or t == "rotate_captcha" then
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
    return string.sub(uri, 1, 7) == "/_guard"
end

local cached_secret

local function secret()
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

local function verify_pass_cookie(value, host, ip, filter_type, filter_id)
    if type(value) ~= "string" or value == "" then
        return false
    end
    local parts = {}
    for item in string.gmatch(value, "([^|]+)") do
        table.insert(parts, item)
        if #parts > 8 then
            break
        end
    end
    if #parts < 6 then
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

    if parts[1] == "v1" then
        local payload = table.concat({ parts[1], parts[2], parts[3], parts[4], parts[5] }, "|")
        local expected = hmac_hex(payload)
        return constant_time_eq(parts[6], expected)
    end

    if parts[1] ~= "v2" then
        return false
    end
    if #parts < 7 then
        return false
    end
    if parts[6] ~= tostring(filter_id or 0) then
        return false
    end
    local payload = table.concat({ parts[1], parts[2], parts[3], parts[4], parts[5], parts[6] }, "|")
    local expected = hmac_hex(payload)
    return constant_time_eq(parts[7], expected)
end

local function build_pass_cookie(host, ip, filter_type, filter_id, ttl)
    local exp = now() + (ttl or PASS_TTL)
    local payload = table.concat(
        { "v2", tostring(exp), host or "", ip or "", normalize_type(filter_type), tostring(filter_id or 0) },
        "|"
    )
    local sig = hmac_hex(payload)
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
        local st = load_state(nonce8)
        if st and st.host == host and st.ip == ip and st.type == filter_type and st.filter_id == filter_id then
            return nonce8, st, guard_value
        end
    end

    nonce8 = rand_hex(4)
    local issued_at = now()
    local st = { host = host, ip = ip, type = filter_type, filter_id = filter_id, issued = issued_at, attempts = 0 }
    if filter_type == "rotate_captcha" then
        local group = (tonumber(rand_hex(1), 16) or 1) % 30 + 1
        local degree = (tonumber(rand_hex(2), 16) or 15) % 331 + 15
        st.rotate = { file = string.format("%d-%d.jpeg", group, degree), degree = degree, answer = (360 - degree) % 360 }
    end
    save_state(nonce8, st, STATE_TTL)

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
        local x, y, a = tonumber(payload.x), tonumber(payload.y), tonumber(payload.a)
        if not x or not y or not a or a ~= x + y then
            return false, "click invalid"
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
        if first_ts > 0 and last_ts > 0 and last_ts-first_ts < 300 then
            return false, "slide too fast"
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
        local degree = tonumber(st.rotate.degree) or 0
        if abs_diff_mod360(deg, answer) <= ROTATE_TOLERANCE_DEG or abs_diff_mod360(deg, degree) <= ROTATE_TOLERANCE_DEG then
            return true
        end
        return false, "rotate mismatch"
    end

    return false, "unsupported type"
end

function _M.ensure_passed(filter, host, ip)
    local filter_type = normalize_type(filter.type or filter.Type or "")
    local filter_id = filter.id or filter.ID or 0

    local pass_value = cookie(COOKIE_GUARD_PASS)
    if verify_pass_cookie(pass_value, host, ip, filter_type, filter_id) then
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
        set_cookie(COOKIE_GUARD_PASS, build_pass_cookie(host, ip, filter_type, filter_id, pass_ttl), {
            path = "/",
            max_age = pass_ttl,
            http_only = true,
            secure = is_https(),
            same_site = "Lax",
        })
        clear_cookie(COOKIE_GUARD_RET, false)
        clear_cookie(COOKIE_GUARD, false)
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
    else
        save_state(nonce8, st, STATE_TTL)
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

function _M.challenge(filter, host, ip)
    local filter_type = normalize_type(filter.type or filter.Type or "")
    ensure_state(filter, host, ip)
    guard_debug_log("guard challenge host=", host or "", " ip=", ip or "", " type=", filter_type)

    ngx.header["Content-Type"] = "text/html; charset=utf-8"
    ngx.header["Cache-Control"] = "no-store"

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
    ngx.print(content)
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
    local st = load_state(nonce8)
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
