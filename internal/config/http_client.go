package config

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	httpTimeout = 30 * time.Second
	// maxRedirects keeps net/http's own limit, which a custom CheckRedirect
	// replaces.
	maxRedirects = 10
	httpsScheme  = "https"
)

var errInsecureRedirect = errors.New("refusing a redirect from https to a non-https URL")

// newHTTPClient is the client every adapter gets: bounded in time, and never
// following a redirect off https. net/http copies the Authorization header to
// a redirect on the same host whatever its scheme, so following https → http
// would put the tracker token on the wire in clear.
func newHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout, CheckRedirect: refuseInsecureRedirect}
}

func refuseInsecureRedirect(next *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if via[0].URL.Scheme == httpsScheme && next.URL.Scheme != httpsScheme {
		return errInsecureRedirect
	}
	return nil
}
