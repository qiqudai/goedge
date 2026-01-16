-- lua/balancer.lua
local _M = {}

-- State for Round Robin (per worker)
-- Index: upstream_key -> current_index (integer)
-- Ideally this should be per-upstream-peer logic, but simple RR is O(1)
local rr_state = {}

local function target_weight(target)
    local w = tonumber(target.weight)
    if not w or w <= 0 then
        return 1
    end
    return w
end

local function build_rr_list(targets)
    local list = {}
    for i, t in ipairs(targets) do
        local w = target_weight(t)
        for _ = 1, w do
            table.insert(list, i)
        end
    end
    return list
end

local function targets_hash(targets)
    local parts = {}
    for _, t in ipairs(targets) do
        local addr = ""
        if type(t) == "table" then
            addr = t.addr or ""
        else
            addr = tostring(t or "")
        end
        local weight = type(t) == "table" and t.weight or nil
        table.insert(parts, addr .. ":" .. tostring(weight or ""))
    end
    return table.concat(parts, "|")
end

function _M.get_target(upstream_key, targets)
    if not targets or #targets == 0 then return nil end
    
    -- Single target optimization
    if #targets == 1 then
        return targets[1].addr
    end

    -- Default Policy: Round Robin
    local policy = "round_robin"
    -- If 'targets' table has a policy field (meta info), use it. 
    -- But usually this comes from a separate config arg. 
    -- For simplicity, we'll try to guess or use a passed-in arg if function signature changes.
    -- To keep signature compatible, let's assume `targets.policy` might exist or we just randomness if requested.
    
    -- NOTE: To properly support config-driven policy, we need to read it from `upstream_conf`.
    -- We can see from `access.lua` that we only pass `targets`. 
    -- Let's update the signature to `get_target(upstream_key, targets, policy)`.
    
    -- TEMPORARY: Just randomizing if someone calls with "random" logic in mind, 
    -- but for now updating the code to be ready for the signature change in access.lua.
    
    -- Since I cannot change access.lua in the same tool call, I will handle the logic here generically.
    -- But I'll assume the caller might pass policy later. For now, let's implement the logic blocks.
    
end

function _M.get_target(upstream_key, targets, policy)
    if not targets or #targets == 0 then return nil end
    if #targets == 1 then
        return targets[1]
    end
    
    policy = policy or "round_robin"

    if policy == "random" then
        local total = 0
        for _, t in ipairs(targets) do
            total = total + target_weight(t)
        end
        if total <= 0 then
            return targets[math.random(#targets)]
        end
        local r = math.random(total)
        for _, t in ipairs(targets) do
            r = r - target_weight(t)
            if r <= 0 then
                return t
            end
        end
        return targets[#targets]
        
    elseif policy == "ip_hash" then
        -- Simple hash of remote_addr
        local ip = ngx.var.remote_addr or ""
        local hash = ngx.crc32_short(ip)
        local idx = (hash % #targets) + 1
        return targets[idx]
        
    else -- "round_robin"
        local hash = targets_hash(targets)
        local state = rr_state[upstream_key]
        if not state or state.hash ~= hash then
            state = { list = build_rr_list(targets), index = 0, hash = hash }
            rr_state[upstream_key] = state
        end
        if #state.list == 0 then
            return targets[1]
        end
        state.index = state.index + 1
        if state.index > #state.list then state.index = 1 end
        return targets[state.list[state.index]]
    end
end

return _M
