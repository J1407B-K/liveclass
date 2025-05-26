-- live_room_user_hash.lua
-- KEYS[1] = hashKey (e.g. "live:room:123:members")
-- ARGV[1] = action ("add" or "del")
-- ARGV[2] = username
-- ARGV[3] = auth (仅当 action="add" 时使用)

local hashKey  = KEYS[1]
local action   = ARGV[1]
local user     = ARGV[2]
local auth     = ARGV[3]

if action == "add" then
    -- 添加或更新用户
    redis.call("HSET", hashKey, user, auth)
    return "OK"
elseif action == "del" then
    -- 删除用户
    redis.call("HDEL", hashKey, user)
    return "OK"
else
    return redis.error_reply("invalid action: " .. tostring(action))
end
