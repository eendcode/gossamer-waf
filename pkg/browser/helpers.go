package browser

import (
	"net/http"
	"regexp"
	"strings"
)

func parseMobile(h string) (bool, error) {

	switch h {
	case "":
		return false, ErrNoHint

	case "?0":
		return false, nil

	case "?1":
		return true, nil

	default:
		return false, ErrUnexpectedHintValue
	}
}

func parseHeader(h string, m *http.Header) (string, error) {

	switch v := m.Get(h); v {
	case "":
		return "Unknown", ErrNoHint
	default:
		return v, nil
	}
}

func ParseUserAgentHeader(userAgent string) (*Browser, error) {
	// Parse the user agent header and get all information from there
	pattern := `Mozilla/(4|5)\.0 \((?P<system_information>.*?)\) (?P<platform>[^()]+?)(?: \((?P<platform_details>.*?)\))? (?P<extensions>.*)`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := re.FindStringSubmatch(userAgent)

	if len(matches) == 0 {
		return nil, ErrUnknownBrowser
	}

	systemInfo := matches[2]

	platformSnippets := strings.Split(systemInfo, ";")
	platform, isMobile := searchPlatformSnippets(platformSnippets)

	extensions := strings.Split(matches[5], " ")
	browserType := searchExtensions(extensions)

	return &Browser{
		Platform:    strings.ReplaceAll(platform, "\"", ""),
		UserAgent:   userAgent,
		IsMobile:    isMobile,
		BrowserType: browserType,
	}, nil
}

const (
	pAndroid = 1 << iota
	pCrOS
	pWindows
	pMac
	pLinux
	pIOS
)

func searchPlatformSnippets(snippets []string) (string, bool) {
	var seen int

	for _, possiblyCapitalizedSnip := range snippets {
		snip := strings.ToLower(possiblyCapitalizedSnip)
		if strings.Contains(snip, "android") {
			seen |= pAndroid
		}
		if strings.Contains(snip, "cros") {
			seen |= pCrOS
		}
		if strings.Contains(snip, "windows") {
			seen |= pWindows
		}
		if strings.Contains(snip, "macintosh") {
			seen |= pMac
		}
		if strings.Contains(snip, "linux") || strings.Contains(snip, "x11") {
			seen |= pLinux
		}
		if strings.Contains(snip, "ipad") ||
			strings.Contains(snip, "ipod") ||
			strings.Contains(snip, "iphone") {
			seen |= pIOS
		}
	}

	switch {
	case seen&pAndroid != 0:
		return "Android", true
	case seen&pCrOS != 0:
		return "Chrome OS", false
	case seen&pWindows != 0:
		return "Windows", false
	case seen&pMac != 0:
		return "macOS", false
	case seen&pLinux != 0:
		return "Linux", false
	case seen&pIOS != 0:
		return "iOS", true
	default:
		return "Unknown", false
	}
}

const (
	pFirefox = 1 << iota
	pOpera
	pEdge
	pChrome
	pSafari
)

func searchExtensions(snippets []string) BrowserType {
	// Returns browser type + version

	var seen int

	for _, snip := range snippets {
		splitSnip := strings.Split(snip, "/")
		if len(splitSnip) != 2 {
			// unable to parse - don't take into consideration
			continue
		}

		product := splitSnip[0]

		switch v := strings.ToLower(product); v {
		case "firefox":
			seen |= pFirefox
		case "edge":
			seen |= pEdge
		case "opr":
			seen |= pOpera
		case "chromium", "chrome", "crios":
			seen |= pChrome
		case "safari":
			seen |= pSafari
		}
	}

	switch {
	case seen&pFirefox != 0:
		return Firefox
	case seen&pEdge != 0:
		return Edge
	case seen&pOpera != 0:
		return Opera
	case seen&pChrome != 0:
		return Chrome
	case seen&pSafari != 0:
		return Safari
	default:
		return Unknown
	}

}
