package cli

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/client"
)

// T7a: redactHandler must scrub Bearer tokens and URL credentials from
// both the message and any string attribute, no matter how the caller
// names the field. Anything else (caller forgot to redact) must still be
// caught by this handler.
func TestRedactHandler_ScrubsBearerAndCreds(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(redactHandler{inner: base})

	logger.LogAttrs(context.Background(), slog.LevelDebug, "Bearer ctx7sk-leaky-key was sent",
		slog.String("auth", "Bearer ctx7sk-also-leaky"),
		slog.String("dsn", "postgres://u:s3cret@host/db"),
		slog.String("safe", "no secret here"),
	)

	out := buf.String()
	for _, leaked := range []string{"ctx7sk-leaky-key", "ctx7sk-also-leaky", "s3cret"} {
		if strings.Contains(out, leaked) {
			t.Errorf("redactor missed %q in output: %s", leaked, out)
		}
	}
	if !strings.Contains(out, "Bearer ***") {
		t.Errorf("expected Bearer *** marker; got: %s", out)
	}
	if !strings.Contains(out, "no secret here") {
		t.Errorf("benign attr was dropped: %s", out)
	}
}

// T7: --debug must produce structured request logs on stderr, but the
// Authorization bearer token must never appear in plaintext.
func TestDebug_RedactsBearer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	const secret = "ctx7sk-thisShouldNeverAppearOnStderr"
	prev := newAPIClient
	newAPIClient = func(cmd *cobra.Command) *client.Client {
		return client.New(client.Options{
			BaseURL:    srv.URL,
			HTTPClient: srv.Client(),
			MaxRetries: 0,
			APIKey:     secret,
			Logger:     newDebugLogger(cmd),
		})
	}
	t.Cleanup(func() { newAPIClient = prev })

	var stdout, stderr bytes.Buffer
	// --no-cache so we don't read leftover entries from a previous dev run
	// (real os.UserCacheDir is in play here — the helper's temp-cache
	// override is intentionally NOT used for this test).
	exit := run(&stdout, &stderr, []string{"--debug", "--no-cache", "--api-key", secret, "resolve", "anything"})
	// resolve returns errNoResults (exit 1) on empty result set — fine for
	// our purposes; we only care about the debug log content.
	_ = exit

	if !strings.Contains(stderr.String(), "GET") {
		t.Errorf("expected debug log entry with GET, got stderr=%q", stderr.String())
	}
	if strings.Contains(stderr.String(), secret) {
		t.Errorf("bearer secret leaked into stderr: %q", stderr.String())
	}
}
