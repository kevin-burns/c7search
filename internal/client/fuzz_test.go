package client

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzNormalizeLibraryID asserts that NormalizeLibraryID is total
// (never panics on arbitrary input) and preserves the documented invariants:
//
//   - no leading slash
//   - no surrounding whitespace
//   - output length never exceeds input length
//   - if input is valid UTF-8, output is valid UTF-8
//
// Run with `go test -fuzz=FuzzNormalizeLibraryID -fuzztime=30s ./internal/client/...`.
func FuzzNormalizeLibraryID(f *testing.F) {
	for _, seed := range []string{
		"/vercel/next.js",
		"vercel/next.js",
		"  /vercel/next.js  ",
		"@types/node",
		"",
		"/",
		"////a//b",
		"\x00",
		"日本/library",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := NormalizeLibraryID(in)
		if strings.HasPrefix(out, "/") {
			t.Fatalf("leading slash leaked: in=%q out=%q", in, out)
		}
		if strings.TrimSpace(out) != out {
			t.Fatalf("surrounding whitespace leaked: in=%q out=%q", in, out)
		}
		if len(out) > len(in) {
			t.Fatalf("output longer than input: in=%q (%d) out=%q (%d)", in, len(in), out, len(out))
		}
		if utf8.ValidString(in) && !utf8.ValidString(out) {
			t.Fatalf("invalid UTF-8 produced from valid input: in=%q out=%q", in, out)
		}
	})
}
