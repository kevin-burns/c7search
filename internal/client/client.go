// Package client is the typed HTTP wrapper around context7.com/api/v1.
//
// Two public methods mirror the Context7 MCP server's two tools:
//
//	Search(ctx, query)     → resolve-library-id
//	Docs(ctx, id, opts)    → get-library-docs
//
// All external requests run under a bounded context.WithTimeout — a hostile
// or slow upstream must not be able to wedge the CLI. Errors returned from
// this package are pre-redacted so callers can print them directly.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kevin-burns/c7search/internal/redact"
	"github.com/kevin-burns/c7search/internal/version"
)

const DefaultBaseURL = "https://context7.com"

// Per-call timeouts framed as security controls, not just perf knobs.
const (
	TimeoutResolve = 10 * time.Second
	TimeoutDocs    = 30 * time.Second
	TimeoutAuth    = 5 * time.Second
)

// Options configures a Client. Zero values are usable.
type Options struct {
	APIKey     string
	BaseURL    string       // default: DefaultBaseURL
	HTTPClient *http.Client // default: &http.Client{}; set in tests for httptest server
	UserAgent  string       // default: version.UserAgent()
	MaxRetries int          // default: 3 (set 0 in tests)
	Now        func() time.Time
	// Logger is optional. When set, the client emits structured debug
	// records for each request and response. Callers SHOULD wrap their
	// handler with a redacting handler — the client will not double-redact
	// values passed to the handler. Bodies are NOT logged.
	Logger *slog.Logger
}

// Client talks to the Context7 HTTP API. Construct via New.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	userAgent  string
	maxRetries int
	now        func() time.Time
	log        *slog.Logger
}

// New constructs a Client. APIKey is optional — anonymous tier works for
// search and docs, just rate-limited (see .notes/api-probes.md).
func New(opts Options) *Client {
	c := &Client{
		httpClient: opts.HTTPClient,
		baseURL:    strings.TrimRight(opts.BaseURL, "/"),
		apiKey:     opts.APIKey,
		userAgent:  opts.UserAgent,
		maxRetries: opts.MaxRetries,
		now:        opts.Now,
		log:        opts.Logger,
	}
	if c.log == nil {
		c.log = slog.New(discardHandler{})
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	if c.userAgent == "" {
		c.userAgent = version.UserAgent()
	}
	if c.maxRetries == 0 {
		c.maxRetries = 3
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c
}

// do performs a request with retry on 5xx and 429. The body is read fully so
// the caller can inspect it on success and on error. err is already redacted
// for safe printing.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, accept string) (*http.Response, []byte, error) {
	full := c.baseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, full, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		c.log.Debug("request",
			slog.String("method", method),
			slog.String("url", redact.RedactURL(full)),
			slog.Int("attempt", attempt))

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http: %s", redact.RedactBearer(redact.RedactURL(err.Error())))
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		c.log.Debug("response",
			slog.Int("status", resp.StatusCode),
			slog.Int("bytes", len(body)),
			slog.Int("attempt", attempt))
		if readErr != nil {
			lastErr = fmt.Errorf("read body: %w", readErr)
			continue
		}

		// Retryable statuses
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = mapStatusError(resp, body)
			if attempt < c.maxRetries {
				continue
			}
			return resp, body, lastErr
		}

		if resp.StatusCode >= 400 {
			return resp, body, mapStatusError(resp, body)
		}

		return resp, body, nil
	}
	return nil, nil, lastErr
}

// backoff returns the wait time between retries: 250ms, 500ms, 1s, 2s,
// 4s, 8s, 16s, 16s, ... with up to 25% additive jitter. Capped at 16s so
// a misconfigured MaxRetries (or future bump) cannot produce negative
// durations via shift overflow or cause a slow client to wait minutes.
func backoff(attempt int) time.Duration {
	const (
		base    = 250 * time.Millisecond
		ceiling = 16 * time.Second
		maxStep = 6 // 250ms<<6 = 16s
	)
	step := attempt - 1
	if step < 0 {
		step = 0
	}
	if step > maxStep {
		step = maxStep
	}
	d := base << uint(step)
	if d > ceiling {
		d = ceiling
	}
	if jitter := int64(d / 4); jitter > 0 {
		d += time.Duration(rand.Int64N(jitter))
	}
	return d
}

// mapStatusError converts an HTTP error response into a typed sentinel,
// preserving useful detail from the body (which may be JSON or plain text —
// 404 from this API is plain text).
func mapStatusError(resp *http.Response, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}

	// JSON errors carry a structured message
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var je struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &je) == nil && je.Message != "" {
			msg = je.Message
		}
	}

	msg = redact.RedactBearer(redact.RedactURL(msg))

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%s: %w", msg, ErrNotFound)
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%s: %w", msg, ErrUnauthorized)
	case resp.StatusCode == http.StatusTooManyRequests:
		retry := resp.Header.Get("Retry-After")
		if retry == "" {
			if reset := resp.Header.Get("Ratelimit-Reset"); reset != "" {
				if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
					retry = fmt.Sprintf("%ds", epoch-time.Now().Unix())
				}
			}
		}
		return fmt.Errorf("%s (retry-after=%s): %w", msg, retry, ErrRateLimited)
	case resp.StatusCode >= 500:
		return fmt.Errorf("%s: %w", msg, ErrServer)
	default:
		return fmt.Errorf("%s: %w", msg, ErrBadRequest)
	}
}
