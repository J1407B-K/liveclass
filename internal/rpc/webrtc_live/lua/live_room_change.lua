-- live_room_change.lua
-- KEYS[1] = countKey   (e.g. "live:room:123:count")
-- KEYS[2] = hashKey    (e.g. "live:room:123:members")
-- ARGV[1] = action ("add" or "del")
-- ARGV[2] = username
-- ARGV[3] = auth (only for add)

local countKey = KEYS[1]
local hashKey  = KEYS[2]
local action   = ARGV[1]
local user     = ARGV[2]
local auth     = ARGV[3]

local function clamp_nonneg(n)
  if n < 0 then return 0 end
  return n
end

if action == "add" then
  -- 幂等：如果已经在 hash 里，就不重复 +1
  local existed = redis.call("HEXISTS", hashKey, user)
  redis.call("HSET", hashKey, user, auth)
  if existed == 0 then
    local newCount = redis.call("INCRBY", countKey, 1)
    if newCount < 0 then
      redis.call("SET", countKey, 0)
      newCount = 0
    end
    return newCount
  else
    -- 已存在：人数不变，返回当前人数
    local cur = tonumber(redis.call("GET", countKey)) or 0
    return clamp_nonneg(cur)
  end

elseif action == "del" then
  -- 幂等：如果本来就不在 hash 里，就不 -1
  local existed = redis.call("HEXISTS", hashKey, user)
  redis.call("HDEL", hashKey, user)
  if existed == 1 then
    local newCount = redis.call("INCRBY", countKey, -1)
    if newCount < 0 then
      redis.call("SET", countKey, 0)
      newCount = 0
    end
    return newCount
  else
    local cur = tonumber(redis.call("GET", countKey)) or 0
    return clamp_nonneg(cur)
  end

else
  return redis.error_reply("invalid action: " .. tostring(action))
end