package coraza

import (
	"bytes"
	"context"
	"fmt"
	"gossamer/internal/gossamer"
	"gossamer/internal/helpers"
	"gossamer/internal/logging"

	// localtbl "gossamer/internal/token_bucket/local_tbl"
	// valkeytbl "gossamer/internal/token_bucket/valkey_tbl"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	czv3 "github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/goccy/go-yaml"
)

var logger *slog.Logger
var rootFolder string

const (
	repoName = "gossamer"
	VALKEY   = "valkey"
	LOCAL    = "local"
)

type TokenBucketRules struct {
	// The size of the bucket
	Capacity int `json:"capacity"`

	// Refill rate per second
	RefillRate float64 `json:"refill_rate"`
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

	// Sets if we want to use coraza or not
	DisableCoraza bool `json:"disable_coraza"`

	// Sets the paranoia level - default is 3. Ignored if `DisableCoraza` is `true` .
	ParanoiaLevel int `json:"paranoia_level" validate:"lte=4"`

	// Ignore existing rules for the specific upstream. ID's only
	IngoreRules []int `json:"ignore_rules"`

	// Describe how rate limiting should be implemented
	TokenBucketRules TokenBucketRules `json:"rate_limiting"`

	// Whether to inspect the response or not.
	// WARNING: applying this introduces a huge overhead, so be careful here.
	// It is advised to only use this for API's.
	InspectResponses bool `json:"inspect_responses"`
}

type TokenBucketImplementation interface {
	// Check if the client has hit their rate limit or if the cookie is unknown
	Allow(context.Context, string) (bool, int64, error)

	// Create a cookie for the client
	CreateCookie(context.Context, string) error

	// Increase the current strike level
	IncreaseStrikes(context.Context, string, int64) error

	// Get the current strike level
	GetStrikes(context.Context, string) (int64, error)

	// Check if the cookie exists
	Exists(context.Context, string) (bool, error)
}

type WAF struct {
	// We directly inherit the Coraza WAF
	czv3.WAF

	// Set the default paranoia level. Note that this can be overriden on a per-host basis
	DefaultParanoiaLevel int `env:"DEFAULT_PARANOIA_LEVEL" envDefault:"3" validate:"gte=1,lte=4"`

	// Point to the config file
	ConfigFile string `env:"CONFIG_FILE_PATH" envDefault:"upstreams.yaml"`

	// We have multiple implementations of a token bucket algorithm. We select one here.
	TokenBucketType string `env:"TB_TYPE" envDefault:"valkey" validate:"oneof=valkey,local"`

	// For each host, we create a seperate token bucket implementation
	TokenBucketMap map[string]TokenBucketImplementation

	// All upstreams
	Upstreams []Upstream `json:"upstreams"`

	// Extra Coraza rules
	ExtraRules []string `json:"extra_rules"`

	// Virtual patches
	VirtualPatches []string `json:"virtual_patches"`
}

func New() (*WAF, error) {

	logger = logging.NewLogger()

	var waf WAF
	if err := env.Parse(&waf); err != nil {
		logger.Error(
			"failed to parse environment",
			"module", "coraza",
			"error", err,
		)
		return nil, err
	}

	// ------------------------------------------------------------------------------------------------

	// Read the config file `upstreams.yaml`
	if err := determineRootFolder(); err != nil {
		return nil, err
	}
	configFile := filepath.Join(rootFolder, "conf", waf.ConfigFile)
	rawData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// Parse the YAML file
	if err := yaml.Unmarshal(rawData, &waf); err != nil {
		return nil, err
	}

	// Create the token bucket map -----------------------------------------------------------------------
	tokenBucketMap, err := waf.createTokenBucketMap()
	if err != nil {
		return &waf, err
	}
	waf.TokenBucketMap = tokenBucketMap

	// Create the coraza WAF ----------------------------------------------------------
	cWaf, err := waf.buildCoraza()
	if err != nil {
		return &waf, err
	}
	waf.WAF = cWaf

	return &waf, nil

}

