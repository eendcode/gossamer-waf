package coraza

import (
	"fmt"
	"gossamer/internal/gossamer"
	"gossamer/internal/helpers"
	"gossamer/internal/logging"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	czv3 "github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
)

var logger *slog.Logger

type WAF struct {
	czv3.WAF
	ParanoiaLevel   int    `env:"CRS_PARANOIA_LEVEL" envDefault:"3" validate:"gte=1,lte=4"`
	RepoName        string `env:"REPO_NAME" envDefault:"gossamer"`
	InspectResponse bool   `env:"CORAZA_INSPECT_RESPONSE" envDefault:"false"`
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
	if !strings.HasSuffix(cwd, waf.RepoName) {
		splitCwd := strings.Split(cwd, "/")
		for index, split := range splitCwd {
			if split == waf.RepoName {
				splitCwd = splitCwd[:len(splitCwd)-index+1]
				break
			}

		}
		rootFolder = strings.Join(splitCwd, "/")

	}

	corazaConf := filepath.Join(rootFolder, "conf", "rules", "coraza.conf")
	crsSetupConf := filepath.Join(rootFolder, "conf", "rules", "coreruleset", "crs-setup.conf.example")
	crsRules := filepath.Join(rootFolder, "conf", "rules", "coreruleset", "rules", "*.conf")
	customConf := filepath.Join(rootFolder, "conf", "rules", "custom.conf")
	paranoiaLevelRule := fmt.Sprintf(
		`SecAction "id:900000,phase:1,pass,t:none,nolog,tag:'OWASP_CRS',ver:'OWASP_CRS/4.22.0-dev',setvar:tx.blocking_paranoia_level=%d"`,
		waf.ParanoiaLevel,
	)

	cfg := czv3.NewWAFConfig().
		WithDirectivesFromFile(corazaConf).
		WithDirectives(paranoiaLevelRule).
		WithDirectivesFromFile(crsSetupConf).
		WithDirectivesFromFile(crsRules).
		WithDirectivesFromFile(customConf)

	waf.WAF, err = czv3.NewWAF(cfg)

	return &waf, err

}

func (w *WAF) Validate(r gossamer.Connection) bool {

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
	if !w.InspectResponse {
		return true, nil
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
		"client_ip", clientIp,
		"audit_logs", auditLogs,
	)

	r.Transaction.ProcessLogging()
}
