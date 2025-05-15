-- live_room_count.lua
-- KEYS[1]：直播间人数计数键，例如 "live:room:123:count"
-- ARGV[1]：delta，整数，正数为加，负数为减

local key   = KEYS[1]
local delta = tonumber(ARGV[1])

-- 原子增减
local newCount = redis.call("INCRBY", key, delta)

-- 保证非负
if newCount < 0 then
    redis.call("SET", key, 0)
    newCount = 0
end

-- 返回最新人数
return newCount