func (w *WAF) createTokenBucketMap() (map[string]TokenBucketImplementation, error) {

	tokenBucketMap := make(map[string]TokenBucketImplementation)

	// We loop through all the upstreams and create a token bucket implementation for each of them.
	for _, upstream := range w.Upstreams {

		tokenBucketImplementation, err := NewLimiter(upstream.TokenBucketRules)
		if err != nil {
			return nil, err
		}

		tokenBucketMap[upstream.Hostname] = tokenBucketImplementation

	}

	return tokenBucketMap, nil
}

func (w *WAF) buildCoraza() (czv3.WAF, error) {
	corazaConf := filepath.Join(rootFolder, "conf", "rules", "coraza.conf")
	crsSetupConf := filepath.Join(rootFolder, "conf", "rules", "coreruleset", "crs-setup.conf.example")
	crsRules := filepath.Join(rootFolder, "conf", "rules", "coreruleset", "rules", "*.conf")
	customConf := filepath.Join(rootFolder, "conf", "rules", "custom.conf")
	paranoiaLevelRule := fmt.Sprintf(
		`SecAction "id:900000,phase:1,pass,t:none,nolog,tag:'OWASP_CRS',ver:'OWASP_CRS/4.22.0-dev',setvar:tx.blocking_paranoia_level=%d"`,
		w.DefaultParanoiaLevel,
	)

	// We make extra coraza rules for specific hosts.
	var extraDirectives []string
	var ruleCounter int = 1100000
	for _, upstream := range w.Upstreams {

		if upstream.DisableCoraza {
			// Disable Coraza if we're instructed to do so
			rule := fmt.Sprintf(`SecRule REQUEST_HEADERS:Host "@eq %s" "id:%d,phase:1,pass,ctl:ruleEngine=Off"`, upstream.Hostname, ruleCounter)
			extraDirectives = append(extraDirectives, rule)

			logger.Debug(
				"adding rule for disabling coraza",
				"upstream", upstream.Hostname,
				"rule", rule,
				"plugin", "coraza",
			)

			ruleCounter++
		} else if upstream.ParanoiaLevel > 0 {
			// Override the default paranoia level if we have one set
			rule := fmt.Sprintf(`SecRule REQUEST_HEADERS:Host "@eq %s" "id:%d,phase:1,pass,nolog,setvar:tx.paranoia_level=%d"`, upstream.Hostname, ruleCounter, upstream.ParanoiaLevel)
			extraDirectives = append(extraDirectives, rule)

			logger.Debug(
				"adding rule for paranoia override",
				"upstream", upstream.Hostname,
				"new_level", upstream.ParanoiaLevel,
				"rule", rule,
				"plugin", "coraza",
			)

			ruleCounter++
		}

		for _, ignoreRule := range upstream.IngoreRules {
			rule := fmt.Sprintf(`SecRule REQUEST_HEADERS:Host "@eq %s" "id:%d,phase:1,pass,ctl:ruleRemoveById=%d"`, upstream.Hostname, ruleCounter, ignoreRule)
			extraDirectives = append(extraDirectives, rule)

			logger.Debug(
				"adding ignore rule",
				"upstream", upstream.Hostname,
				"rule", rule,
				"plugin", "coraza",
			)

			ruleCounter++
		}
	}
	// Add the `extra_rules` part
	extraDirectives = append(extraDirectives, w.ExtraRules...)

	// Make it into one giant string
	extraDirectivesString := strings.Join(extraDirectives, "\n")

	cfg := czv3.NewWAFConfig().
		WithDirectivesFromFile(corazaConf).
		WithDirectives(extraDirectivesString).
		WithDirectivesFromFile(crsSetupConf).
		WithDirectivesFromFile(crsRules).
		WithDirectivesFromFile(customConf).
		WithDirectives(paranoiaLevelRule)

	return czv3.NewWAF(cfg)
}

