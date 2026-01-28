package main

import (
	"context"
	"encoding/json"
	"errors"
	"gossamer/internal/gossamer"
	"gossamer/internal/helpers"
	"gossamer/internal/plugins"
	"gossamer/internal/plugins/coraza"
	"gossamer/internal/plugins/ratelimit"
	"gossamer/pkg/hook"
	"gossamer/pkg/rendering"
	"net/http"
	"net/http/httputil"
	"time"
)

func raiseInternalServerError(w http.ResponseWriter, err error) {
	logger.Error("encountered an internal server error", "error", err)
	rendering.RenderInternalServerError(w, nil)
}

func noContextCancelationErrors(w http.ResponseWriter, r *http.Request, err error) {
	if err != context.Canceled {
		logger.Error("proxy error", "error", err)
	}
	rendering.RenderBadGateway(w, r)
}

func checkIn(w http.ResponseWriter, r *http.Request) {

	// Get the right plugin
	kvStore := pluginMap["ratelimit"].(*ratelimit.RateLimiter)
	if kvStore == nil {
		raiseInternalServerError(w, errors.New("unable to get key-value store"))
		return
	}

	cookie, err := helpers.GetCookie(r)
	if err == http.ErrNoCookie {
		// no cookie set -> 403

		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("put those foolish ambitions to rest"))
		return

	} else if err != nil {
		// unknown error -> 500
		raiseInternalServerError(w, err)
		return
	}

	logger.Debug("got cookie in hook", "cookie", cookie)

	// Check the cookies current strike level
	strikes, err := kvStore.KvStore.GetStrikes(context.Background(), cookie)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	logger.Debug("fetched client strikes", "strikes", strikes, "client", cookie)

	type response struct {
		Strikes int64 `json:"strikes"`
		Debug   bool  `json:"debug"`
	}

	res := response{Strikes: strikes, Debug: settings.HookDebugMode}

	data, err := json.Marshal(res)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)

}

func handleProxy(w http.ResponseWriter, rq *http.Request) {

	waf := pluginMap["coraza"].(*coraza.WAF)

	c := gossamer.Connection{Request: rq, Transaction: waf.NewTransaction()}
	if err := c.Load(); err != nil {
		raiseInternalServerError(w, err)
		return
	}
	defer c.Close()

	// Cookie interception
	kvStore := pluginMap["ratelimit"].(*ratelimit.RateLimiter)
	if kvStore == nil {
		raiseInternalServerError(w, errors.New("unable to get key-value store"))
		return
	}

	cookie, err := helpers.GetCookie(rq)
	if err == http.ErrNoCookie {
		// no cookie set - we set the cookie and do a quick redirect
		logger.Debug("no cookie set", "ip", c.IpAddress)

		allowed, remaining, err := kvStore.KvStore.Allow(context.Background(), c.IpAddress)
		if err != nil {
			raiseInternalServerError(w, err)
			return
		}

		if !allowed {
			logger.Info("client hit rate limit", "ip", c.IpAddress)
			rendering.RenderForbidden(w, rq)
			return
		}

		newCookie, err := helpers.GenerateCookie()
		if err != nil {
			raiseInternalServerError(w, err)
			return
		}

		if err := kvStore.KvStore.CreateCookie(context.Background(), newCookie); err != nil {
			raiseInternalServerError(w, err)
			return
		}

		logger.Debug("new client been given a cookie", "ip", c.IpAddress, "cookie", newCookie, "rate_limit_remaining", remaining)

		cmd := hook.NewCookie(newCookie, rq.RequestURI)
		w.Write([]byte(cmd.Script))
		return

	} else if err != nil {
		// unknown error -> 500
		raiseInternalServerError(w, err)
		return
	}

	// Always strip the Accept-Encoding header
	c.Request.Header.Del("Accept-Encoding")

	// Validators
	startValidation := time.Now()
	if !plugins.RunValidation(pluginMap, c) {
		logger.Debug("received negative answer from validation", "cookie", c.Cookie, "url", c.Request.URL.String())

		// Increase the number of strikes for this cookie
		if err := kvStore.KvStore.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
		}
		rendering.RenderForbidden(w, rq)
		return
	}

	endValidation := time.Now()
	logger.Debug("validation phase ended", "cookie", c.Cookie, "ip_address", c.IpAddress, "duration_ms", endValidation.Sub(startValidation).Milliseconds())

	// Preprocessors
	startPreprocession := time.Now()
	ok, err := plugins.RunPreprocessor(pluginMap, c)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if !ok {
		rendering.RenderForbidden(w, rq)
		return
	}

	// Send to proxy
	px := httputil.ReverseProxy{
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			// proxyRequest.SetURL(c.Request.URL)
			proxyRequest.Out.Host = proxyRequest.In.Host
		},
		ModifyResponse: modifyFunc,
		ErrorHandler:   noContextCancelationErrors,
	}

	endPreprocession := time.Now()
	logger.Debug("preprocession phase ended", "cookie", c.Cookie, "ip_address", c.IpAddress, "duration_ms", endPreprocession.Sub(startPreprocession).Milliseconds())

	c.Timing.UpstreamRequest = time.Now()
	px.ServeHTTP(c.Recorder, c.Request)
	c.Timing.UpstreamResponse = time.Now()

	// Verifiers
	startVerification := time.Now()
	if !plugins.RunVerification(pluginMap, c) {
		logger.Debug("received negative answer from verification", "cookie", c.Cookie, "url", c.Request.URL.String())

		// Increase the number of strikes for this cookie
		if err := kvStore.KvStore.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
		}
		rendering.RenderForbidden(w, rq)
		return
	}
	endVerification := time.Now()
	logger.Debug("verification phase ended", "cookie", c.Cookie, "ip_address", c.IpAddress, "duration_ms", endVerification.Sub(startVerification).Milliseconds())

	// Postprocessors
	startPostprocession := time.Now()
	ok, err = plugins.RunPostprocessor(pluginMap, c)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if !ok {
		rendering.RenderForbidden(w, rq)
		return
	}

	endPostprocession := time.Now()
	logger.Debug("postprocession phase ended", "cookie", c.Cookie, "ip_address", c.IpAddress, "duration_ms", endPostprocession.Sub(startPostprocession).Milliseconds())

	// Replay to the client
	logger.Debug("replaying response to client", "phase", 4)

	for k, vv := range c.Recorder.Header() {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Fix the content-length
	w.Header().Del("Content-Length")

	w.WriteHeader(c.Recorder.Code)
	w.Write(c.Recorder.Body.Bytes())
	c.Timing.Response = time.Now()

	c.Log(logger)
}
