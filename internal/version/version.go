// Package version exposes build-time metadata stamped into the binary.
//
// Three sources contribute, in priority order:
//
//  1. ldflags injection via -X (used by goreleaser and `make build`).
//     Stamps the package-level Version, Commit, BuildDate variables.
//  2. runtime/debug build info (populated by Go for `go install ...@vX.Y.Z`
//     and any `go build` from a git checkout).
//  3. Default placeholders ("dev", "none", "unknown") for `go test` and
//     edge cases where neither of the above is available.
//
// The composite value is exposed via String() and UserAgent(). Callers
// also see the "source" annotation — release / go install / local build —
// so it's obvious where any given binary came from.
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
)

// Stamped via -ldflags "-X .../version.Version=$(VERSION)" etc. by
// goreleaser and `make build`. Defaults are placeholders that the
// resolver will try to overwrite from runtime/debug build info.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// canonicalURL is the upstream repo. Stamped into String() and
// UserAgent() so users + ops can attribute traffic and find the project.
const canonicalURL = "https://github.com/kevin-burns/c7search"

type info struct {
	version string
	commit  string
	date    string
	source  string // "release" | "go install" | "local build" | "dev"
}

var (
	resolveOnce sync.Once
	resolved    info
)

// resolve returns the cached, fully-attributed version info. First call
// reads runtime/debug.BuildInfo; subsequent calls return the cached value.
func resolve() info {
	resolveOnce.Do(func() {
		bi, _ := debug.ReadBuildInfo()
		resolved = resolveFromBuildInfo(bi, Version, Commit, BuildDate)
	})
	return resolved
}

// resolveFromBuildInfo is the testable form of resolve. It composes the
// three sources without touching package-level state, so a test can
// drive every branch by constructing a synthetic BuildInfo.
func resolveFromBuildInfo(bi *debug.BuildInfo, ldVersion, ldCommit, ldDate string) info {
	out := info{
		version: ldVersion,
		commit:  ldCommit,
		date:    ldDate,
		source:  "dev",
	}

	// ldflags-stamped builds (goreleaser, `make build`) win outright.
	if ldVersion != "dev" {
		out.source = "release"
		return out
	}

	if bi == nil {
		return out
	}

	// `go install ...@vX.Y.Z` populates Main.Version with that tag.
	// `go build` / `go run` from a working tree populates it with "(devel)".
	switch bi.Main.Version {
	case "", "dev":
		// Leave defaults.
	case "(devel)":
		out.version = "(devel)"
		out.source = "local build"
	default:
		out.version = bi.Main.Version
		out.source = "go install"
	}

	// VCS settings are only present when Go could read .git. Empty for
	// `go install` of a binary cached without VCS metadata, which is fine.
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value != "" {
				out.commit = shortSHA(s.Value)
			}
		case "vcs.time":
			if s.Value != "" {
				out.date = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" && out.commit != "" && out.commit != "none" {
				out.commit += "+dirty"
			}
		}
	}

	return out
}

// shortSHA truncates to the conventional 7-char prefix. Anything shorter
// is returned unchanged.
func shortSHA(s string) string {
	if len(s) <= 7 {
		return s
	}
	return s[:7]
}

// String returns the human-readable version line printed by
// `c7search version` / `c7search --version`. Single line so scripts can
// parse the first word without ceremony; the source annotation and
// canonical URL appear at the end.
func String() string {
	i := resolve()
	return fmt.Sprintf("c7search %s (%s, %s, %s) %s",
		i.version, i.commit, i.date, i.source, canonicalURL)
}

// UserAgent returns the value sent in the User-Agent header. Lets
// Context7 ops attribute traffic and contact us if needed.
func UserAgent() string {
	v := resolve().version
	// Strip a leading "v" so the UA matches the "name/<semver>" RFC 7231
	// convention ("c7search/0.1.0" rather than "c7search/v0.1.0").
	v = strings.TrimPrefix(v, "v")
	return fmt.Sprintf("c7search/%s (+%s)", v, canonicalURL)
}
