package browser_test

import (
	"gossamer/internal/gossamer"
	"gossamer/internal/plugins/browser"
	"net/http/httptest"
	"testing"
)

func TestFirefox(t *testing.T) {
	b := browser.BrowserPlugin{}

	r := httptest.NewRequest("GET", "/", nil)
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X x.y; rv:42.0) Gecko/20100101 Firefox/42.0"

	r.Header.Set("user-agent", userAgent)
	c := gossamer.Connection{Request: r}

	if !b.Validate(c) {
		t.Errorf("plugin incorrectly invalidated browser %s", userAgent)
	}
}

func TestChrome(t *testing.T) {
	b := browser.BrowserPlugin{}
	r := httptest.NewRequest("GET", "/", nil)

	r.Header.Set("user-agent", "Mozilla/5.0 (iPad; CPU OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/143.0.7499.108 Mobile/15E148 Safari/604.1")
	r.Header.Set("sec-ch-dpr", "1.0")
	r.Header.Set("sec-ch-ua", `" Not A;Brand";v="99", "Chromium";v="143", "Google Chrome";v="143"`)
	r.Header.Set("sec-ch-ua-full-version", "143.0.7499.108")
	r.Header.Set("sec-ch-ua-model", "iPad Pro 2025")
	r.Header.Set("sec-ch-ua-bitness", "64")
	r.Header.Set("sec-ch-ua-mobile", "?1")
	r.Header.Set("sec-ch-ua-platform", "iOS")

	c := gossamer.Connection{Request: r}

	if !b.Validate(c) {
		t.Errorf("plugin incorrectly invalidated browser")
	}

}

func TestFaultyChrome(t *testing.T) {
	b := browser.BrowserPlugin{}

	r := httptest.NewRequest("GET", "/", nil)

	r.Header.Set("user-agent", "Mozilla/5.0 (iPad; CPU OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/143.0.7499.108 Mobile/15E148 Safari/604.1")
	r.Header.Set("sec-ch-ua", `" Not A;Brand";v="99", "Chromium";v="143", "Google Chrome";v="143"`)
	r.Header.Set("sec-ch-ua-full-version", "143.0.7499.108")
	r.Header.Set("sec-ch-ua-model", "iPad Pro 2025")
	r.Header.Set("sec-ch-ua-bitness", "64")
	r.Header.Set("sec-ch-ua-mobile", "?1")
	r.Header.Set("sec-ch-ua-platform", "iOS")

	c := gossamer.Connection{Request: r}

	if b.Validate(c) {
		t.Errorf("plugin incorrectly validated browser")
	}

}

func TestClassification(t *testing.T) {

	browsers := map[string]browser.Browser{
		"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:38.0) Gecko/20100101 Firefox/38.0":                                                                                  {BrowserType: browser.Firefox, Platform: "Windows"},
		"Mozilla/5.0 (Windows NT 6.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/45.0.2454.85 Safari/537.36":                                                     {BrowserType: browser.Chrome, Platform: "Windows"},
		"Mozilla/5.0 (iPad; CPU OS 8_1_2 like Mac OS X) AppleWebKit/600.1.4 (KHTML, like Gecko) Version/8.0 Mobile/12B440 Safari/600.1.4":                           {BrowserType: browser.Safari, Platform: "iOS"},
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.10; rv:40.0) Gecko/20100101 Firefox/40.0":                                                                        {BrowserType: browser.Firefox, Platform: "macOS"},
		"Mozilla/5.0 (X11; CrOS x86_64 7077.134.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.156 Safari/537.36":                                       {BrowserType: browser.Chrome, Platform: "Chrome OS"},
		"Mozilla/5.0 (iPhone; CPU iPhone OS 8_4_1 like Mac OS X) AppleWebKit/600.1.4 (KHTML, like Gecko) Version/8.0 Mobile/12H321 Safari/600.1.4":                  {BrowserType: browser.Safari, Platform: "iOS"},
		"Mozilla/5.0 (Linux; U; Android 4.4.3; en-us; KFTHWI Build/KTU84M) AppleWebKit/537.36 (KHTML, like Gecko) Silk/3.68 like Chrome/39.0.2171.93 Safari/537.36": {BrowserType: browser.Chrome, Platform: "Android"},
		"Mozilla/5.0 (Windows NT 6.3; ARM; Trident/7.0; Touch; rv:11.0) like Gecko":                                                                                 {BrowserType: browser.Unknown, Platform: "Windows"},
		"Mozilla/5.0 (X11; FC Linux i686; rv:24.0) Gecko/20100101 Firefox/24.0":                                                                                     {BrowserType: browser.Firefox, Platform: "Linux"},
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 OPR/124.0.0.0":                                       {BrowserType: browser.Opera, Platform: "Linux"},
		"curl/8.17.0": {BrowserType: browser.Unknown, Platform: "Unknown"},
	}

	for k, v := range browsers {
		parsedBrowser, err := browser.ParseUserAgentHeader(k)
		if err != nil || parsedBrowser == nil {
			if v.BrowserType == browser.Unknown {
				t.Logf("identified correctly as unknown browser")
				continue
			}
			t.Errorf("encountered error when parsing user-agent %s: %v", k, err)
		}

		t.Logf("parsed browser for %s: %v", k, parsedBrowser)

		if v.BrowserType != parsedBrowser.BrowserType || v.Platform != parsedBrowser.Platform {
			t.Errorf("classification error for %s. got type %s, while expecting %s, and got platform %s while expecting %s", k, parsedBrowser.BrowserType.String(), v.BrowserType.String(), parsedBrowser.Platform, v.Platform)
		}
	}

}
