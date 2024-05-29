-- Get all keys and remove all expired items from them
local keys = redis.call('KEYS', "*")
local currentTime = tonumber(redis.call('TIME')[1])

for i, key in ipairs(keys) do
    redis.call('ZREMRANGEBYSCORE', key, '-inf', currentTime)
end
