package kv

var TokenBucketScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])

local data = redis.call('HMGET', key, 'tokens', 'timestamp', 'strikes')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])
local strikes = tonumber(data[3])

if strikes == nil then
    strikes = 0
end

if tokens == nil then
    tokens = capacity
    last_refill = now
end

local delta = now - last_refill
if delta > 0 then
    local refill = math.floor(delta * refill_rate / 1000)
    tokens = math.min(capacity, tokens + refill)
end

local allowed = 0
if tokens >= 1 then
    allowed = 1
    tokens = tokens - 1
    if strikes > 0 then 
        strikes = strikes - 1
    end
end

redis.call('HMSET', key, 'tokens', tokens, 'timestamp', now, 'strikes', strikes)
redis.call('PEXPIRE', key, 86400000)

return {allowed, tokens, strikes}
`
