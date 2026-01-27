package browser

import (
	"errors"
	"gossamer/internal/gossamer"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNoHint               = errors.New("no client hint given")
	ErrUnexpectedHintValue  = errors.New("unexpected hint given")
	ErrBrowserInconsistency = errors.New("browser returns inconsistent information")
	ErrUnknownBrowser       = errors.New("unknown browser")
)

const (
	hintUa          string = "Sec-CH-UA"
	hintBitness     string = "Sec-CH-UA-Bitness"
	hintFullVersion string = "Sec-CH-UA-Full-Version"
	hintMobile      string = "Sec-CH-UA-Mobile"
	hintModel       string = "Sec-CH-UA-Model"
	hintPlatform    string = "Sec-CH-UA-Platform"
	hintDpr         string = "Sec-CH-DPR"
)

type BrowserType int64

const (
	Chrome BrowserType = iota
	Edge
	Firefox
	Safari
	Opera
	Unknown
)

type Browser struct {
	// The Raw User agent string
	UserAgent string

	UserAgentHint string

	// The type of browser
	BrowserType BrowserType

	// Version of the browser version
	Version string

	// Whether or not the user is on a mobile device
	IsMobile bool

	// Platform client hint
	Platform string `validate:"oneof=Android,Chrome OS,Chromium OS,iOS,Linux,macOS,Windows,Unknown"`

	// States whether the user is on a 32 or 64 bits device
	Bitness string `validate:"oneof=32,64"`

	// The device model
	Model string

	// Device to pixel ratio
	DPR float32
}

func (bt BrowserType) String() string {
	switch bt {
	case Chrome:
		return "Chrome"
	case Edge:
		return "Edge"
	case Firefox:
		return "Firefox"
	case Safari:
		return "Safari"
	case Opera:
		return "Opera"
	default:
		return "Unknown"
	}
}

func (b *Browser) DetermineType() error {
	// Determine the browser type

	sanitizedUserAgentHint := strings.ReplaceAll(b.UserAgentHint, "\"", "")
	// Collect all UA hints
	hintMap := make(map[string]string)

	for hint := range strings.SplitSeq(sanitizedUserAgentHint, ",") {
		if strings.Count(hint, ";") != 1 {
			// invalid brand
			continue
		}
		splitHint := strings.Split(hint, ";")
		brand := strings.ReplaceAll(splitHint[0], " ", "")
		v := splitHint[1]

		hintMap[brand] = v
	}

	if hintMap["MicrosoftEdge"] != "" {
		b.BrowserType = Edge
		return nil
	} else if hintMap["GoogleChrome"] != "" {
		b.BrowserType = Chrome
		return nil

	} else if hintMap["Opera"] != "" {
		b.BrowserType = Opera
		return nil
	} else if hintMap["Chromium"] != "" {
		b.BrowserType = Chrome
		return nil
	} else {
		if strings.Contains(b.UserAgent, "Firefox") {
			b.BrowserType = Firefox
			return nil
		} else if strings.Contains(b.UserAgent, "CriOS") {
			b.BrowserType = Chrome
			return nil
		} else if strings.Contains(b.UserAgent, "Safari") {
			b.BrowserType = Safari
			return nil
		}

	}
	b.BrowserType = Unknown

	return nil

}

func (b *Browser) ComplianceCheck() error {
	// Perform a compliance check to see if the browser doesn't hold back information it shouldn't

	if b.UserAgent == "" {
		return ErrBrowserInconsistency
	}

	sameBrowser, err := ParseUserAgentHeader(b.UserAgent)
	if err != nil {

		return err
	}

	switch b.BrowserType {
	case Unknown:
		return ErrUnknownBrowser

	case Firefox, Safari:
		// These browsers don't give client hints, so the check always passes.
		if sameBrowser.BrowserType != b.BrowserType {
			return ErrBrowserInconsistency
		}

		return nil

	default:
		if b.Platform == "Unknown" || b.UserAgentHint == "Unknown" || b.Version == "Unknown" || b.DPR == 0 {
			return ErrBrowserInconsistency
		}

	}

	if sameBrowser.IsMobile != b.IsMobile || sameBrowser.Platform != b.Platform || b.BrowserType != sameBrowser.BrowserType {

		return ErrBrowserInconsistency
	}

	return nil
}

func NewBrowser(r *http.Request) (*Browser, error) {
	// Parse an http request to get the browser information

	platform, err := parseHeader(hintPlatform, &r.Header)
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	ua, err := parseHeader(hintUa, &r.Header)
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	isMobile, err := parseMobile(r.Header.Get(hintMobile))
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	version, err := parseHeader(hintFullVersion, &r.Header)
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	model, err := parseHeader(hintModel, &r.Header)
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	bitness, err := parseHeader(hintBitness, &r.Header)
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	dpr, err := parseHeader(hintDpr, &r.Header)
	if err != nil && err != ErrNoHint {
		return nil, err
	}

	var dprInt float64
	if dpr == "Unknown" {
		dprInt = 0.0
	} else {
		dprInt, err = strconv.ParseFloat(dpr, 32)
		if err != nil {
			return nil, err
		}
	}

	browser := Browser{
		UserAgent:     r.UserAgent(),
		UserAgentHint: ua,
		IsMobile:      isMobile,
		Platform:      strings.ReplaceAll(platform, "\"", ""),
		Version:       strings.ReplaceAll(version, "\"", ""),
		Model:         strings.ReplaceAll(model, "\"", ""),
		Bitness:       bitness,
		DPR:           float32(dprInt),
	}

	err = browser.DetermineType()

	return &browser, err
}

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

type BrowserPlugin struct{}

func (b *BrowserPlugin) Validate(c gossamer.Connection) bool {
	// This is the only non-trivial function this plugin provides.

	browser, err := NewBrowser(c.Request)
	if err != nil {
		return false
	}

	if err := browser.ComplianceCheck(); err != nil {
		return false
	}

	return true
}

func (b *BrowserPlugin) Verify(_ gossamer.Connection) bool { return true }

func (b *BrowserPlugin) Preprocess(_ gossamer.Connection) error { return nil }

func (b *BrowserPlugin) Postprocess(_ gossamer.Connection) error { return nil }

func New() (*BrowserPlugin, error) {
	return &BrowserPlugin{}, nil
}
