-- Lua script to add an element with a TTL to the ZSET
local key = KEYS[1]
local element = ARGV[1]
local ttl = tonumber(ARGV[2])
local currentTime = tonumber(redis.call('TIME')[1])
local expireTime = currentTime + ttl

-- Add the element with its expiration time
-- If the element already exists, update its expiration time
redis.call('ZADD', key, expireTime, element)
