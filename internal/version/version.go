// Package version exposes build-time metadata stamped into the binary via
// -ldflags -X. Read by the `c7search version` command and by the HTTP client
// when constructing the User-Agent header. Defaults are chosen so that
// `go run` and `go install` produce something interpretable even without the
// release ldflags.
package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// String returns a human-readable build identifier.
func String() string {
	return fmt.Sprintf("c7search %s (%s, %s)", Version, Commit, BuildDate)
}

// UserAgent returns the value sent in the User-Agent header on every API
// request. Lets Context7 ops attribute traffic and contact us if needed.
func UserAgent() string {
	return fmt.Sprintf("c7search/%s (+https://github.com/kevin-burns/c7search)", Version)
}
