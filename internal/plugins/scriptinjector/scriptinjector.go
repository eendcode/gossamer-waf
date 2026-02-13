package scriptinjector

import (
	"errors"
	"fmt"
	"gossamer/internal/gossamer"
	"gossamer/internal/logging"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v11"
)

var logger *slog.Logger

type ScriptInjector struct {
	Script string `env:"HOOK_SCRIPT_NAME" envDefault:"hook"`
}

func New() (*ScriptInjector, error) {
	var settings ScriptInjector
	err := env.Parse(&settings)

	logger = logging.NewLogger()

	return &settings, err
}

// Default stuff
func (s *ScriptInjector) Validate(_ gossamer.Connection) bool            { return true }
func (s *ScriptInjector) Preprocess(_ gossamer.Connection) (bool, error) { return true, nil }
func (s *ScriptInjector) Verify(_ gossamer.Connection) bool              { return true }

func (s *ScriptInjector) Postprocess(c gossamer.Connection) (bool, error) {

	logger.Debug("starting preprocessor", "plugin", "scriptinjector", "cookie", c.Cookie, "ip", c.IpAddress)

	contentType := c.Recorder.Header().Get("content-type")
	if !strings.Contains(contentType, "text/html") {
		logger.Debug("no html, ignoring")
		return true, nil
	}

	body := c.Recorder.Body.String()
	result, err := injectScriptIntoHtml(body, s.Script)

	if err != nil {
		logger.Error("error on attempting script injection", "body", body, "error", err, "cookie", c.Cookie, "url", c.Request.RequestURI, "ip", c.IpAddress)
		return false, err
	}

	c.Recorder.Body.Reset()
	c.Recorder.WriteString(result)

	if body != result {
		logger.Debug("altered html", "plugin", "scriptinjector", "old", len(body), "new", len(result), "cookie", c.Cookie, "ip", c.IpAddress)
	} else {
		logger.Debug("preprocessor did not alter html", "plugin", "scriptinjector", "cookie", c.Cookie, "ip", c.IpAddress)
	}

	return true, nil
}

func injectScriptIntoHtml(html, script string) (string, error) {
	lower := strings.ToLower(html)

	endHeadIdx := strings.Index(lower, "</head>")
	if endHeadIdx == -1 {
		if strings.Contains(lower, "<head>") {
			// There's a <head>, but not a </head> -> error
			return html, errors.New("can't find closing head")
		}
		// There wasn't a <head> to begin with - not unusual for stuff like 302's
		return html, nil
	}

	injection := fmt.Sprintf(
		`<script src="/gstatic/js/%s.js"></script>`,
		script,
	)

	return html[:endHeadIdx] + injection + html[endHeadIdx:], nil
}
