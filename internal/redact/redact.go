// Package redact provides allocation-light helpers for stripping secrets
// out of strings before they hit logs, errors, or stdout.
//
// Three flavours are exposed because three flavours of secret can leak:
//   - URL passwords (postgres://user:pw@host) → RedactURL
//   - Bearer tokens in error/debug output    → RedactBearer
//   - API keys destined for human display    → RedactAPIKey
//
// All three are pure and side-effect-free; callable from hot error paths.
package redact

import (
	"net/url"
	"regexp"
	"strings"
)

// bearerRe matches "Bearer <token>" where the token is the RFC 6750
// "token68" charset (alphanumerics + a few punctuation marks). The
// keyword is case-insensitive so headers like "BEARER xyz" are caught
// too. The constrained character class avoids eating trailing
// punctuation in prose like "Bearer abc, ...".
var bearerRe = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-+/=~]+`)

// RedactURL replaces ":<password>@" in a URL with ":***@". Uses string-level
// surgery because url.UserPassword re-encodes the password on String(),
// which can leave the secret partially intact in encoded form.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	pw, hasPass := u.User.Password()
	if !hasPass || pw == "" {
		return raw
	}
	return strings.Replace(raw, ":"+pw+"@", ":***@", 1)
}

// RedactBearer replaces "Bearer <token>" occurrences with "Bearer ***".
// Apply before printing HTTP errors or --debug traces.
func RedactBearer(s string) string {
	return bearerRe.ReplaceAllString(s, "Bearer ***")
}

// RedactAPIKey returns a masked form of an API key suitable for display:
// the first 6 chars + "..." + last 4. Returns "***" for keys shorter than
// 12 chars so length itself doesn't leak useful information for short
// inputs (the empty string also returns "***" for callers that don't
// pre-check).
func RedactAPIKey(key string) string {
	if len(key) < 12 {
		return "***"
	}
	return key[:6] + "..." + key[len(key)-4:]
}