func (w *WAF) FindMatchingUpstream(c gossamer.Connection) *Upstream {
	var matchingUpstream *Upstream
	for _, upstream := range w.Upstreams {
		if upstream.Hostname == c.Request.Host {
			matchingUpstream = &upstream
		}
	}

	return matchingUpstream
}

func (w *WAF) Validate(c gossamer.Connection) bool {

	// First, find the matching upstream to the request
	matchingUpstream := w.FindMatchingUpstream(c)
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

	// Check if the client is rate limited ----------------------------------------
	tokenBucketLimiter := w.TokenBucketMap[matchingUpstream.Hostname]
	strikes, err := tokenBucketLimiter.GetStrikes(c.Request.Context(), c.Cookie)
	if err != nil {
		logger.Error(
			"unable to get strikes for client",
			"cookie", c.Cookie,
			"error", err,
			"ip_address", c.IpAddress,
			"url", c.Request.RequestURI,
		)
	}

	if strikes >= 20 { // TODO: make the threshold a variable
		logger.Debug(
			"blocking request due to high strike level",
			"cookie", c.Cookie,
			"ip_address", c.IpAddress,
			"strikes", strikes,
			"url", c.Request.RequestURI,
		)
	}

	ok, remaining, err := tokenBucketLimiter.Allow(c.Request.Context(), c.Cookie)

	if err != nil {
		logger.Error(
			"unable to call lua script",
			"error", err,
			"cookie", c.Cookie,
			"ip_address", c.IpAddress,
			"url", c.Request.RequestURI,
		)

		return false
	}

	if !ok {
		logger.Debug(
			"rate limit reached",
			"cookie", c.Cookie,
			"ip_address", c.IpAddress,
			"strikes", strikes,
			"url", c.Request.RequestURI,
		)

		return false
	}

	logger.Debug(
		"rate limit check passed",
		"cookie", c.Cookie,
		"ip_address", c.IpAddress,
		"strikes", strikes,
		"url", c.Request.RequestURI,
		"remaining", remaining,
	)

	// --------------------------------------------------------------------------------

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

func (w *WAF) Verify(r gossamer.Connection) bool {
	return true
}

func (w *WAF) Preprocess(r gossamer.Connection) (bool, error) {
	if err := r.Load(); err != nil {
		logger.Error("error loading connection", "error", "err", "module", "coraza", "cookie", r.Cookie)
		return false, err
	}

	logger.Debug(
		"new transaction",
		"module", "coraza",
		"id", r.Transaction.ID(),
		"ip_address", r.IpAddress,
		"cookie", r.Cookie,
	)

	r.Transaction.ProcessConnection(r.Request.RemoteAddr, 0, r.Request.RequestURI, 0)

	// Process request headers
	for name, values := range r.Request.Header {
		for _, v := range values {
			r.Transaction.AddRequestHeader(name, v)
		}
	}

	logger.Debug(
		"processing request headers",
		"module", "coraza",
		"transaction_id", r.Transaction.ID(),
		"ip", r.IpAddress,
		"cookie", r.Cookie,
	)

	// Coraza phase 1
	if it := r.Transaction.ProcessRequestHeaders(); it != nil {
		logInterruption(it, r)
		return false, nil
	}

	r.Transaction.ProcessURI(r.Request.URL.String(), r.Request.Method, r.Request.Proto)

	logger.Debug(
		"processing request body",
		"module", "coraza",
		"transaction_id", r.Transaction.ID(),
		"cookie", r.Cookie,
		"ip", r.IpAddress,
	)

	// phase 2

	body, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return false, err
	}

	// Extract the response and put it back
	r.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Write body to Coraza transaction
	if _, _, err := r.Transaction.WriteRequestBody(body); err != nil {
		return false, err
	}

	if it, err := r.Transaction.ProcessRequestBody(); it != nil {
		logInterruption(it, r)
		return false, nil
	} else if err != nil {
		logger.Error("error on processing request body", "module", "coraza", "error", err, "cookie", r.Cookie)
		return false, err
	}

	logger.Debug(
		"passed",
		"stage", "validation",
		"module", "coraza",
		"transaction_id", r.Transaction.ID(),
	)

	return true, nil
}

