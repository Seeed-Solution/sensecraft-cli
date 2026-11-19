package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxHTTPSRedirects = 10

// externalHTTPClient is the client for OSS uploads and model downloads.
// Tests replace it so httptest certificates are trusted.
var externalHTTPClient = pinnedHTTPSClient

func pinnedHTTPSClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: refuseCrossHostRedirect,
	}
}

func requireHTTPSURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return errors.New("only https URLs are allowed")
	}
	return nil
}

func refuseCrossHostRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxHTTPSRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if len(via) == 0 || via[0].URL == nil || req == nil || req.URL == nil {
		return errors.New("refusing redirect")
	}
	if req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname()) {
		return fmt.Errorf("refusing redirect to %s", req.URL.Host)
	}
	return nil
}
