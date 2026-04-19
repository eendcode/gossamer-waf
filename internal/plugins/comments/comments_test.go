package comments

import (
	"log/slog"
	"testing"
)

func TestSimple(t *testing.T) {

	logger = slog.Default()

	newBody := removeComments("hoi hoi <!-- Login Modal --> hoi ")

	logger.Debug("got new body", "new_body", newBody)

}
