package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := New(Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 0,
		APIKey:     "ctx7sk-test-deadbeef-key",
	})
	return c, srv
}

func TestSearch_Happy(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "next.js" {
			t.Errorf("unexpected query: %s", r.URL.Query().Get("query"))
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ctx7sk-test-deadbeef-key" {
			t.Errorf("missing/wrong bearer: %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "c7search/") {
			t.Errorf("missing user-agent: %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"id":"/vercel/next.js","title":"Next.js","trustScore":10,"totalSnippets":2201,"stars":131745},
			{"id":"/foo/bar","title":"Foo","trustScore":5}
		]}`))
	})

	libs, err := c.Search(context.Background(), "next.js")
	if err != nil {
		t.Fatalf("Search returned: %v", err)
	}
	if len(libs) != 2 {
		t.Fatalf("expected 2 results, got %d", len(libs))
	}
	if libs[0].ID != "/vercel/next.js" || libs[0].TrustScore != 10 {
		t.Errorf("first result mismatch: %+v", libs[0])
	}
}

// T12: v2 search exposes the libraryName/query split that the public skill
// at github.com/intellectronica/agent-skills/skills/context7 documents.
// Path: /api/v2/libs/search?libraryName=X&query=Y.
func TestSearchV2_Happy(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/libs/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("libraryName") != "react" {
			t.Errorf("libraryName: %s", r.URL.Query().Get("libraryName"))
		}
		if r.URL.Query().Get("query") != "hooks" {
			t.Errorf("query: %s", r.URL.Query().Get("query"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"/websites/react_dev_reference","title":"React","trustScore":9}]}`))
	})

	libs, err := c.SearchV2(context.Background(), "react", "hooks")
	if err != nil {
		t.Fatalf("SearchV2: %v", err)
	}
	if len(libs) != 1 || libs[0].ID != "/websites/react_dev_reference" {
		t.Errorf("unexpected: %+v", libs)
	}
}

func TestSearch_Unauthorized(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key","message":"Invalid API key"}`))
	})

	_, err := c.Search(context.Background(), "x")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestDocs_TextHappy(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vercel/next.js" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "txt" {
			t.Errorf("expected type=txt, got %s", r.URL.Query().Get("type"))
		}
		if r.URL.Query().Get("topic") != "routing" {
			t.Errorf("expected topic=routing, got %s", r.URL.Query().Get("topic"))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# Next.js\n## Routing\n..."))
	})

	doc, err := c.Docs(context.Background(), "/vercel/next.js", DocsOptions{Topic: "routing", Tokens: 500, Format: "txt"})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	if !strings.HasPrefix(doc.Body, "# Next.js") {
		t.Errorf("unexpected body: %q", doc.Body)
	}
	if doc.Format != "txt" {
		t.Errorf("expected format txt, got %s", doc.Format)
	}
}

func TestDocs_JSONHappy(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "json" {
			t.Errorf("expected type=json")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"codeSnippets":[{"codeTitle":"hello","codeLanguage":"go","codeList":[{"language":"go","code":"package main"}]}]}`))
	})

	doc, err := c.Docs(context.Background(), "vercel/next.js", DocsOptions{Format: "json"})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	if len(doc.Snippets) != 1 || doc.Snippets[0].CodeTitle != "hello" {
		t.Errorf("unexpected snippets: %+v", doc.Snippets)
	}
}

// T12 (cont.): DocsV2 calls /api/v2/context with the libraryId in the
// querystring (not the URL path) and exposes a semantic `query`
// parameter distinct from v1's `topic`. This matches the public skill
// at github.com/intellectronica/agent-skills/skills/context7.
func TestDocsV2_TextHappy(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/context" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("libraryId"); got != "/vercel/next.js" {
			t.Errorf("libraryId: %s", got)
		}
		if got := r.URL.Query().Get("query"); got != "app router" {
			t.Errorf("query: %s", got)
		}
		if got := r.URL.Query().Get("type"); got != "txt" {
			t.Errorf("type: %s", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# Next.js app router\n..."))
	})

	doc, err := c.DocsV2(context.Background(), "/vercel/next.js", DocsOptions{Topic: "app router", Format: "txt"})
	if err != nil {
		t.Fatalf("DocsV2: %v", err)
	}
	if !strings.HasPrefix(doc.Body, "# Next.js") {
		t.Errorf("body not streamed: %q", doc.Body)
	}
	if doc.LibraryID != "/vercel/next.js" {
		t.Errorf("libraryId not preserved: %q", doc.LibraryID)
	}
}

func TestDocsV2_JSONHappy(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "json" {
			t.Errorf("expected type=json")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"codeSnippets":[{"codeTitle":"hi","codeLanguage":"go","codeList":[]}]}`))
	})
	doc, err := c.DocsV2(context.Background(), "/a/b", DocsOptions{Format: "json"})
	if err != nil {
		t.Fatalf("DocsV2: %v", err)
	}
	if len(doc.Snippets) != 1 || doc.Snippets[0].CodeTitle != "hi" {
		t.Errorf("snippets: %+v", doc.Snippets)
	}
}

func TestDocs_NotFoundPlainText(t *testing.T) {
	// Real API returns plain-text 404; verify we still surface ErrNotFound.
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Library /does/not-exist not found."))
	})

	_, err := c.Docs(context.Background(), "/does/not-exist", DocsOptions{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message lost: %v", err)
	}
}

func TestDocs_RetriesOn500(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	c := New(Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 2,
	})
	doc, err := c.Docs(context.Background(), "vercel/next.js", DocsOptions{})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if doc.Body != "ok" || calls != 2 {
		t.Errorf("retry path wrong: body=%q calls=%d", doc.Body, calls)
	}
}

// T5: backoff must be monotonically non-decreasing up to a fixed ceiling
// and must never produce a negative duration regardless of attempt count.
// A misconfigured MaxRetries cannot be allowed to wedge the client in a
// tight retry loop (negative durations make time.After fire immediately).
func TestBackoff_ClampedAndNonNegative(t *testing.T) {
	const ceiling = 16 * time.Second
	for attempt := 1; attempt <= 64; attempt++ {
		d := backoff(attempt)
		if d < 0 {
			t.Fatalf("attempt=%d produced negative duration %v", attempt, d)
		}
		if d > ceiling+ceiling/4 { // ceiling + max jitter
			t.Fatalf("attempt=%d exceeded ceiling: %v", attempt, d)
		}
	}
	// Sanity: small attempts grow.
	if backoff(1) >= backoff(3) {
		t.Errorf("backoff should grow with attempt: 1=%v 3=%v", backoff(1), backoff(3))
	}
}

func TestNormalizeLibraryID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/vercel/next.js", "vercel/next.js"},
		{"vercel/next.js", "vercel/next.js"},
		{"  /vercel/next.js  ", "vercel/next.js"},
		{"@types/node", "@types/node"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizeLibraryID(c.in); got != c.want {
			t.Errorf("NormalizeLibraryID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
