package coraza

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/valkey-io/valkey-go"
)

type ValkeyTokenBucketLimiter struct {
	Client     valkey.Client
	Script     *valkey.Lua
	Capacity   int
	RefillRate float64
	Host       string `env:"VALKEY_HOST" envDefault:"localhost"`
	Port       int    `env:"VALKEY_PORT" envDefault:"6379"`
}

func NewLimiter(rules TokenBucketRules) (*ValkeyTokenBucketLimiter, error) {
	var tbl ValkeyTokenBucketLimiter
	if err := env.Parse(&tbl); err != nil {
		return nil, err
	}

	tbl.Capacity = rules.Capacity
	tbl.RefillRate = rules.RefillRate

	var err error
	tbl.Client, err = valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", tbl.Host, tbl.Port)},
		},
	)

	tbl.Script = valkey.NewLuaScript(TokenBucketScript)

	return &tbl, err
}

func (tbl *ValkeyTokenBucketLimiter) CreateCookie(ctx context.Context, cookie string) error {
	// Create a cookie in valkey
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Hmset().Key(cookie).FieldValue().FieldValue("cookie", cookie).Build(),
	).Error()

}

func (tbl *ValkeyTokenBucketLimiter) IncreaseStrikes(ctx context.Context, cookie string, incr int64) error {
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Hincrby().Key(cookie).Field("strikes").Increment(incr).Build(),
	).Error()
}

func (tbl *ValkeyTokenBucketLimiter) GetStrikes(ctx context.Context, cookie string) (int64, error) {
	val, err := tbl.Client.Do(
		ctx,
		tbl.Client.B().Hget().Key(cookie).Field("strikes").Build(),
	).AsInt64()

	if err == valkey.Nil {
		return 0, nil
	} else {
		return val, err
	}
}

func (tbl *ValkeyTokenBucketLimiter) Exists(ctx context.Context, key string) (bool, error) {

	// Check if the cookie exists at all
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Exists().Key(key).Build(),
	).AsBool()

}

func (tbl *ValkeyTokenBucketLimiter) Allow(ctx context.Context, key string) (bool, int64, error) {

	now := time.Now().UnixMilli()

	result, err := tbl.Script.Exec(
		ctx,
		tbl.Client,
		[]string{key},
		[]string{
			strconv.Itoa(int(now)),
			strconv.FormatFloat(tbl.RefillRate, 'f', -1, 64),
			strconv.Itoa(tbl.Capacity),
		},
	).ToArray()

	if len(result) != 2 || err != nil {
		return false, 0, fmt.Errorf("unexpected return type %v from script", result)
	}

	allowedInt, err := result[0].AsInt64()
	if err != nil {
		return false, 0, fmt.Errorf("expected int64 for allowed, got %v", result[0])
	}

	allowed := allowedInt == 1

	remaining, err := result[1].AsInt64()
	if err != nil {
		return false, 0, fmt.Errorf("expected int64 for remaining, got %v", result[1])
	}

	return allowed, remaining, nil

}

var TokenBucketScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])

local data = redis.call('HMGET', key, 'tokens', 'timestamp')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])


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
end

redis.call('HMSET', key, 'tokens', tokens, 'timestamp', now)
redis.call('PEXPIRE', key, 86400000)

return {allowed, tokens}
`
