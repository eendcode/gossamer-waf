package coraza_test

import (
	"gossamer/internal/gossamer"
	"gossamer/internal/plugins/coraza"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeBackend() *httptest.Server {
	mux := http.NewServeMux()

	// Give a 500, that should break in phase 3
	mux.HandleFunc("GET /break4", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("SQL error"))
	})

	return httptest.NewServer(mux)

}

func TestOk(t *testing.T) {
	waf, err := coraza.New()
	if err != nil {
		t.Errorf("error on initializing waf: %v", err)
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

	if !waf.Validate(gossamer.Connection{
		Request:  r,
		Recorder: w,
	}) {
		t.Errorf("waf did not validate valid request")
	}
}

func TestBadPayloadPhase1(t *testing.T) {
	waf, err := coraza.New()
	if err != nil {
		t.Errorf("error on initializing waf: %v", err)
	}

	backend := fakeBackend()
	defer backend.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com", nil)

	r.Header.Add("user-agent", "nikto")
	r.Header.Add("accept", "application/json")

	if waf.Validate(gossamer.Connection{
		Request:  r,
		Recorder: w,
	}) {
		t.Errorf("waf validated invalid request")
	}
}

func TestBadPayloadPhase2(t *testing.T) {
	waf, err := coraza.New()
	if err != nil {
		t.Errorf("error on initializing waf: %v", err)
	}

	backend := fakeBackend()
	defer backend.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com?id=;/bin/whoami", nil)

	r.Header.Add("user-agent", "foo/bar")
	r.Header.Add("accept", "application/json")

	if waf.Validate(gossamer.Connection{
		Request:  r,
		Recorder: w,
	}) {
		t.Errorf("waf validated invalid request")
	}
}

func TestBadPayloadPhase4(t *testing.T) {
	waf, err := coraza.New()
	if err != nil {
		t.Errorf("error on initializing waf: %v", err)
	}

	backend := fakeBackend()
	defer backend.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com/break4", nil)

	r.Header.Add("user-agent", "foo/bar")
	r.Header.Add("accept", "application/json")

	if waf.Validate(gossamer.Connection{
		Request:  r,
		Recorder: w,
	}) {
		t.Errorf("waf validated invalid request")
	}
}
