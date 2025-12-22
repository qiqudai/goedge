-- lua/balancer.lua
local _M = {}

-- State for Round Robin (per worker)
-- Index: upstream_key -> current_index (integer)
-- Ideally this should be per-upstream-peer logic, but simple RR is O(1)
local rr_state = {}

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
    if #targets == 1 then return targets[1].addr end
    
    policy = policy or "round_robin"

    if policy == "random" then
        return targets[math.random(#targets)].addr
        
    elseif policy == "ip_hash" then
        -- Simple hash of remote_addr
        local ip = ngx.var.remote_addr or ""
        local hash = ngx.crc32_short(ip)
        local idx = (hash % #targets) + 1
        return targets[idx].addr
        
    else -- "round_robin"
        local state_idx = rr_state[upstream_key] or 0
        state_idx = state_idx + 1
        if state_idx > #targets then state_idx = 1 end
        rr_state[upstream_key] = state_idx
        return targets[state_idx].addr
    end
end

return _M
