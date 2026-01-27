package browser

import (
	"errors"
	"net/http"
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

func New(r *http.Request) (*Browser, error) {
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
