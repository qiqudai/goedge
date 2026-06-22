local _M = {}

local function runtime_root()
    local prefix = ngx.config.prefix() or ""
    if prefix ~= "" and prefix:sub(-1) ~= "/" then
        prefix = prefix .. "/"
    end
    if prefix ~= "" then
        return prefix:sub(1, -2)
    end
    local root = os.getenv("CDN_RUNTIME_ROOT")
    if root and root ~= "" then
        return root
    end
    return "/usr/local/openresty/nginx"
end

local function read_file(path)
    local file, err = io.open(path, "rb")
    if not file then
        return nil, err
    end
    local content = file:read("*a")
    file:close()
    return content
end

local function read_json(path)
    local content, err = read_file(path)
    if not content then
        return nil, err
    end
    local cjson = require "cjson.safe"
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

local function resolve_lang(site_lang, i18n, accept_language)
    local default_lang = normalize_locale(i18n.default_lang or "zh-CN")
    local enabled_langs = i18n.enabled_langs or { default_lang }
    site_lang = normalize_locale(site_lang or "")
    if site_lang ~= "" and site_lang ~= "browser" then
        return site_lang
    end
    if site_lang == "browser" then
        return parse_accept_language(accept_language, enabled_langs, default_lang)
    end
    local mode = i18n.lang_mode or "browser"
    if mode == "browser" then
        return parse_accept_language(accept_language, enabled_langs, default_lang)
    end
    return default_lang
end

local function extract_error_code(uri)
    if not uri then
        return nil
    end
    return string.match(uri, "/__cdn_error/([^%.]+)%.html")
end

function _M.serve()
    local uri = ngx.var.uri or ""
    local code = extract_error_code(uri)
    if not code or code == "" then
        ngx.status = 404
        ngx.say("error page not found")
        return
    end

    local root = runtime_root()
    local i18n, err = read_json(root .. "/conf/error_page_i18n.json")
    if not i18n then
        ngx.log(ngx.ERR, "load error_page_i18n.json failed: ", err or "unknown")
        i18n = { default_lang = "zh-CN", lang_mode = "browser", enabled_langs = { "zh-CN" } }
    end

    local site_lang = ngx.var.cdn_error_lang or ""
    local accept_language = ngx.var.http_accept_language or ""
    local lang = resolve_lang(site_lang, i18n, accept_language)
    if lang == "" then
        lang = i18n.default_lang or "zh-CN"
    end

    local candidates = { lang }
    local base = string.match(lang, "^([^-]+)")
    if base and base ~= lang then
        candidates[#candidates + 1] = base
    end
    candidates[#candidates + 1] = i18n.default_lang or "zh-CN"

    local content
    for _, candidate in ipairs(candidates) do
        local path = string.format("%s/conf/error_pages/%s/%s.html", root, candidate, code)
        content = read_file(path)
        if content then
            break
        end
    end
    if not content then
        ngx.status = 404
        ngx.say("error page not found")
        return
    end

    local client_ip = ngx.var.cdn_client_ip or ngx.var.remote_addr or ""
    local node_ip = ngx.var.server_addr or ""
    local host = ngx.var.host or ""
    local request_id = ngx.var.request_id or ngx.var.connection or ""
    content = string.gsub(content, "{client_ip}", client_ip)
    content = string.gsub(content, "{node_ip}", node_ip)
    content = string.gsub(content, "{host}", host)
    content = string.gsub(content, "{request_id}", tostring(request_id))

    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.say(content)
end

return _M
