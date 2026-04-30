package redact

import "testing"

func TestRedactURL(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"postgres with password", "postgres://user:hunter2@host:5432/db", "postgres://user:***@host:5432/db"},
		{"https with password", "https://user:s3cret@api.example.com/path", "https://user:***@api.example.com/path"},
		{"no userinfo", "https://api.example.com/path", "https://api.example.com/path"},
		{"user only, no password", "https://user@api.example.com", "https://user@api.example.com"},
		{"empty password", "https://user:@host", "https://user:@host"},
		{"not a url", "totally not a url", "totally not a url"},
		{"password with special chars", "https://u:p@ss:w0rd@host", "https://u:***@host"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RedactURL(c.in); got != c.want {
				t.Errorf("RedactURL(%q)\n  got:  %q\n  want: %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactBearer(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain", "Authorization: Bearer abc123xyz", "Authorization: Bearer ***"},
		{"uppercase", "BEARER ctx7sk-deadbeef", "Bearer ***"},
		{"in error message", "request failed: Bearer ctx7sk-foo expired", "request failed: Bearer *** expired"},
		{"no bearer", "request failed: 500 Internal", "request failed: 500 Internal"},
		{"two bearers", "old: Bearer aaa, new: Bearer bbb", "old: Bearer ***, new: Bearer ***"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RedactBearer(c.in); got != c.want {
				t.Errorf("RedactBearer(%q)\n  got:  %q\n  want: %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactAPIKey(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"long ctx7sk key", "ctx7sk-abcdefghijklmnopqrst", "ctx7sk...qrst"},
		{"exact 12 chars", "abcdef123456", "abcdef...3456"},
		{"too short", "ctx7sk", "***"},
		{"empty", "", "***"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RedactAPIKey(c.in); got != c.want {
				t.Errorf("RedactAPIKey(%q)\n  got:  %q\n  want: %q", c.in, got, c.want)
			}
		})
	}
}
