package limiter

// Redis Lua script constants for atomic distributed execution across multi-node clusters.

const (
	// RedisTokenBucketScript atomically refills tokens and decrements if available.
	// KEYS[1]: bucket key (e.g., "ratelimit:tb:{client_id}")
	// ARGV[1]: capacity (number of tokens)
	// ARGV[2]: refill_rate_per_sec (float)
	// ARGV[3]: requested_tokens (int)
	// ARGV[4]: current_timestamp_sec (float)
	RedisTokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local requested = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

local data = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

if not tokens then
    tokens = capacity
    last_refill = now
else
    local elapsed = math.max(0, now - last_refill)
    tokens = math.min(capacity, tokens + (elapsed * refill_rate))
    last_refill = now
end

if tokens >= requested then
    tokens = tokens - requested
    redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
    redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) * 2)
    return {1, math.floor(tokens)}
else
    local deficit = requested - tokens
    local retry_after = math.ceil(deficit / refill_rate)
    return {0, math.floor(tokens), retry_after}
end
`

	// RedisSlidingWindowScript tracks requests using a Redis Sorted Set (ZSET).
	// KEYS[1]: rate limit key
	// ARGV[1]: window_size_ms
	// ARGV[2]: max_requests
	// ARGV[3]: current_timestamp_ms
	// ARGV[4]: unique_request_id
	RedisSlidingWindowScript = `
local key = KEYS[1]
local window_size = tonumber(ARGV[1])
local max_limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local req_id = ARGV[4]

local clear_before = now - window_size
redis.call("ZREMRANGEBYSCORE", key, "-inf", clear_before)

local current_count = redis.call("ZCARD", key)

if current_count < max_limit then
    redis.call("ZADD", key, now, req_id)
    redis.call("PEXPIRE", key, window_size)
    local remaining = max_limit - (current_count + 1)
    return {1, remaining}
else
    return {0, 0}
end
`

	// RedisFixedWindowScript increments and sets TTL on first hit.
	// KEYS[1]: rate limit key
	// ARGV[1]: limit
	// ARGV[2]: window_sec
	RedisFixedWindowScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local count = redis.call("INCR", key)
if count == 1 then
    redis.call("EXPIRE", key, window)
end

if count <= limit then
    return {1, limit - count}
else
    local ttl = redis.call("TTL", key)
    return {0, 0, ttl}
end
`
)
