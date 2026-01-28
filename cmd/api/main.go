package main

import (
	"fmt"
	"gossamer/internal/logging"
	"gossamer/internal/plugins"
	"log/slog"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

var settings struct {
	ListenAddress      string `env:"LISTEN_ADDR" envDefault:"0.0.0.0"`
	ListenPort         int16  `env:"LISTEN_PORT" envDefault:"8080"`
	TlsCertificatePath string `env:"TLS_CERTIFICATE_PATH" envDefault:"tls.crt"`
	TlsKeyPath         string `env:"TLS_KEY_PATH" envDefault:"tls.key"`
	// DefaultUpstreamUrl        string        `env:"DEFAULT_UPSTREAM_URL" envDefault:"http://127.0.0.1:8000"`
	// Version                   string        `env:"VERSION" envDefault:"0.1.0"`
	// MaxStrikes                int64         `env:"MAX_STRIKES" envDefault:"100"`
	// StrikesOnOffence          int64         `env:"STRIKES_ON_OFFENCE" envDefault:"10"`
	// CookieName                string        `env:"COOKIE_NAME" envDefault:"phpSessionId"`
	// CookieValidTime           time.Duration `env:"COOKIE_VALID_TIME" envDefault:"9h"`
	// BlockStatusCode           int           `env:"BLOCK_STATUS_CODE" envDefault:"403"`
	// LogLevel                  string        `env:"LOG_LEVEL" envDefault:"info"`
	// Domain                    string        `env:"DOMAIN"`
	IgnorePlugins []string `env:"IGNORE_PLUGINS"`
	HookDebugMode bool     `env:"HOOK_DEBUG_MODE" envDefault:"false"`
	// LoginRateLimitCapacity    int           `env:"LOGIN_RATE_LIMIT_CAPACITY" envDefault:"5"`
	// LoginRateLimitRefillRate  float64       `env:"LOGIN_RATE_LIMIT_REFILL_RATE" envDefault:"0.1"` // 0.1 per second
	// CookieRateLimitCapacity   int           `env:"COOKIE_RATE_LIMIT_CAPACITY" envDefault:"40"`
	// CookieRateLimitRefillRate float64       `env:"COOKIE_RATE_LIMIT_REFILL_RATE" envDefault:"2.0"` // 2 per second
	// CrsParanoiaLevel          int           `env:"CRS_PARANOIA_LEVEL" envDefault:"3" validate:"gte=1,lte=4"`
	// ClientHints               []string      `env:"CLIENT_HINTS" envDefault:"Sec-CH-UA,Sec-CH-UA-Bitness,Sec-CH-UA-Full-Version,Sec-CH-UA-Mobile,Sec-CH-UA-Model,Sec-CH-UA-Platform,Sec-CH-DPR"`
	// InjectionScript           string        `env:"INJECTION_SCRIPT" envDefault:"troll.min.js"`
}

var (
	logger     *slog.Logger
	validate   *validator.Validate = validator.New(validator.WithRequiredStructEnabled())
	pluginMap  map[string]plugins.Plugin
	modifyFunc func(*http.Response) error
	// ErrNoCookieFound      = errors.New("no cookie found")
	// ErrInvalidCookie      = errors.New("invalid cookie")
	// ErrUnknownCookie      = errors.New("no such cookie found in database")
	// ErrMarshalCookie      = errors.New("marshalling cookie failed")
	// ErrRateLimiting       = errors.New("rate limit reached")
	// ErrWafBlocked         = errors.New("blocked by waf")
	// ErrTooManyStrikes     = errors.New("too many strikes")
	// ErrNotActive          = errors.New("cookie not yet active")
	// ErrNoMatchingUpstream = errors.New("no matching upstream")
	// ErrForbidden          = errors.New("forbidden")
)

const (
	repoName            string = "gossamer"
	NoUpstreamForClient int    = 702
)

func healthz(w http.ResponseWriter, _ *http.Request) {
	// Health endpoint
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {

	// parse environment variables
	if err := env.Parse(&settings); err != nil {
		panic(err)
	}

	logger = logging.NewLogger()

	// Set up our plugins
	var err error
	pluginMap, err = plugins.InitializePlugins()
	if err != nil {
		logger.Error("error initializing plugins", "error", err)
		return
	}

	// Set up modifiers
	modifyFunc, err = plugins.ModifyFunc()
	if err != nil {
		logger.Error("error initializing modifiers", "error", err)
		return
	}

	// Set up the valkey connection

	mux := http.NewServeMux()
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/gstatic/", http.StripPrefix("/gstatic/", http.FileServer(http.Dir("static/"))))
	mux.HandleFunc("/gshealth", healthz)
	mux.HandleFunc("/_gsHook", checkIn)
	mux.HandleFunc("/", handleProxy)

	logger.Info(
		"running gossamer",
		"listen_port", settings.ListenPort,
	)

	listenString := fmt.Sprintf("%s:%d", settings.ListenAddress, settings.ListenPort)

	if settings.ListenPort == 443 || settings.ListenPort == 8443 {
		if err := http.ListenAndServeTLS(listenString, settings.TlsCertificatePath, settings.TlsKeyPath, mux); err != nil {
			logger.Error("error occured on setting up listener", "error", err)
		}
	} else {
		if err := http.ListenAndServe(listenString, mux); err != nil {
			logger.Error("error occured on setting up listener", "error", err)
		}
	}
}
