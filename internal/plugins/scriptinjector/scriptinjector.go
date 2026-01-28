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
		logger.Error("error on attempting script injection", "error", err, "cookie", c.Cookie, "url", c.Request.RequestURI, "ip", c.IpAddress)
		return false, err
	}

	c.Recorder.Body.Reset()
	c.Recorder.WriteString(result)

	if body != result {
		logger.Debug("altered html", "plugin", "scriptinjector", "old", body, "new", result, "cookie", c.Cookie, "ip", c.IpAddress)
	} else {
		logger.Debug("preprocessor did not alter html", "plugin", "scriptinjector", "cookie", c.Cookie, "ip", c.IpAddress)
	}

	return true, nil
}

// func injectScriptIntoHtml(html, script string) string {
// 	// Modifies a HTML response to include a script.
// 	lower := strings.ToLower(html)
// 	headIdx := strings.Index(lower, "<head")
// 	if headIdx == -1 {
// 		// no head found
// 		logger.Debug("not injecting script into response")
// 		return html
// 	}

// 	// Find the closing tag, and don't do anything if we can't find it
// 	tagEnd := strings.Index(lower[headIdx:], ">")

// 	if tagEnd == -1 {
// 		return html
// 	}

// 	insertPosition := headIdx + tagEnd + 1

// 	new := html[:insertPosition] + "\n" + fmt.Sprintf(`<script src="/gstatic/js/%s.js"></script>`, script) + html[insertPosition:]
// 	return new

// }

func injectScriptIntoHtml(html, script string) (string, error) {
	lower := strings.ToLower(html)

	endHeadIdx := strings.Index(lower, "</head>")
	if endHeadIdx == -1 {
		return html, errors.New("can't find closing head")
	}

	injection := fmt.Sprintf(
		`<script src="/gstatic/js/%s.js"></script>`,
		script,
	)

	return html[:endHeadIdx] + injection + html[endHeadIdx:], nil
}
