package main

import (
	"gossamer/internal/plugins"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/caarlos0/env/v11"
)

func createMockUpstream() (*httptest.Server, error) {
	if err := env.Parse(&settings); err != nil {
		panic(err)
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	backend := httptest.NewServer(mux)

	var err error
	pluginMap, err = plugins.InitializePlugins()
	if err != nil {
		logger.Error("error initializing plugins", "error", err)
		return nil, err
	}

	// Set up modifiers
	modifyFunc, err = plugins.ModifyFunc()
	if err != nil {
		logger.Error("error initializing modifiers", "error", err)
		return nil, err
	}
	return backend, nil
}

func TestDummy(t *testing.T) {
	backend, err := createMockUpstream()
	if err != nil {
		t.Errorf("error occured on init: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", backend.URL, nil)

	handleProxy(w, r)

	if w.Code != 200 {
		t.Error("not 200")
	}
}

// import (
// 	"gossamer/internal/kv"
// 	"gossamer/pkg/rendering"
// 	"log/slog"
// 	"net/http"
// 	"net/http/httptest"
// 	"os"
// 	"testing"

// 	"github.com/caarlos0/env/v11"
// )

// func createMockUpstream() *httptest.Server {
// 	// Create a mock upstream configuration for testing

// 	if err := env.Parse(&settings); err != nil {
// 		panic(err)
// 	}

// 	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

// 	var err error
// 	waf, err = newWaf(4)
// 	if err != nil {
// 		panic(err)
// 	}

// 	mux := http.NewServeMux()

// 	// Give a 500, that should break in phase 3
// 	mux.HandleFunc("GET /break3", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(500)
// 		w.Write([]byte("SQL error"))
// 	})

// 	// Give a 200, but a PHP error that should break in phase 4
// 	mux.HandleFunc("GET /break4", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(200)
// 		w.Write([]byte("Cannot use object as array"))
// 	})
// 	// We do a 204 (no content) for other requests, to distinguish easily between the 200's given by redirects
// 	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(204)
// 	})

// 	backend := httptest.NewServer(mux)

// 	upstreamCfg = &UpstreamsConfig{
// 		Upstreams: []Upstream{
// 			{
// 				Hostname: "example.com",
// 				Url:      backend.URL,
// 				AllowedSources: []string{
// 					"192.0.2.0/24",
// 				},
// 				RateLimit: RateLimit{
// 					Capacity:   10,
// 					RefillRate: 1,
// 					RateLimiter: &kv.LocalTbl{
// 						Capacity:   10,
// 						RefillRate: 1.0,
// 					},
// 				},
// 			},
// 		},
// 		LoginRateLimiter: &kv.LocalTbl{
// 			Capacity:   10,
// 			RefillRate: 1.0,
// 		},
// 	}

// 	return backend
// }

// func TestOk(t *testing.T) {
// 	// A test that should be an OK request, forwarded to a proper upstream.

// 	backend := createMockUpstream()
// 	defer backend.Close()

// 	w := httptest.NewRecorder()
// 	r := httptest.NewRequest("GET", "http://example.com/", nil)
// 	handleProxy(w, r)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
// 	}

// }

// func TestNoUpstream(t *testing.T) {
// 	// A test that should have no viable upstream
// 	backend := createMockUpstream()
// 	defer backend.Close()

// 	w := httptest.NewRecorder()

// 	// note that `example.com` differs from the valid `example.test`
// 	r := httptest.NewRequest("GET", "http://example.test/", nil)
// 	handleProxy(w, r)

// 	if w.Code != http.StatusForbidden {
// 		t.Errorf("expected %d, got %d", http.StatusForbidden, w.Code)
// 	}

// }

// func TestWithCookie(t *testing.T) {
// 	backend := createMockUpstream()
// 	defer backend.Close()

// 	cookie, err := generateCookie(upstreamCfg.LoginRateLimiter)
// 	if err != nil {
// 		t.Errorf("encountered error on generating cookie: %v", err)
// 	}

// 	t.Logf("generated cookie: %s", cookie)

// 	w := httptest.NewRecorder()
// 	r := httptest.NewRequest("GET", "http://example.com", nil)
// 	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
// 	r.Header.Set("Cookie", cookie)
// 	r.Header.Set("User-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:146.0) Gecko/20100101 Firefox/146.0")
// 	handleProxy(w, r)

// 	if w.Code != http.StatusNoContent {
// 		t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
// 	}

// }

// func TestWafBlockPhase1(t *testing.T) {
// 	// A test that should be stopped by the WAF
// 	backend := createMockUpstream()
// 	defer backend.Close()

// 	cookie, err := generateCookie(upstreamCfg.LoginRateLimiter)
// 	if err != nil {
// 		t.Errorf("encountered error on generating cookie: %v", err)
// 	}

// 	t.Logf("generated cookie: %s", cookie)

// 	w := httptest.NewRecorder()
// 	r := httptest.NewRequest("GET", "http://example.com", nil)
// 	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
// 	r.Header.Set("Cookie", cookie)
// 	r.Header.Set("user-agent", "nikto")
// 	handleProxy(w, r)

// 	if w.Code != rendering.UnworthyReturnCode && w.Code != http.StatusPaymentRequired {
// 		t.Errorf("expected %d, got %d", rendering.UnworthyReturnCode, w.Code)
// 	}

// }

// func TestWafBlockPhase2(t *testing.T) {
// 	// A test that should be stopped by the WAF
// 	backend := createMockUpstream()
// 	defer backend.Close()

// 	cookie, err := generateCookie(upstreamCfg.LoginRateLimiter)
// 	if err != nil {
// 		t.Errorf("encountered error on generating cookie: %v", err)
// 	}

// 	t.Logf("generated cookie: %s", cookie)

// 	w := httptest.NewRecorder()
// 	r := httptest.NewRequest("GET", "http://example.com/exec?cmd=;%20/bin/bash", nil)
// 	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
// 	r.Header.Set("Cookie", cookie)
// 	r.Header.Set("User-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:146.0) Gecko/20100101 Firefox/146.0")
// 	handleProxy(w, r)

// 	if w.Code != rendering.UnworthyReturnCode && w.Code != http.StatusPaymentRequired {
// 		t.Errorf("expected %d, got %d", rendering.UnworthyReturnCode, w.Code)
// 	}

// }

// func TestWafBlockPhase4(t *testing.T) {
// 	// A test that should be stopped by the WAF

// 	backend := createMockUpstream()
// 	defer backend.Close()

// 	cookie, err := generateCookie(upstreamCfg.LoginRateLimiter)
// 	if err != nil {
// 		t.Errorf("encountered error on generating cookie: %v", err)
// 	}

// 	t.Logf("generated cookie: %s", cookie)

// 	w := httptest.NewRecorder()
// 	r := httptest.NewRequest("GET", "http://example.com/break4", nil)
// 	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
// 	r.Header.Set("Cookie", cookie)
// 	r.Header.Set("User-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:146.0) Gecko/20100101 Firefox/146.0")
// 	handleProxy(w, r)

// 	if w.Code != rendering.UnworthyReturnCode && w.Code != http.StatusPaymentRequired {
// 		t.Errorf("expected %d, got %d", rendering.UnworthyReturnCode, w.Code)
// 	}

// }
