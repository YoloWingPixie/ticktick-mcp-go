package safety

import (
	"crypto/tls"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultRate  = 10
	DefaultBurst = 20
)

type RateLimitedTransport struct {
	limiter   *rate.Limiter
	transport http.RoundTripper
}

func NewRateLimitedTransport(transport http.RoundTripper, rps float64, burst int) *RateLimitedTransport {
	return &RateLimitedTransport{
		limiter:   rate.NewLimiter(rate.Limit(rps), burst),
		transport: transport,
	}
}

func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req)
}

func NewHTTPClient(transport http.RoundTripper, timeout time.Duration, rps float64, burst int) *http.Client {
	if transport == nil {
		transport = &http.Transport{
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			MaxIdleConnsPerHost:   4,
			ResponseHeaderTimeout: 5 * time.Second,
		}
	}

	rateLimited := NewRateLimitedTransport(transport, rps, burst)

	return &http.Client{
		Transport: rateLimited,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) == 0 {
				return nil
			}
			// Strip Authorization header when redirected to a different host
			original := via[0]
			if req.URL.Host != original.URL.Host {
				req.Header.Del("Authorization")
			}
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}
