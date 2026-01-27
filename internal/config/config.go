package config

import (
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/goccy/go-yaml"
)

// var cfg *Config

type Settings struct {
	ConfigFile string `env:"CONFIG_FILE_PATH" envDefault:"conf/upstreams.yaml"`
}

type Upstream struct {
	// An upstream is a service protected by Gossamer.
	// Upstreams are stored in valkey, but are loaded on each Gossamer node.
	// In case of configuration changes, a Pub/Sub message is sent to all Gossamer nodes, prompting them to change their configuration.

	// The expected Host header
	Hostname string `json:"hostname" validate:"fqdn"`

	// The upstream URL
	Url string `json:"url" validate:"required,url"`

	// IP addresses that are permitted. if nil, everything is ok
	AllowedSources []string `json:"allowed_sources" validate:"dive,cidr"`

	// Paths that should be exposed via the WAF
	Paths []string `json:"paths" validate:"dive,dirpath"`

	// Paths that are exposed, but aren't checked by gossamer. Use sparingly
	ExceptionPaths []string `json:"exception_paths" validate:"dive,dirpath"`

	// Paths that should not be accessible. Note that you only have to worry about paths that are exposed via the `Paths` field
	BlockedPaths []string `json:"blocked_paths" validate:"dive,dirpath"`

	// Whether to require cookies. Only set this to false if you know what you are doing.
	RequireCookie bool `json:"require_cookie"`
}

// type RateLimit struct {
// 	Capacity    int              `json:"capacity"`
// 	RefillRate  float64          `json:"refill_rate"`
// 	RateLimiter kv.KeyValueStore `json:"-"`
// }

type Config struct {
	Upstreams []Upstream `json:"upstreams"`
}

func ParseConfig() (*Config, error) {
	// Parse `upstreams.yaml`
	var settings Settings
	if err := env.Parse(&settings); err != nil {
		return nil, err
	}

	rawData, err := os.ReadFile(settings.ConfigFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(rawData, &cfg); err != nil {
		return nil, err
	}

	return &cfg, err

}

// func init() {
// 	var err error
// 	cfg, err = ParseConfig()
// 	if err != nil {
// 		panic(err)
// 	}
// }

// func DetermineUpstream(r *http.Request) *Upstream {

// }
