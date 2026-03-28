local xdb = require("resty.ip2region.xdb_searcher")

local ip2region = {}
ip2region.__index = ip2region

function ip2region.new(opts)
    local file = opts
    if type(opts) == "table" then
        file = opts.file
    end
    if type(file) ~= "string" or file == "" then
        return nil, "missing ip2region file path"
    end

    local header, err = xdb.load_header(file)
    if err ~= nil then
        return nil, err
    end

    local version, err = xdb.version_from_header(header)
    if err ~= nil then
        return nil, err
    end

    local v_index, err = xdb.load_vector_index(file)
    if err ~= nil then
        return nil, err
    end

    local searcher, err = xdb.new_with_vector_index(version, file, v_index)
    if err ~= nil then
        return nil, err
    end

    return setmetatable({ searcher = searcher }, ip2region)
end

function ip2region:search(ip)
    local res, err = self.searcher:search_by_string(ip)
    if err ~= nil then
        return nil, err
    end
    return res
end

return ip2region
