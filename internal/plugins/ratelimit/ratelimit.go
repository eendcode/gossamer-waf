package ratelimit

import (
	"context"
	"gossamer/internal/gossamer"
	"gossamer/internal/logging"
	localtbl "gossamer/internal/plugins/ratelimit/local_tbl"
	valkeytbl "gossamer/internal/plugins/ratelimit/valkey_tbl"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v11"
)

var logger *slog.Logger

const (
	VALKEY string = "valkey"
	LOCAL  string = "local"
)

type Settings struct {
	TblLibrary string `env:"TBL_LIBRARY" envDefault:"valkey" validate:"oneof=valkey,local"`
}

type KeyValueStore interface {
	// Check if the client has hit their rate limit
	Allow(context.Context, string) (bool, int64, error)

	// Create a cookie for the client
	CreateCookie(context.Context, string) error

	// Increase the current strike level
	IncreaseStrikes(context.Context, string, int64) error

	// Get the current strike level
	GetStrikes(context.Context, string) (int64, error)
}

type RateLimiter struct {
	KvStore KeyValueStore
}

func New() (*RateLimiter, error) {
	var settings Settings
	if err := env.Parse(&settings); err != nil {
		return nil, err
	}

	logger = logging.NewLogger()

	var kv KeyValueStore
	var err error
	switch v := strings.ToLower(settings.TblLibrary); v {
	case VALKEY:
		kv, err = valkeytbl.New()
	case LOCAL:
		kv, err = localtbl.New()
	}

	return &RateLimiter{KvStore: kv}, err

}

func (r *RateLimiter) Validate(c gossamer.Connection) bool {
	ok, remaining, err := r.KvStore.Allow(context.TODO(), c.Cookie)

	if !ok {
		logger.Debug(
			"rate limit reached",
			"cookie", c.Cookie,
			"ip", c.IpAddress,
		)
		return false
	}

	if err != nil {
		logger.Error("error on key-value operation", "error", err)
		return false
	}

	logger.Debug(
		"validated request",
		"plugin", "ratelimit",
		"cookie", c.Cookie,
		"uri", c.Request.RequestURI,
		"ip", c.IpAddress,
		"remaining", remaining,
	)

	return true
}

func (r *RateLimiter) Verify(c gossamer.Connection) bool       { return true }
func (r *RateLimiter) Preprocess(c gossamer.Connection) error  { return nil }
func (r *RateLimiter) Postprocess(c gossamer.Connection) error { return nil }
