local countKey = KEYS[1]
local memberKey = KEYS[2]

-- 一次性删除两个 Key，返回被删除的键数量
local deleted = redis.call("DEL", countKey, memberKey)
return deleted