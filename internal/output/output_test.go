package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevin-burns/c7search/internal/client"
)

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"":         FormatText,
		"text":     FormatText,
		"md":       FormatMarkdown,
		"markdown": FormatMarkdown,
		"JSON":     FormatJSON,
		"  json  ": FormatJSON,
		"weird":    FormatText,
	}
	for in, want := range cases {
		if got := ParseFormat(in); got != want {
			t.Errorf("ParseFormat(%q)=%v want %v", in, got, want)
		}
	}
}

func TestRenderLibraries_Text(t *testing.T) {
	libs := []client.Library{
		{ID: "/a/b", TrustScore: 9.5, TotalSnippets: 12, Stars: 100, Description: "first"},
		{ID: "/c/d", TrustScore: 7.0, TotalSnippets: 3, Stars: -1, Description: "second"},
	}
	var buf bytes.Buffer
	if err := RenderLibraries(&buf, libs, FormatText, 0); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"/a/b", "trust=9.5", "snippets=12", "★100", "first", "/c/d", "second"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "★-1") {
		t.Error("Stars=-1 sentinel leaked into output")
	}
}

func TestRenderLibraries_JSON(t *testing.T) {
	libs := []client.Library{{ID: "/a/b", TrustScore: 9.5}}
	var buf bytes.Buffer
	if err := RenderLibraries(&buf, libs, FormatJSON, 0); err != nil {
		t.Fatal(err)
	}
	var got []client.Library
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 1 || got[0].ID != "/a/b" {
		t.Errorf("decode mismatch: %+v", got)
	}
}

func TestRenderLibraries_Limit(t *testing.T) {
	libs := []client.Library{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	var buf bytes.Buffer
	if err := RenderLibraries(&buf, libs, FormatText, 2); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "c") {
		t.Errorf("limit=2 did not truncate; got %s", out)
	}
}

// T4: Stars=0 (legitimate) must render distinctly from Stars=-1 (sentinel
// for "no data"). Old code suppressed both, hiding zero-star repos.
func TestRenderLibraries_StarsZeroVsMinusOne(t *testing.T) {
	libs := []client.Library{
		{ID: "/zero/repo", TrustScore: 5, TotalSnippets: 1, Stars: 0, Description: "z"},
		{ID: "/nodata/repo", TrustScore: 5, TotalSnippets: 1, Stars: -1, Description: "n"},
	}
	var buf bytes.Buffer
	if err := RenderLibraries(&buf, libs, FormatText, 0); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "★0") {
		t.Errorf("Stars=0 should render as ★0; got %q", out)
	}
	if strings.Contains(out, "★-1") {
		t.Errorf("Stars=-1 sentinel must not render; got %q", out)
	}
	// The -1 row should still appear, just without the star marker.
	if !strings.Contains(out, "/nodata/repo") {
		t.Errorf("missing /nodata/repo line: %q", out)
	}
}

func TestRenderLibraries_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderLibraries(&buf, nil, FormatText, 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no results") {
		t.Errorf("expected 'no results' marker, got %q", buf.String())
	}
}

func TestRenderDoc_Text(t *testing.T) {
	doc := client.Doc{LibraryID: "vercel/next.js", Format: "txt", Body: "# Title\nbody"}
	var buf bytes.Buffer
	if err := RenderDoc(&buf, doc, FormatMarkdown); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "# Title\nbody" {
		t.Errorf("body not streamed verbatim: %q", buf.String())
	}
}

func TestRenderDoc_JSON_WithSnippets(t *testing.T) {
	doc := client.Doc{
		LibraryID: "vercel/next.js",
		Format:    "json",
		Snippets:  []client.DocSnippet{{CodeTitle: "hi"}},
	}
	var buf bytes.Buffer
	if err := RenderDoc(&buf, doc, FormatJSON); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, buf.String())
	}
	if got["libraryId"] != "vercel/next.js" {
		t.Errorf("missing libraryId; got %+v", got)
	}
	snips, ok := got["snippets"].([]any)
	if !ok || len(snips) != 1 {
		t.Errorf("snippets not preserved: %+v", got["snippets"])
	}
}

func TestRenderDoc_JSON_TextFallback(t *testing.T) {
	doc := client.Doc{LibraryID: "vercel/next.js", Format: "txt", Body: "raw text"}
	var buf bytes.Buffer
	if err := RenderDoc(&buf, doc, FormatJSON); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if got["body"] != "raw text" {
		t.Errorf("body not wrapped: %+v", got)
	}
}
