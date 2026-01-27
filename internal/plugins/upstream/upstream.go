package upstream

import (
	"gossamer/internal/gossamer"
	"gossamer/internal/logging"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/goccy/go-yaml"
)

var logger *slog.Logger

type UpstreamConfig struct {
	Upstreams  []Upstream `json:"upstreams"`
	ConfigFile string     `env:"CONFIG_FILE_PATH" envDefault:"upstreams.yaml"`
	RepoName   string     `env:"REPO_NAME" envDefault:"gossamer"`
}

type Upstream struct {
	// An upstream is a service protected by Gossamer.

	// The expected Host header
	Hostname string `json:"hostname" validate:"fqdn"`

	// The upstream URL
	Url string `json:"url" validate:"required,url"`

	// IP addresses that are permitted. if nil, everything is ok
	AllowedSources []string `json:"allowed_sources" validate:"dive,cidr"`

	// Paths that should not be accessible.
	BlockedPaths []string `json:"blocked_paths" validate:"dive,dirpath"`
}

func New() (*UpstreamConfig, error) {
	// Parse `upstreams.yaml`
	var cfg UpstreamConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error(
			"failed to get working directory",
			"module", "coraza",
			"error", err,
		)
		return nil, err
	}

	var rootFolder string
	if !strings.HasSuffix(cwd, cfg.RepoName) {
		splitCwd := strings.Split(cwd, "/")
		for index, split := range splitCwd {
			if split == cfg.RepoName {
				splitCwd = splitCwd[:len(splitCwd)-index+1]
				break
			}

		}
		rootFolder = strings.Join(splitCwd, "/")

	}

	configFile := filepath.Join(rootFolder, "conf", cfg.ConfigFile)

	rawData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(rawData, &cfg); err != nil {
		return nil, err
	}

	logger = logging.NewLogger()

	return &cfg, err

}

func (u *UpstreamConfig) findMatchingUpstream(c gossamer.Connection) *Upstream {
	var matchingUpstream *Upstream
	for _, upstream := range u.Upstreams {
		if upstream.Hostname == c.Request.Host {
			matchingUpstream = &upstream
		}
	}

	return matchingUpstream
}

func (u *UpstreamConfig) Validate(c gossamer.Connection) bool {

	// First, find the matching upstream to the request
	matchingUpstream := u.findMatchingUpstream(c)
	if matchingUpstream == nil {
		logger.Info(
			"no matching upstream found",
			"cookie", c.Cookie,
			"ip", c.IpAddress,
			"url", c.Request.URL.String(),
			"plugin", "upstream",
		)
		return false
	}

	// Check if our path is on the blocked paths
	for _, blockedPath := range matchingUpstream.BlockedPaths {
		if strings.HasPrefix(c.Request.URL.Path, blockedPath) {
			logger.Debug(
				"path is on blocklist",
				"cookie", c.Cookie,
				"ip", c.IpAddress,
				"url", c.Request.URL.String(),
				"blocklist_entry", blockedPath,
				"plugin", "upstream",
			)
			return false
		}
	}

	// Check if the client IP matches the allowed sources list
	clientIp, err := netip.ParseAddr(c.IpAddress)
	if err != nil {
		logger.Error(
			"unable to parse client IP address",
			"ip", c.IpAddress,
			"error", err,
			"plugin", "upstream",
		)
		return false
	}

	var sourceAllowed bool
	if len(matchingUpstream.AllowedSources) == 0 {
		sourceAllowed = true
	}

SOURCE_CHECK:
	for _, allowedSource := range matchingUpstream.AllowedSources {
		prefix, err := netip.ParsePrefix(allowedSource)
		if err != nil {
			logger.Error(
				"unable to parse IP network",
				"network", allowedSource,
				"error", err,
				"plugin", "upstream",
			)
			return false
		}

		if prefix.Contains(clientIp) {
			sourceAllowed = true
			break SOURCE_CHECK
		}

	}

	if !sourceAllowed {
		logger.Debug(
			"source not allowed",
			"ip", c.IpAddress,
			"url", matchingUpstream.Hostname,
			"cookie", c.Cookie,
			"plugin", "upstream",
		)
		return false
	}

	return true

}

func (u *UpstreamConfig) Preprocess(c gossamer.Connection) error {
	// Set the request URL to the upstream URL
	upstream := u.findMatchingUpstream(c)

	upstreamUrl, err := url.Parse(upstream.Url + c.Request.URL.Path)
	c.Request.URL = upstreamUrl

	return err
}

func (u *UpstreamConfig) Postprocess(_ gossamer.Connection) error {
	return nil
}

func (u *UpstreamConfig) Verify(_ gossamer.Connection) bool {
	return true
}
