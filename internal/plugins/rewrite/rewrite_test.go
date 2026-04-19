package rewrite

import (
	"gossamer/internal/gossamer"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeBackend() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("<script src='cdn.example.com>"))
	})

	return httptest.NewServer(mux)

}

func TestRewrite(t *testing.T) {

	logger = slog.Default()

	rewriter, err := New()
	if err != nil {
		t.Errorf("error on initializing: %v", err)
	}

	backend := fakeBackend()
	defer backend.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com/", nil)
	r.AddCookie(&http.Cookie{
		Name:  "_gSession",
		Value: "foobar",
	})

	r.Header.Add("user-agent", "foo/bar")
	r.Header.Add("accept", "application/json")

	ok, err := rewriter.Postprocess(gossamer.Connection{
		Request:  r,
		Recorder: w,
	})

	if err != nil {
		t.Errorf("postprocessing failed with err: %v", err)
	}

	if !ok {
		t.Errorf("postprocessing returned false")
	}

}

func TestSimple(t *testing.T) {

	logger = slog.Default()

	rewriter, err := New()
	if err != nil {
		t.Errorf("error on initializing: %v", err)
	}

	newBody := rewriter.rewriteCdn([]byte("<script src='cdn.example.com>"))

	logger.Debug("got new body", "new_body", newBody)

}
