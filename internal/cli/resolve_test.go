package cli

import (
	"strings"
	"testing"

	"github.com/kevin-burns/c7search/internal/cache"
)

// T6a: anon and keyed callers must hit distinct cache slots — same query
// can return different results based on rate-limit tier.
func TestResolveCacheKey_PerAPIKey(t *testing.T) {
	t.Parallel()
	const q = "next.js routing"
	anon := cache.Key("search", apiKeyScope(""), normalizeQuery(q))
	keyed := cache.Key("search", apiKeyScope("ctx7sk-real"), normalizeQuery(q))
	if anon == keyed {
		t.Fatalf("anon and keyed produced same cache key: %s", anon)
	}
	if !strings.HasPrefix(anon, "search_") {
		t.Errorf("expected search_ namespace prefix, got %s", anon)
	}
}

// T6b: whitespace-insensitive query normalization — "next  js" and
// "next js" share a slot (saves cache space, matches user intent).
func TestNormalizeQuery_CollapsesWhitespace(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"next js":          "next js",
		"next  js":         "next js",
		"  NEXT\tJS  ":     "next js",
		"\nNext\n  Js\t\t": "next js",
		"":                 "",
	}
	for in, want := range cases {
		if got := normalizeQuery(in); got != want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

// T6c: apiKeyScope must NOT echo any portion of the key on disk.
// (Cache filenames embed this; a leak would put key fragments in
// directory listings.)
func TestAPIKeyScope_DoesNotLeakKey(t *testing.T) {
	t.Parallel()
	const key = "ctx7sk-supersecret-1234567890"
	scope := apiKeyScope(key)
	if strings.Contains(scope, "ctx7sk") || strings.Contains(scope, "supersecret") {
		t.Errorf("scope leaked key material: %q", scope)
	}
	if scope == "" || scope == "anon" {
		t.Errorf("scope of non-empty key should be non-empty and != 'anon': %q", scope)
	}
}
