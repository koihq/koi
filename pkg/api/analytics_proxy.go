package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// BrowserAnalyticsRes is the analytics configuration of /capabilities. It is null when the instance
// sends no analytics, and the app then loads no analytics code.
type BrowserAnalyticsRes struct {
	Key            string   `json:"key"`
	APIHost        string   `json:"apiHost"`
	UIHost         string   `json:"uiHost"`
	ExcludedEmails []string `json:"excludedEmails"`
}

func browserAnalyticsRes(browser *BrowserAnalytics) *BrowserAnalyticsRes {
	if browser == nil {
		return nil
	}

	excluded := browser.ExcludedEmails
	if excluded == nil {
		excluded = []string{}
	}

	return &BrowserAnalyticsRes{
		Key:            browser.Key,
		APIHost:        "/ingest",
		UIHost:         browser.UIHost,
		ExcludedEmails: excluded,
	}
}

// newIngestProxy forwards the app's analytics to PostHog from Koi's own origin, so the content
// security policy stays 'self' and blockers of third-party hosts do not drop the events. The
// session cookie and any credentials stay behind.
func newIngestProxy(ingestHost, assetsHost string) http.Handler {
	ingest, ingestErr := url.Parse(ingestHost)
	assets, assetsErr := url.Parse(assetsHost)
	if ingestErr != nil || assetsErr != nil {
		return http.NotFoundHandler()
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			path := strings.TrimPrefix(r.In.URL.Path, "/ingest")
			target := ingest
			if strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/array/") {
				target = assets
			}

			r.SetURL(target)
			r.Out.URL.Path = path
			r.Out.URL.RawPath = ""
			r.Out.Host = target.Host

			r.Out.Header.Del("Cookie")
			r.Out.Header.Del("Authorization")

			// PostHog derives the country from the visitor's address, kept from the proxies in front.
			if forwarded := r.In.Header.Values("X-Forwarded-For"); len(forwarded) > 0 {
				r.Out.Header["X-Forwarded-For"] = forwarded
			}
			r.SetXForwarded()
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Del("Set-Cookie")

			return nil
		},
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   16,
		},
	}

	return proxy
}
