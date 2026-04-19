package comments

import (
	"gossamer/internal/gossamer"
	"log/slog"
	"regexp"
	"strings"
)

var logger *slog.Logger

type CommentRemover struct{}

func New() (*CommentRemover, error) {
	return &CommentRemover{}, nil
}

func (c *CommentRemover) Validate(_ gossamer.Connection) bool            { return true }
func (c *CommentRemover) Preprocess(_ gossamer.Connection) (bool, error) { return true, nil }
func (c *CommentRemover) Verify(_ gossamer.Connection) bool              { return true }

func (c *CommentRemover) Postprocess(r gossamer.Connection) (bool, error) {

	logger.Debug("starting preprocessor", "plugin", "comments")

	contentType := r.Recorder.Header().Get("content-type")
	if !strings.Contains(contentType, "text/html") {
		logger.Debug("no html, ignoring")
		return true, nil
	}

	body := r.Recorder.Body.String()

	result := removeComments(body)

	r.Recorder.Body.Reset()
	r.Recorder.WriteString(result)

	return true, nil
}

func removeComments(html string) string {
	re := regexp.MustCompile(`(?s)<!--.*?-->`)
	return re.ReplaceAllString(html, "")
}
