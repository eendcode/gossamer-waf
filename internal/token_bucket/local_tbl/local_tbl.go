package localtbl

import (
	"context"
	"errors"
	"gossamer/internal/plugins/coraza"
	"math"
	"time"

	"github.com/caarlos0/env/v11"
)

type TokenBucketLimiter struct {
	Capacity   int     `env:"RATE_LIMIT_CAPACITY" envDefault:"60"`
	RefillRate float64 `env:"RATE_LIMIT_REFILL_RATE" envDefault:"2.0"`
	Records    map[string]*Record
}

type Record struct {
	Tokens    int64
	Timestamp int64
	Strikes   int64
}

func New(rules coraza.TokenBucketRules) (*TokenBucketLimiter, error) {
	var tbl TokenBucketLimiter
	err := env.Parse(&tbl)

	tbl.Capacity = rules.Capacity
	tbl.RefillRate = rules.RefillRate

	return &tbl, err
}

func (tbl *TokenBucketLimiter) CreateCookie(_ context.Context, cookie string) error {
	if tbl.Records == nil {
		tbl.Records = make(map[string]*Record)
	}

	record := &Record{
		Tokens:    int64(tbl.Capacity),
		Timestamp: time.Now().UnixMilli(),
	}

	tbl.Records[cookie] = record

	return nil
}

func (lt *TokenBucketLimiter) IncreaseStrikes(_ context.Context, cookie string, increment int64) error {
	if lt.Records == nil {
		lt.Records = make(map[string]*Record)
	}

	var record *Record

	record = lt.Records[cookie]
	if record == nil {
		return errors.New("no such cookie")
	}

	record.Strikes += int64(increment)

	lt.Records[cookie] = record

	return nil
}

func (lt *TokenBucketLimiter) GetStrikes(_ context.Context, cookie string) (int64, error) {
	if lt.Records == nil {
		lt.Records = make(map[string]*Record)
	}

	record := lt.Records[cookie]
	if record == nil {
		return -1, errors.New("no such cookie")
	}

	return record.Strikes, nil
}

func (lt *TokenBucketLimiter) Exists(_ context.Context, key string) (bool, error) {

	return lt.Records[key] != nil, nil
}

func (lt *TokenBucketLimiter) Allow(_ context.Context, key string) (bool, int64, error) {

	if lt.Records == nil {
		lt.Records = make(map[string]*Record)
	}

	var record *Record
	var allowed bool

	record = lt.Records[key]

	now := time.Now().UnixMilli()

	// if record == nil {
	// 	record = &Record{
	// 		Tokens:    int64(lt.Capacity),
	// 		Timestamp: now,
	// 	}
	// }

	if record == nil {
		return false, -1, nil
	}

	delta := now - record.Timestamp
	if delta > 0 {
		refill := int64(
			math.Floor(
				float64(delta) * lt.RefillRate / 1000,
			),
		)

		record.Tokens = int64(
			math.Min(
				float64(lt.Capacity),
				float64(record.Tokens+refill),
			),
		)
	}

	if record.Tokens >= 1 {
		allowed = true
		record.Tokens = record.Tokens - 1
	}

	lt.Records[key] = record

	return allowed, record.Tokens, nil
}
