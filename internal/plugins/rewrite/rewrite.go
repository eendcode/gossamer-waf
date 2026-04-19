package rewrite

import (
	"bytes"
	"errors"
	"gossamer/internal/gossamer"
	"gossamer/internal/logging"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v11"
)

var logger *slog.Logger

type Rewriter struct {
	Source string `env:"REWRITE_SOURCE" envDefault:"cdn.example.com"`
	Target string `env:"REWRITE_TARGET" envDefault:"cdn.internal.com"`
}

func New() (*Rewriter, error) {
	var rewriter Rewriter
	if err := env.Parse(&rewriter); err != nil {
		return nil, err
	}

	if rewriter.Source == "" {
		return nil, errors.New("REWRITE_SOURCE must not be empty")
	}
	if rewriter.Target == "" {
		return nil, errors.New("REWRITE_TARGET must not be empty")
	}

	logger = logging.NewLogger()

	return &rewriter, nil
}

func (r *Rewriter) rewriteCdn(body []byte) []byte {
	return bytes.ReplaceAll(body, []byte(r.Source), []byte(r.Target))
}

func (r *Rewriter) Validate(_ gossamer.Connection) bool            { return true }
func (r *Rewriter) Verify(_ gossamer.Connection) bool              { return true }
func (r *Rewriter) Preprocess(_ gossamer.Connection) (bool, error) { return true, nil }
func (r *Rewriter) Postprocess(c gossamer.Connection) (bool, error) {
	logger.Debug("starting postprocessor", "plugin", "rewrite")

	contentType := c.Recorder.Header().Get("content-type")
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		logger.Debug("no html, ignoring")
		return true, nil
	}

	body := c.Recorder.Body.Bytes()
	result := r.rewriteCdn(body)

	c.Recorder.Body.Reset()
	c.Recorder.Write(result)

	return true, nil
}
