local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local member = ARGV[3]
local allowed

-- Get Redis server time
local redis_time = redis.call('TIME')
local now = tonumber(redis_time[1]) + tonumber(redis_time[2])/1000000

-- Remove old requests outside the window
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current requests
local count = redis.call('ZCARD', key)

-- Add new request only if under limit
if count < limit then
    redis.call('ZADD', key, now, member)
    allowed = 1
else
    allowed = 0
end

-- Set expiration to 1 hour (3600 seconds) from now
redis.call('EXPIRE', key, 3600)

return allowed
