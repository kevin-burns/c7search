package cli

import (
	"testing"
	"unicode/utf8"
)

// FuzzExtractTopic asserts that extractTopic never panics on arbitrary
// natural-language input and respects the documented cap (80 bytes).
//
// Run with `go test -fuzz=FuzzExtractTopic -fuzztime=30s ./internal/cli/...`.
func FuzzExtractTopic(f *testing.F) {
	for _, seed := range []string{
		"how do I do middleware in next.js",
		"What are React hooks?",
		"",
		"a an the of for",
		"???????",
		"日本語のドキュメント",
		"   \t\n  ",
		"a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := extractTopic(in)
		if len(out) > 80 {
			t.Fatalf("cap violated: in=%q out=%q (%d bytes)", in, out, len(out))
		}
		if utf8.ValidString(in) && !utf8.ValidString(out) {
			t.Fatalf("invalid UTF-8 produced: in=%q out=%q", in, out)
		}
	})
}

// FuzzNormalizeQuery asserts that normalizeQuery is idempotent — applying
// it twice produces the same result as applying it once. Since the
// function lowercases via strings.ToLower, byte length can grow on
// invalid UTF-8 input (replacement U+FFFD is 3 bytes), so we don't assert
// length monotonicity. Idempotence is the property the cache key relies on.
func FuzzNormalizeQuery(f *testing.F) {
	for _, seed := range []string{
		"next.js",
		"next  js",
		"  NEXT\tJS\n",
		"",
		"\x00\x01",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		once := normalizeQuery(in)
		twice := normalizeQuery(once)
		if once != twice {
			t.Fatalf("not idempotent: in=%q once=%q twice=%q", in, once, twice)
		}
	})
}
