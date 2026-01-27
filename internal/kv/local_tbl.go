package kv

import (
	"context"
	"errors"
	"math"
	"time"
)

var (
	ErrNoCookieFound = errors.New("no cookie found")
)

type LocalRecord struct {
	Strikes   int64 `validate:"gte=-1"`
	Tokens    int64 `validate:"gte=0"`
	Timestamp int64 `validate:"gte=0"`
}

type LocalTbl struct {
	Capacity   int
	RefillRate float64
	Records    map[string]*LocalRecord
}

func (lt *LocalTbl) Increase(_ context.Context, cookie string, number int64) error {

	record := lt.Records[cookie]
	if record == nil {
		return ErrNoCookieFound
	}

	record.Strikes = record.Strikes + number

	lt.Records[cookie] = record

	return nil
}

func (lt *LocalTbl) CreateCookie(_ context.Context, cookie string) error {

	if lt.Records == nil {
		lt.Records = make(map[string]*LocalRecord)
	}

	record := &LocalRecord{
		Strikes:   -1,
		Tokens:    int64(lt.Capacity),
		Timestamp: time.Now().UnixMilli(),
	}

	lt.Records[cookie] = record

	return nil
}

func (lt *LocalTbl) GetStrikes(_ context.Context, cookie string) (int64, error) {
	return lt.Records[cookie].Strikes, nil
}

func (lt *LocalTbl) ResetStrikes(_ context.Context, cookie string) error {
	record := lt.Records[cookie]
	record.Strikes = 0

	lt.Records[cookie] = record

	return nil
}

func (lt *LocalTbl) Allow(_ context.Context, key string) (bool, int64, int64, error) {

	if lt.Records == nil {
		lt.Records = make(map[string]*LocalRecord)
	}

	var record *LocalRecord
	var allowed bool

	record = lt.Records[key]

	now := time.Now().UnixMilli()

	if record == nil {
		record = &LocalRecord{
			Strikes:   0,
			Tokens:    int64(lt.Capacity),
			Timestamp: now,
		}
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
		if record.Strikes > 0 {
			record.Strikes = record.Strikes - 1
		}
	}

	lt.Records[key] = record

	return allowed, record.Tokens, record.Strikes, nil
}
