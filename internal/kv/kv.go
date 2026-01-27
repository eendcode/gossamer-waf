package kv

import (
	"context"
)

type Settings struct {
	Host string `env:"KV_HOST" envDefault:"localhost"`
	Port int    `env:"KV_PORT" envDefault:"6379"`
	Type string `env:"KV_TYPE" envDefault:"valkey"`
}

const (
	strikesField string = "strikes"
)

type KeyValueStore interface {
	// An interface for interacting with a key-value store

	// Increase the `strikes` counter corresponding to a cookie
	Increase(context.Context, string, int64) error

	// Create a cookie in the KV-store
	CreateCookie(context.Context, string) error

	ResetStrikes(context.Context, string) error

	// Allow calls our custom lua script to check if the rate limiting
	// is activated and returns the number of strikes.
	Allow(context.Context, string) (bool, int64, int64, error)

	GetStrikes(context.Context, string) (int64, error)
}
