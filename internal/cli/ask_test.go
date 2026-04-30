package cli

import (
	"strings"
	"testing"

	"github.com/kevin-burns/c7search/internal/client"
)

func TestExtractTopic(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"how do I do middleware in next.js", "middleware next.js"},
		{"What are React hooks?", "react hooks"},
		{"", ""},
		{"a an the of for", ""},
	}
	for _, c := range cases {
		got := extractTopic(c.in)
		if got != c.want {
			t.Errorf("extractTopic(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractTopic_Cap(t *testing.T) {
	long := strings.Repeat("token ", 50)
	got := extractTopic(long)
	if len(got) > 80 {
		t.Errorf("expected cap at 80 chars, got %d", len(got))
	}
}

func TestPickBest(t *testing.T) {
	libs := []client.Library{
		{ID: "/a", TrustScore: 7, TotalSnippets: 100},
		{ID: "/b", TrustScore: 9, TotalSnippets: 5},
		{ID: "/c", TrustScore: 9, TotalSnippets: 50}, // tie on trust, beats /b on snippets
	}
	got, err := pickBest(libs)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "/c" {
		t.Errorf("expected /c, got %s", got.ID)
	}
}

func TestPickBest_Empty(t *testing.T) {
	if _, err := pickBest(nil); err == nil {
		t.Error("expected error on empty libs")
	}
}

func TestPickBest_NoUsefulData(t *testing.T) {
	libs := []client.Library{{ID: "/x"}, {ID: "/y"}}
	if _, err := pickBest(libs); err == nil {
		t.Error("expected error when no library has trust or snippets")
	}
}
