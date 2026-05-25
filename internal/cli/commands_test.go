package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/cache"
	"github.com/kevin-burns/c7search/internal/client"
)

// fakeAPI installs a test HTTP server and rebinds newAPIClient/newCacheStore
// for the duration of t. Returns the server so individual tests can swap
// handlers when they need per-test behavior.
type fakeAPI struct {
	srv     *httptest.Server
	handler http.Handler
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.handler.ServeHTTP(w, r)
	}))
	t.Cleanup(f.srv.Close)

	prevAPI := newAPIClient
	newAPIClient = func(cmd *cobra.Command) *client.Client {
		return client.New(client.Options{
			BaseURL:    f.srv.URL,
			HTTPClient: f.srv.Client(),
			MaxRetries: 0,
			APIKey:     resolveAPIKey(cmd),
			Logger:     newDebugLogger(cmd),
		})
	}
	t.Cleanup(func() { newAPIClient = prevAPI })

	prevCache := newCacheStore
	tmp := t.TempDir()
	newCacheStore = func(cmd *cobra.Command) *cache.FS {
		if noCacheFlag(cmd) {
			return nil
		}
		return &cache.FS{Dir: tmp}
	}
	t.Cleanup(func() { newCacheStore = prevCache })

	return f
}

func (f *fakeAPI) handle(h http.HandlerFunc) { f.handler = h }

// ---- resolve ---------------------------------------------------------------

func TestResolve_Happy_Text(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/search") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"/vercel/next.js","trustScore":10,"totalSnippets":42,"stars":1000}]}`))
	})

	var stdout, stderr bytes.Buffer
	exit := run(&stdout, &stderr, []string{"resolve", "next.js"})
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "/vercel/next.js") {
		t.Errorf("stdout missing id: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "trust=10.0") {
		t.Errorf("stdout missing trust: %s", stdout.String())
	}
}

func TestResolve_Empty_ExitOne(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	})

	var stdout, stderr bytes.Buffer
	exit := run(&stdout, &stderr, []string{"resolve", "totally-fake-lib"})
	if exit != ExitNoResults {
		t.Errorf("expected exit %d, got %d (stderr=%s)", ExitNoResults, exit, stderr.String())
	}
}

func TestResolve_Unauthorized_ExitThree(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Invalid API key"}`))
	})

	exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"resolve", "x"})
	if exit != ExitAuth {
		t.Errorf("expected exit %d, got %d", ExitAuth, exit)
	}
}

func TestResolve_RateLimited_ExitTwo(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	})

	exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"resolve", "x"})
	if exit != ExitAPI {
		t.Errorf("expected exit %d, got %d", ExitAPI, exit)
	}
}

func TestResolve_JSONFormat(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"/a/b","trustScore":7}]}`))
	})

	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"--json", "resolve", "x"}); exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, stdout.String())
	}
	if len(got) != 1 || got[0]["id"] != "/a/b" {
		t.Errorf("unexpected JSON: %+v", got)
	}
}

func TestResolve_NoCache_BypassesCache(t *testing.T) {
	f := newFakeAPI(t)
	calls := 0
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"/a/b","trustScore":7}]}`))
	})

	for i := 0; i < 3; i++ {
		if exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"--no-cache", "resolve", "x"}); exit != 0 {
			t.Fatalf("iter %d exit %d", i, exit)
		}
	}
	if calls != 3 {
		t.Errorf("--no-cache did not bypass cache: calls=%d", calls)
	}
}

