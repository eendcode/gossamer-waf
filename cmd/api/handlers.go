package main

import (
	"context"
	"encoding/json"
	"errors"
	"gossamer/internal/gossamer"
	"gossamer/internal/helpers"
	"gossamer/internal/plugins"
	"gossamer/internal/plugins/coraza"
	"gossamer/pkg/hook"
	"gossamer/pkg/rendering"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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
	waf := pluginMap["coraza"].(*coraza.WAF)
	if waf == nil {
		raiseInternalServerError(w, errors.New("unable to start WAF"))
		return
	}

	tbi := waf.TokenBucketMap[r.Host]

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
	strikes, err := tbi.GetStrikes(context.Background(), cookie)
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

func handleMissingCookie(tbi coraza.TokenBucketImplementation, w http.ResponseWriter, c gossamer.Connection) {
	logger.Debug("no cookie set", "ip", c.IpAddress)

	// Check if we're dealing with AJAX
	if c.Request.Header.Get("X-Requested-With") != "" {
		// ajax
		w.WriteHeader(403)
		w.Write([]byte("NO"))
		return
	}

	allowed, remaining, err := tbi.Allow(context.Background(), c.IpAddress)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if !allowed {
		logger.Info("client hit rate limit", "ip", c.IpAddress)
		rendering.RenderForbidden(w, c.Request)
		return
	}

	newCookie, err := helpers.GenerateCookie()
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if err := tbi.CreateCookie(context.Background(), newCookie); err != nil {
		raiseInternalServerError(w, err)
		return
	}

	logger.Info("new client been given a cookie", "ip", c.IpAddress, "cookie", newCookie, "rate_limit_remaining", remaining)

	cmd := hook.NewCookie(newCookie, c.Request.RequestURI)
	w.Write([]byte(cmd.Script))
}

func handleProxy(w http.ResponseWriter, rq *http.Request) {

	// Load core plugins --------------------------------------------------------
	waf := pluginMap["coraza"].(*coraza.WAF)
	if waf == nil {
		raiseInternalServerError(w, errors.New("unable to start WAF"))
		return
	}

	// ---------------------------------------------------------------------------
	// Set up the `Connection` object
	c := gossamer.Connection{Request: rq, Transaction: waf.NewTransaction()}
	if err := c.Load(); err != nil {
		raiseInternalServerError(w, err)
		return
	}
	defer c.Close()

	// ----------------------------------------------------------------------------
	// Cookie interception
	cookie, err := helpers.GetCookie(rq)
	tbi := waf.TokenBucketMap[c.Request.Host]
	if err == http.ErrNoCookie {
		handleMissingCookie(waf.TokenBucketMap[c.Request.Host], w, c)
		return

	} else if err != nil {
		// unknown error -> 500
		raiseInternalServerError(w, err)
		return
	}

	cookieKnown, err := tbi.Exists(context.Background(), cookie)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if !cookieKnown {
		logger.Debug("client provided invalid cookie", "cookie", cookie, "ip", c.IpAddress)
		rendering.RenderForbidden(w, rq)
		return
	}

	// ------------------------------------------------------------------------------------------
	// Always strip the Accept-Encoding header. We can't properly inspect a package if it's encoded
	c.Request.Header.Del("Accept-Encoding")

	// ------------------------------------------------------------------------------------------
	// We now start the main course. We perform five stages
	// 1. Validation
	// 2. Preprocession
	// 3. Sending the response upstream and awaiting the response
	// 4. Verification
	// 5. Postprocession
	// before sending a response to the client.

	// Validators -------------------------------------------------------------------------------
	startValidation := time.Now()
	if !plugins.RunValidation(pluginMap, c) {

		logger.Debug(
			"received negative answer from validation",
			"cookie", c.Cookie,
			"url", c.Request.URL.String(),
		)

		// Increase the number of strikes for this cookie
		if err := tbi.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
			return
		}
		rendering.RenderForbidden(w, rq)
		return
	}

	endValidation := time.Now()

	logger.Debug(
		"validation phase ended",
		"cookie", c.Cookie,
		"ip_address", c.IpAddress,
		"duration_ms", endValidation.Sub(startValidation).Milliseconds(),
	)

	// Preprocessors -------------------------------------------------------------------------------
	startPreprocession := time.Now()
	ok, err := plugins.RunPreprocessor(pluginMap, c)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if !ok {

		logger.Debug(
			"received negative answer from preprocession",
			"cookie", c.Cookie,
			"url", c.Request.URL.String(),
		)

		// Increase the number of strikes for this cookie
		if err := tbi.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
			return
		}

		rendering.RenderForbidden(w, rq)
		return
	}

	endPreprocession := time.Now()
	logger.Debug(
		"preprocession phase ended",
		"cookie", c.Cookie,
		"ip_address", c.IpAddress,
		"duration_ms", endPreprocession.Sub(startPreprocession).Milliseconds(),
	)

	// Send to proxy -------------------------------------------------------------------------------
	px := httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {

			upstream := waf.FindMatchingUpstream(c)

			url, err := url.Parse(upstream.Url)
			if err != nil || url == nil {
				logger.Error(
					"unable to parse upstream URL",
					"error", err,
					"ip_address", c.IpAddress,
					"cookie", c.Cookie,
					"url", c.Request.RequestURI,
					"target", upstream.Url,
				)
			}

			logger.Debug(
				"rewriting URL for request",
				"ip_address", c.IpAddress,
				"cookie", c.Cookie,
				"url", c.Request.RequestURI,
				"target", url.String(),
			)
			pr.SetURL(url)

			pr.Out.Host = url.Host

			cookies := pr.In.Cookies()
			var remaining []string
			for _, c := range cookies {
				if c.Name != "_gSession" {
					remaining = append(remaining, c.Name+"="+c.Value)
				}
			}

			if len(remaining) > 0 {
				pr.Out.Header.Set("Cookie", strings.Join(remaining, "; "))
			} else {
				pr.Out.Header.Del("Cookie")
			}

			// 4. Set Forwarding Headers so phpMyAdmin knows the "Real" source
			pr.SetXForwarded()
		},
		// ModifyResponse: modifyFunc,
		ErrorHandler: noContextCancelationErrors,
	}

	c.Timing.UpstreamRequest = time.Now()
	px.ServeHTTP(c.Recorder, c.Request)
	c.Timing.UpstreamResponse = time.Now()

	// Verifiers -------------------------------------------------------------------------------
	startVerification := time.Now()
	if !plugins.RunVerification(pluginMap, c) {
		logger.Debug("received negative answer from verification", "cookie", c.Cookie, "url", c.Request.URL.String())

		// Increase the number of strikes for this cookie
		if err := tbi.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
		}
		rendering.RenderForbidden(w, rq)
		return
	}
	endVerification := time.Now()
	logger.Debug(
		"verification phase ended",
		"cookie", c.Cookie,
		"ip_address", c.IpAddress,
		"duration_ms", endVerification.Sub(startVerification).Milliseconds(),
	)

	// Postprocessors -------------------------------------------------------------------------------
	startPostprocession := time.Now()
	ok, err = plugins.RunPostprocessor(pluginMap, c)
	if err != nil {
		raiseInternalServerError(w, err)
		return
	}

	if !ok {

		logger.Debug(
			"received negative answer from postprocession",
			"cookie", c.Cookie,
			"url", c.Request.URL.String(),
		)

		// Increase the number of strikes for this cookie
		if err := tbi.IncreaseStrikes(context.Background(), cookie, 1); err != nil {
			raiseInternalServerError(w, err)
			return
		}

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

	// Since we may have modified the response, we delete the `Content-Length` header set by the upstream service, and let golang determine it for us.
	w.Header().Del("Content-Length")

	// Override the upstream server header, if we've chosen to do so.
	// The behaviour is set by the environment variable `OVERRIDE_UPSTREAM_SERVER_HEADER`
	// The new value of this header would be equal to `SERVER_HEADER`
	if settings.OverrideUpstreamServerHeader {
		w.Header().Del("server")
		w.Header().Add("server", settings.ServerHeader)
	}

	// Write the response. We reuse the response code from the upstream.
	w.WriteHeader(c.Recorder.Code)
	w.Write(c.Recorder.Body.Bytes())

	// Finally, we make a clean log of our connection
	c.Timing.Response = time.Now()
	c.Log(logger)
}

func stripWAFCookie(req *http.Request) {
	cookies := req.Cookies() // This parses the header into a slice
	if len(cookies) == 0 {
		return
	}

	var remaining []string
	for _, c := range cookies {
		// Only keep cookies that aren't your WAF session
		if c.Name != "_gSession" {
			remaining = append(remaining, c.String())
		}
	}

	// Update the header with the filtered list
	if len(remaining) > 0 {
		req.Header.Set("Cookie", strings.Join(remaining, "; "))
	} else {
		req.Header.Del("Cookie")
	}
}
