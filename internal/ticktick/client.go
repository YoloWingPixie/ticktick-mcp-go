package ticktick

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	BaseURL                     = "https://api.ticktick.com/open/v1"
	MaxRetries                  = 3
	InitialBackoff              = 500 * time.Millisecond
	MaxBackoff                  = 10 * time.Second
	MaxConcurrentProjectFetches = 8
)

var ErrUnauthorized = errors.New("ticktick: unauthorized (token may be expired)")

type Client struct {
	http      *http.Client
	baseURL   string
	userAgent string
}

func NewClient(httpClient *http.Client, version string) *Client {
	return &Client{
		http:      httpClient,
		baseURL:   BaseURL,
		userAgent: "ticktick-mcp-go/" + version,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 10 MiB cap to prevent memory exhaustion from unexpected large responses
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return data, nil
	}

	apiErr := &APIError{StatusCode: resp.StatusCode, RetryAfter: resp.Header.Get("Retry-After")}
	if jsonErr := json.Unmarshal(data, apiErr); jsonErr != nil {
		apiErr.Err = http.StatusText(resp.StatusCode)
		apiErr.Path = path
	}
	if apiErr.StatusCode == 0 {
		apiErr.StatusCode = resp.StatusCode
	}

	return nil, apiErr
}

func (c *Client) doWithRetry(ctx context.Context, safety OperationSafety, method, path string, body []byte) ([]byte, error) {
	maxAttempts := 1
	if safety != NonIdempotentWrite {
		maxAttempts = MaxRetries + 1
	}

	var lastErr error
	backoff := InitialBackoff

	for attempt := range maxAttempts {
		if attempt > 0 {
			jitter := time.Duration(rand.Int64N(int64(backoff) / 2))
			sleep := backoff + jitter
			slog.Debug("retrying request", "method", method, "path", path, "attempt", attempt+1, "backoff", sleep)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleep):
			}

			backoff *= 2
			if backoff > MaxBackoff {
				backoff = MaxBackoff
			}
		}

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		data, err := c.do(ctx, method, path, bodyReader)
		if err == nil {
			return data, nil
		}

		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode == http.StatusUnauthorized {
				return nil, ErrUnauthorized
			}

			if apiErr.StatusCode == http.StatusTooManyRequests {
				if safety == NonIdempotentWrite {
					return nil, err
				}
				if retryAfter := retryAfterFromHeader(apiErr.RetryAfter); retryAfter > 0 {
					backoff = retryAfter
				}
				lastErr = err
				continue
			}

			if apiErr.StatusCode >= 500 {
				if safety == NonIdempotentWrite {
					return nil, err
				}
				lastErr = err
				continue
			}

			// 4xx (other than 401/429) — not retryable
			return nil, err
		}

		// Transport error — only retry for safe reads
		if safety == SafeRead {
			lastErr = err
			continue
		}
		return nil, err
	}

	return nil, lastErr
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	return c.doWithRetry(ctx, SafeRead, http.MethodGet, path, nil)
}

func (c *Client) postWithSafety(ctx context.Context, safety OperationSafety, path string, body any) ([]byte, error) {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}
	return c.doWithRetry(ctx, safety, http.MethodPost, path, data)
}

func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	return c.postWithSafety(ctx, NonIdempotentWrite, path, body)
}

func (c *Client) postIdempotent(ctx context.Context, path string, body any) ([]byte, error) {
	return c.postWithSafety(ctx, IdempotentWrite, path, body)
}

func (c *Client) del(ctx context.Context, path string) error {
	_, err := c.doWithRetry(ctx, NonIdempotentWrite, http.MethodDelete, path, nil)
	return err
}

func retryAfterFromHeader(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