func TestResolve_CacheHit_SuppressesSecondRequest(t *testing.T) {
	f := newFakeAPI(t)
	calls := 0
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"/a/b","trustScore":7}]}`))
	})

	for i := 0; i < 3; i++ {
		if exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"resolve", "same-query"}); exit != 0 {
			t.Fatalf("iter %d exit %d", i, exit)
		}
	}
	if calls != 1 {
		t.Errorf("expected 1 upstream call (rest from cache), got %d", calls)
	}
}

// T12: --library-name routes resolve through the v2 endpoint, passing
// libraryName separately from the relevance query.
func TestResolve_LibraryName_RoutesToV2(t *testing.T) {
	f := newFakeAPI(t)
	var path string
	var libraryName, query string
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		libraryName = r.URL.Query().Get("libraryName")
		query = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"/vercel/next.js","trustScore":9}]}`))
	})

	exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{
		"resolve", "--library-name", "next.js", "app router",
	})
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if path != "/api/v2/libs/search" {
		t.Errorf("expected v2 path, got %s", path)
	}
	if libraryName != "next.js" {
		t.Errorf("libraryName: %s", libraryName)
	}
	if query != "app router" {
		t.Errorf("query: %s", query)
	}
}

// ---- docs ------------------------------------------------------------------

func TestDocs_Happy_Text(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/context" {
			t.Errorf("unexpected path: %s (expected /api/v2/context)", r.URL.Path)
		}
		if got := r.URL.Query().Get("libraryId"); got != "/vercel/next.js" {
			t.Errorf("libraryId: %s", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# next.js docs\nhello"))
	})

	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"docs", "/vercel/next.js"}); exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if !strings.HasPrefix(stdout.String(), "# next.js docs") {
		t.Errorf("body not streamed: %q", stdout.String())
	}
}

func TestDocs_NotFound_ExitTwo(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Library not found"))
	})

	exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"docs", "/missing/lib"})
	if exit != ExitAPI {
		t.Errorf("expected exit %d, got %d", ExitAPI, exit)
	}
}

func TestDocs_JSONFormat(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/context" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "json" {
			t.Errorf("expected type=json, got %s", r.URL.Query().Get("type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"codeSnippets":[{"codeTitle":"x","codeLanguage":"go","codeList":[]}]}`))
	})

	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"--json", "docs", "/a/b"}); exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout.String())
	}
	// V2 normalizes IDs with leading slash.
	if got["libraryId"] != "/a/b" {
		t.Errorf("unexpected libraryId: %+v", got)
	}
}

// ---- ask -------------------------------------------------------------------

func TestAsk_HappyPath(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/search"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[
				{"id":"/vercel/next.js","trustScore":10,"totalSnippets":50},
				{"id":"/foo/bar","trustScore":5}
			]}`))
		case r.URL.Path == "/api/v2/context":
			if r.URL.Query().Get("libraryId") == "" {
				t.Errorf("ask should set libraryId, got none")
			}
			if r.URL.Query().Get("query") == "" {
				t.Errorf("ask should set query (semantic topic), got none")
			}
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("# routing docs"))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	var stdout, stderr bytes.Buffer
	exit := run(&stdout, &stderr, []string{"ask", "how do I do middleware in next.js"})
	if exit != 0 {
		t.Fatalf("exit %d stderr=%s", exit, stderr.String())
	}
	if !strings.Contains(stderr.String(), "resolved: /vercel/next.js") {
		t.Errorf("expected resolution announcement on stderr; got %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "# routing docs") {
		t.Errorf("docs body missing from stdout: %s", stdout.String())
	}
}

func TestAsk_NoCandidates_ExitOne(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	})

	exit := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"ask", "anything"})
	if exit != ExitNoResults {
		t.Errorf("expected exit %d, got %d", ExitNoResults, exit)
	}
}

// ---- cache + version (no upstream) -----------------------------------------

func TestCachePath_PrintsDir(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"cache", "path"}); exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if !strings.Contains(stdout.String(), "c7search") {
		t.Errorf("expected c7search in output, got %q", stdout.String())
	}
}

func TestVersionCmd_PrintsBuild(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"version"}); exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if !strings.HasPrefix(stdout.String(), "c7search ") {
		t.Errorf("unexpected version output: %q", stdout.String())
	}
}
