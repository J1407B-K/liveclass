-- 返回一个数组，第一项是 count（字符串），后续为所有 member 元素

local countKey  = KEYS[1]
local memberKey = KEYS[2]

-- 1. 获取 count（如果 key 不存在，则返回 nil，可在客户端处理）
local count = redis.call("GET", countKey)

-- 2. 获取所有 member
local members = redis.call("HGETALL", memberKey) -- 获取哈希中所有字段和值

-- 3. 把 count 与 members 合并到一个数组里
local result = {}
table.insert(result, count or "0")   -- 如果 count 为 nil，默认当“0”处理
for _, m in ipairs(members) do
    table.insert(result, m)
end

-- 4. 返回合并后的数组：{ count, member1, member2, … }
return result
