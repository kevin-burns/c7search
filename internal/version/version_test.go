package version

import (
	"strings"
	"testing"
)

func TestString_ContainsAllFields(t *testing.T) {
	s := String()
	for _, want := range []string{Version, Commit, BuildDate} {
		if !strings.Contains(s, want) {
			t.Errorf("String()=%q missing %q", s, want)
		}
	}
}

func TestUserAgent(t *testing.T) {
	ua := UserAgent()
	if !strings.HasPrefix(ua, "c7search/") {
		t.Errorf("UserAgent()=%q does not start with c7search/", ua)
	}
	if !strings.Contains(ua, Version) {
		t.Errorf("UserAgent()=%q missing version %q", ua, Version)
	}
}
