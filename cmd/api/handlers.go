package main

import (
	"context"
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

// func newCookie(w http.ResponseWriter, r *http.Request) {
// 	// Get the right plugin
// 	kvStore := pluginMap["ratelimit"].(*ratelimit.RateLimiter)
// 	if kvStore == nil {
// 		raiseInternalServerError(w, errors.New("unable to get key-value store"))
// 		return
// 	}

// 	// Check if the rate limiter allows this on an IP basis
// 	// ip, _ := helpers.ParseRemoteAddr(r.RemoteAddr)
// 	// allowed, _, err := kvStore.KvStore.Allow(context.Background(), ip)
// 	// if err != nil {
// 	// 	raiseInternalServerError(w, err)
// 	// 	return
// 	// }

// 	// if !allowed()

// 	cookie, err := helpers.GenerateCookie()
// 	if err != nil {
// 		raiseInternalServerError(w, errors.New("unable to generate new cookie"))
// 	}
// 	kvStore.KvStore.CreateCookie(context.TODO(), cookie)

// 	cmd := hook.NewCookie(cookie)
// 	w.Header().Add("content-type", "text/javascript")
// 	w.WriteHeader(200)
// 	w.Write([]byte(cmd.Script))

// }

func checkIn(w http.ResponseWriter, r *http.Request) {

	// Get the right plugin
	kvStore := pluginMap["ratelimit"].(*ratelimit.RateLimiter)
	if kvStore == nil {
		raiseInternalServerError(w, errors.New("unable to get key-value store"))
		return
	}

	cookie, err := helpers.GetCookie(r)
	if err != http.ErrNoCookie {
		// no cookie set -> 403

		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("put those foolish ambitions to rest"))
		return

	} else if err != nil {
		// unknown error -> 500
		raiseInternalServerError(w, err)
		return
	}

	// Check the cookies current strike level
	strikes, err := kvStore.KvStore.GetStrikes(context.Background(), cookie)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	// Act accordingly to the strike level
	// TODO: expand on this
	if strikes >= 20 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(hook.Blur().Script))
	}

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
		newCookie, err := helpers.GenerateCookie()
		if err != nil {
			raiseInternalServerError(w, err)
			return
		}

		if err := kvStore.KvStore.CreateCookie(context.Background(), newCookie); err != nil {
			raiseInternalServerError(w, err)
			return
		}

		cmd := hook.NewCookie(newCookie, rq.RequestURI)
		w.Write(cmd.ToHtml())
		return

	} else if err != nil {
		// unknown error -> 500
		raiseInternalServerError(w, err)
		return
	}

	// Validators
	if !plugins.RunValidation(pluginMap, c) {
		logger.Debug("received negative answer from validation", "cookie", c.Cookie, "url", c.Request.URL.String())

		// Increase the number of strikes for this cookie
		if err := kvStore.KvStore.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
		}
		rendering.RenderForbidden(w, rq)
		return
	}

	// Preprocessors
	if err := plugins.RunPreprocessor(pluginMap, c); err != nil {
		raiseInternalServerError(w, err)
		return
	}

	// Send to proxy
	px := httputil.ReverseProxy{
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			proxyRequest.SetURL(c.Request.URL)
			proxyRequest.Out.Host = proxyRequest.In.Host
		},
		ModifyResponse: modifyFunc,
		ErrorHandler:   noContextCancelationErrors,
	}

	c.Timing.UpstreamRequest = time.Now()
	px.ServeHTTP(c.Recorder, c.Request)
	c.Timing.UpstreamResponse = time.Now()

	// Verifiers
	if !plugins.RunVerification(pluginMap, c) {
		logger.Debug("received negative answer from verification", "cookie", c.Cookie, "url", c.Request.URL.String())

		// Increase the number of strikes for this cookie
		if err := kvStore.KvStore.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
		}
		rendering.RenderForbidden(w, rq)
		return
	}

	// Postprocessors
	if err := plugins.RunPostprocessor(pluginMap, c); err != nil {
		raiseInternalServerError(w, err)
		return
	}

	// Replay to the client
	logger.Debug("replaying response to client", "phase", 4)

	for k, vv := range c.Recorder.Header() {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(c.Recorder.Code)
	w.Write(c.Recorder.Body.Bytes())
	c.Timing.Response = time.Now()

	c.Log(logger)
}
