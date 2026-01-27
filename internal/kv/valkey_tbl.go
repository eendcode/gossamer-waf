package kv

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/valkey-io/valkey-go"
)

var valkeyClient valkey.Client

func newValkey() error {
	var err error

	if valkeyClient == nil {
		var settings Settings
		if err := env.Parse(&settings); err != nil {
			return err
		}

		valkeyClient, err = valkey.NewClient(
			valkey.ClientOption{
				InitAddress: []string{fmt.Sprintf("%s:%d", settings.Host, settings.Port)},
			},
		)

	}
	return err

}

func NewTbl(capacity int, refillRate float64) (*TokenBucketLimiter, error) {
	err := newValkey()
	script := valkey.NewLuaScript(TokenBucketScript)

	return &TokenBucketLimiter{
		Client:     valkeyClient,
		Script:     script,
		Capacity:   capacity,
		RefillRate: refillRate,
	}, err

}

type TokenBucketLimiter struct {
	Client     valkey.Client
	Script     *valkey.Lua
	Capacity   int     `env:"TBL_CAPACITY" envDefault:"60"`
	RefillRate float64 `env:"TBL_REFILL_RATE" envDefault:"0.01"`
}

func (tbl *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, int64, int64, error) {

	now := time.Now().UnixMilli()

	result, err := tbl.Script.Exec(
		ctx,
		tbl.Client,
		[]string{key},
		[]string{
			strconv.Itoa(int(now)),
			strconv.Itoa(int(tbl.RefillRate)),
			strconv.Itoa(tbl.Capacity),
		},
	).ToArray()

	if len(result) != 3 || err != nil {
		return false, 0, 0, fmt.Errorf("unexpected return type %v from script", result)
	}

	allowedInt, err := result[0].AsInt64()
	if err != nil {
		return false, 0, 0, fmt.Errorf("expected int64 for allowed, got %v", result[0])
	}

	allowed := allowedInt == 1

	remaining, err := result[1].AsInt64()
	if err != nil {
		return false, 0, 0, fmt.Errorf("expected int64 for remaining, got %v", result[1])
	}

	strikes, err := result[2].AsInt64()
	if err != nil {
		return false, 0, 0, fmt.Errorf("expected int64 for strikes, got %v", result[2])
	}

	return allowed, remaining, strikes, nil

}

func (tbl *TokenBucketLimiter) Increase(ctx context.Context, cookie string, number int64) error {
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Hincrby().Key(cookie).Field(strikesField).Increment(number).Build(),
	).Error()
}

func (tbl *TokenBucketLimiter) ResetStrikes(ctx context.Context, cookie string) error {
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Hmset().Key(cookie).FieldValue().FieldValue(strikesField, "0").Build(),
	).Error()
}

func (tbl *TokenBucketLimiter) CreateCookie(ctx context.Context, cookie string) error {
	// Create a cookie in valkey
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Hmset().Key(cookie).FieldValue().FieldValue(strikesField, "-1").Build(),
	).Error()

}

func (tbl *TokenBucketLimiter) GetStrikes(ctx context.Context, cookie string) (int64, error) {
	return tbl.Client.Do(
		ctx,
		tbl.Client.B().Hmget().Key(cookie).Field(strikesField).Build(),
	).AsInt64()
}
