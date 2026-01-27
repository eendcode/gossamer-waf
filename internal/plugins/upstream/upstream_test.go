package upstream_test

import (
	"gossamer/internal/gossamer"
	"gossamer/internal/plugins/upstream"
	"net/http"
	"testing"
)

func TestUpstream(t *testing.T) {
	cfg, err := upstream.New()
	if err != nil {
		t.Errorf("error on parsing upstream config: %v", err)
	}

	r, err := http.NewRequest("GET", "http://code.24.berylia.org/foo", nil)
	if err != nil {
		t.Errorf("error on creating http request: %v", err)
	}

	conn := gossamer.Connection{Request: r, IpAddress: "192.168.1.1"}

	if !cfg.Validate(conn) {
		t.Errorf("incorrectly invalidated request: %v", r)
	}

	if err := cfg.Preprocess(conn); err != nil {
		t.Errorf("error on preprocessing: %v", err)
	}

	if conn.Request.URL.String() != "http://localhost:8000/foo" {
		t.Errorf("expected %s, got %s for url", "http://localhost:8000/foo", conn.Request.URL.String())
	}

	t.Logf("got url %v", conn.Request.URL.String())

}

func TestUnknownUpstream(t *testing.T) {
	cfg, err := upstream.New()
	if err != nil {
		t.Errorf("error on parsing upstream config: %v", err)
	}

	r, err := http.NewRequest("GET", "http://blaat.org", nil)
	if err != nil {
		t.Errorf("error on creating http request: %v", err)
	}

	conn := gossamer.Connection{Request: r, IpAddress: "192.168.1.1"}

	if cfg.Validate(conn) {
		t.Errorf("incorrectly validated request: %v", r)
	}

}

func TestInvalidSourceAddress(t *testing.T) {
	cfg, err := upstream.New()
	if err != nil {
		t.Errorf("error on parsing upstream config: %v", err)
	}

	r, err := http.NewRequest("GET", "http://code.24.berylia.org", nil)
	if err != nil {
		t.Errorf("error on creating http request: %v", err)
	}

	conn := gossamer.Connection{Request: r, IpAddress: "192.2.0.1"}

	if cfg.Validate(conn) {
		t.Errorf("incorrectly validated request: %v", r)
	}

}
