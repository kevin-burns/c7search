package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

// resolve() prefers ldflags-set values; only falls back to BuildInfo when
// the package-level Version is the literal default ("dev").
func TestResolve_PrefersLdflags(t *testing.T) {
	bi := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/kevin-burns/c7search", Version: "v9.9.9"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
		},
	}
	got := resolveFromBuildInfo(bi, "v1.0.0", "deadbeef", "2026-01-01")
	if got.version != "v1.0.0" {
		t.Errorf("version: got %q want v1.0.0 (ldflags should win)", got.version)
	}
	if got.commit != "deadbeef" {
		t.Errorf("commit: got %q want deadbeef", got.commit)
	}
	if got.source != "release" {
		t.Errorf("source: got %q want release", got.source)
	}
}

// When ldflags are absent (Version=="dev") and Go embedded a real module
// version (typical of `go install ...@vX.Y.Z`), use that.
func TestResolve_UsesGoInstallVersion(t *testing.T) {
	bi := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/kevin-burns/c7search", Version: "v0.1.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789012345678901234567890123"},
			{Key: "vcs.time", Value: "2026-04-30T11:18:00Z"},
		},
	}
	got := resolveFromBuildInfo(bi, "dev", "none", "unknown")
	if got.version != "v0.1.0" {
		t.Errorf("version: got %q want v0.1.0 (from build info)", got.version)
	}
	if got.commit != "abcdef0" { // 7-char short SHA
		t.Errorf("commit: got %q want abcdef0", got.commit)
	}
	if got.date != "2026-04-30T11:18:00Z" {
		t.Errorf("date: got %q want 2026-04-30T11:18:00Z", got.date)
	}
	if got.source != "go install" {
		t.Errorf("source: got %q want go install", got.source)
	}
}

// `go run .` / `go build .` from inside the source tree gives Main.Version
// "(devel)" — treat that as a local dev build, not a tagged release.
func TestResolve_DevelMarkedAsLocalBuild(t *testing.T) {
	bi := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/kevin-burns/c7search", Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "feedface00112233"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	got := resolveFromBuildInfo(bi, "dev", "none", "unknown")
	if got.version != "(devel)" {
		t.Errorf("version: got %q want (devel)", got.version)
	}
	if !strings.Contains(got.commit, "+dirty") {
		t.Errorf("commit should be marked dirty when vcs.modified=true; got %q", got.commit)
	}
	if got.source != "local build" {
		t.Errorf("source: got %q want local build", got.source)
	}
}

// `go test` invocations have nil/sparse BuildInfo. resolve must not panic.
func TestResolve_HandlesNilBuildInfo(t *testing.T) {
	got := resolveFromBuildInfo(nil, "dev", "none", "unknown")
	if got.version != "dev" {
		t.Errorf("version: got %q want dev", got.version)
	}
}

// String() output is the user-facing version line. It must:
//   - start with "c7search " for `c7search version`-style scripts
//   - contain the version, commit, and date
//   - include a source attribution and the canonical repo URL
func TestString_ContainsExpectedFields(t *testing.T) {
	s := String()
	if !strings.HasPrefix(s, "c7search ") {
		t.Errorf("String() must start with 'c7search '; got %q", s)
	}
	if !strings.Contains(s, "github.com/kevin-burns/c7search") {
		t.Errorf("String() must include the canonical repo URL; got %q", s)
	}
}

func TestUserAgent(t *testing.T) {
	ua := UserAgent()
	if !strings.HasPrefix(ua, "c7search/") {
		t.Errorf("UserAgent() must start with c7search/; got %q", ua)
	}
	if !strings.Contains(ua, "+https://github.com/kevin-burns/c7search") {
		t.Errorf("UserAgent() must include the contact URL; got %q", ua)
	}
}
