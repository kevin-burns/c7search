package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"testing"
)

// TestContract_ResolveJSON pins the public --json shape for `resolve`.
// Any change to these field names is a BREAKING change for downstream
// scripts piping through jq. If you must change, bump the major version.
func TestContract_ResolveJSON(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{
			"id":"/vercel/next.js",
			"title":"Next.js",
			"description":"d",
			"trustScore":10,
			"totalSnippets":42,
			"stars":1000
		}]}`))
	})

	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"--json", "resolve", "x"}); exit != 0 {
		t.Fatalf("exit=%d", exit)
	}

	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}

	wantFields := []string{"id", "title", "description", "trustScore", "totalSnippets", "stars"}
	for _, want := range wantFields {
		if _, ok := got[0][want]; !ok {
			gotFields := keys(got[0])
			sort.Strings(gotFields)
			t.Errorf("missing required field %q in resolve --json output (got fields: %v)",
				want, gotFields)
		}
	}
}

// TestContract_DocsJSON_Snippets pins the public --json shape for
// `docs` when the upstream returns structured snippets. The v2 endpoint
// returns libraryId with a leading slash; downstream consumers should
// expect either form.
func TestContract_DocsJSON_Snippets(t *testing.T) {
	f := newFakeAPI(t)
	f.handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"snippets":[{
			"codeTitle":"hello",
			"codeLanguage":"go",
			"codeList":[{"language":"go","code":"package main"}]
		}]}`))
	})

	var stdout bytes.Buffer
	if exit := run(&stdout, &bytes.Buffer{}, []string{"--json", "docs", "/a/b"}); exit != 0 {
		t.Fatalf("exit=%d", exit)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	for _, want := range []string{"libraryId", "format", "snippets"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing required top-level field %q (got: %v)", want, keys(got))
		}
	}
	snips, ok := got["snippets"].([]any)
	if !ok || len(snips) != 1 {
		t.Fatalf("snippets shape wrong: %+v", got["snippets"])
	}
	snip := snips[0].(map[string]any)
	for _, want := range []string{"codeTitle", "codeLanguage", "codeList"} {
		if _, ok := snip[want]; !ok {
			t.Errorf("missing snippet field %q (got: %v)", want, keys(snip))
		}
	}
}

// TestContract_ExitCodes pins the documented exit-code mapping. A change
// here breaks shell scripts that switch on $?.
func TestContract_ExitCodes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		code int
		want int
	}{
		{"OK", ExitOK, 0},
		{"NoResults", ExitNoResults, 1},
		{"API", ExitAPI, 2},
		{"Auth", ExitAuth, 3},
		{"Usage", ExitUsage, 4},
	}
	for _, c := range cases {
		if c.code != c.want {
			t.Errorf("exit %s = %d, want %d (BREAKING for shell scripts)", c.name, c.code, c.want)
		}
	}
}

// TestContract_RootFlags pins the persistent flag names. Renaming any of
// these is a breaking change.
func TestContract_RootFlags(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	for _, want := range []string{"api-key", "json", "format", "no-cache", "debug"} {
		if root.PersistentFlags().Lookup(want) == nil {
			t.Errorf("persistent flag %q removed (BREAKING)", want)
		}
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
