package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/client"
)

// T2: when the API rejects the key (401), the user should see "status:
// invalid key" once on stdout and NO duplicate "error: ..." line on
// stderr. Process must still exit 3.
func TestAuthStatus_InvalidKey_NoStderrEcho(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key","message":"Invalid API key"}`))
	}))
	t.Cleanup(srv.Close)

	prev := newAPIClient
	newAPIClient = func(cmd *cobra.Command) *client.Client {
		return client.New(client.Options{
			BaseURL:    srv.URL,
			HTTPClient: srv.Client(),
			MaxRetries: 0,
			APIKey:     "ctx7sk-bogus",
		})
	}
	t.Cleanup(func() { newAPIClient = prev })

	var stdout, stderr bytes.Buffer
	exit := run(&stdout, &stderr, []string{"--api-key", "ctx7sk-bogus", "auth", "status"})

	if exit != ExitAuth {
		t.Errorf("expected exit code %d, got %d", ExitAuth, exit)
	}
	if !strings.Contains(stdout.String(), "status: invalid key") {
		t.Errorf("status line missing from stdout; stdout=%q", stdout.String())
	}
	if strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr should not echo error after status line; stderr=%q", stderr.String())
	}
}
