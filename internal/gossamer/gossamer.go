package gossamer

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/corazawaf/coraza/v3/types"
)

type Settings struct {
	CookieName string `env:"COOKIE_NAME" envDefault:"_gSession"`
}

var settings Settings

func init() {
	if err := env.Parse(&settings); err != nil {
		panic(err)
	}
}

type Timing struct {
	Request          time.Time
	Validation       time.Time
	Preprocession    time.Time
	UpstreamRequest  time.Time
	UpstreamResponse time.Time
	Verification     time.Time
	Postprocession   time.Time
	Response         time.Time
}

func (t Timing) UpstreamDuration() int64 {
	// Returns the duration of the upstream request processing
	return t.UpstreamResponse.Sub(t.UpstreamRequest).Milliseconds()
}

func (t Timing) Total() int64 {
	return t.Response.Sub(t.Request).Milliseconds()
}

func (t Timing) ValidationTime() int64 {
	return t.Validation.Sub(t.Request).Milliseconds()
}

type Connection struct {
	Request     *http.Request
	Recorder    *httptest.ResponseRecorder
	Cookie      string
	IpAddress   string
	Transaction types.Transaction
	Timing      Timing
}

func (r *Connection) Load() error {

	lastIndex := strings.LastIndex(r.Request.RemoteAddr, ":")
	remoteIpWithBrackets := r.Request.RemoteAddr[:lastIndex]
	r.IpAddress = strings.Trim(remoteIpWithBrackets, "[]")

	cookie, err := r.Request.Cookie(settings.CookieName)
	if err == http.ErrNoCookie {
		r.Cookie = ""
	} else if err != nil {
		return err
	} else {
		r.Cookie = strings.Split(cookie.Value, ",")[0]
	}

	r.Recorder = httptest.NewRecorder()
	r.Timing.Request = time.Now()

	return nil
}

func (r *Connection) Close() error {
	if r.Transaction == nil {
		return nil
	}
	return r.Transaction.Close()
}

func (r *Connection) Log(logger *slog.Logger) {
	logger.Info(
		"finished connection successfully",
		"ip", r.IpAddress,
		"cookie", r.Cookie,
		"user_agent", r.Request.Header.Get("user-agent"),
		"url", r.Request.URL.String(),
		"code", r.Recorder.Code,
		"upstream_response_time_ms", r.Timing.UpstreamDuration(),
		"total_response_time_ms", r.Timing.Total(),
		"gossamer_overhead_ms", r.Timing.Total()-r.Timing.UpstreamDuration(),
	)
}