func (w *WAF) Postprocess(r gossamer.Connection) (bool, error) {

	// First, match the hostname
	for _, upstreams := range w.Upstreams {
		if upstreams.Hostname == r.Request.Host {
			if !upstreams.InspectResponses {
				return true, nil
			} else {
				break
			}
		}
	}

	// Coraza phase 3
	for k, values := range r.Recorder.Header() {
		for _, v := range values {
			logger.Debug("adding response header", k, v)
			r.Transaction.AddResponseHeader(k, v)
		}
	}

	if it := r.Transaction.ProcessResponseHeaders(
		r.Recorder.Code,
		"HTTP/1.1",
	); it != nil {
		logInterruption(it, r)
		return false, nil
	}

	// Phase 4
	body := r.Recorder.Body.Bytes()
	if it, _, err := r.Transaction.WriteResponseBody(body); it != nil {
		logInterruption(it, r)
		return false, nil
	} else if err != nil {
		logger.Error("error on writing response body", "error", err, "module", "coraza", "cookie", r.Cookie)
		return false, err
	}

	if it, err := r.Transaction.ProcessResponseBody(); it != nil {
		logInterruption(it, r)
		return false, err
	} else if err != nil {
		logger.Error("error on processing request body", "error", err, "module", "coraza", "cookie", r.Cookie)
		return false, err
	}

	r.Transaction.ProcessLogging()

	return true, nil
}

type AuditLog struct {
	RuleID   int
	Message  string
	Payloads []string
}

func crsRelevantRule(id int) bool {
	return (id >= 910000 && id < 949000) || (id >= 950000 && id < 980000) || id > 1000000
}

func parseRule(matchedDatas []types.MatchData) []string {
	var result []string
	for _, m := range matchedDatas {
		result = append(result, m.Data())
	}
	return result
}

func logInterruption(it *types.Interruption, r gossamer.Connection) {
	var clientIp string
	var auditLogs []AuditLog

	for _, m := range r.Transaction.MatchedRules() {
		ruleId := m.Rule().ID()
		if m.Message() != "" && crsRelevantRule(ruleId) {
			logger.Debug(
				"request blocked by matching rule",
				"transaction_id", r.Transaction.ID(),
				"msg", m.Message(),
				"audit_log", m.AuditLog(),
				"rule_id", ruleId,
				"client_ip", m.ClientIPAddress(),
			)

			clientIp, _ = helpers.ParseRemoteAddr(m.ClientIPAddress())
			auditLogs = append(auditLogs,
				AuditLog{
					RuleID:   ruleId,
					Message:  m.Message(),
					Payloads: parseRule(m.MatchedDatas()),
				})

		}
	}

	logger.Warn(
		"blocked request",
		"cookie", r.Cookie,
		"rule", it.RuleID,
		"transaction_id", r.Transaction.ID(),
		"ip", clientIp,
		"url", r.Url,
		"method", r.Request.Method,
		"audit_logs", auditLogs,
	)

	r.Transaction.ProcessLogging()
}

func determineRootFolder() error {
	// In order to work from the same relative directory each time, we do some unpleasant `os.Getwd()` here.
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error(
			"failed to get working directory",
			"module", "coraza",
			"error", err,
		)
		return err
	}

	if !strings.HasSuffix(cwd, repoName) {
		splitCwd := strings.Split(cwd, "/")
		for index, split := range splitCwd {
			if split == repoName {
				splitCwd = splitCwd[:len(splitCwd)-index+1]
				break
			}

		}
		rootFolder = strings.Join(splitCwd, "/")

	}

	return nil
}
