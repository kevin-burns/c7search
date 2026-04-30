package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/kevin-burns/c7search/internal/client"
)

// apiKeyScope returns a short opaque identifier for an API key suitable for
// namespacing cache entries. Empty key returns "anon"; otherwise the first
// 8 hex chars of sha256(key). Keeps anonymous and authenticated responses
// in separate cache slots without ever writing the key to disk.
func apiKeyScope(key string) string {
	if key == "" {
		return "anon"
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:8]
}

// Exit codes used by Execute().
const (
	ExitOK        = 0
	ExitNoResults = 1
	ExitAPI       = 2
	ExitAuth      = 3
	ExitUsage     = 4
)

// errNoResults is returned by commands that ran successfully but found
// nothing to print. Lets the user script around the difference between
// "search worked, returned empty" and "search exploded".
var errNoResults = errors.New("no results")

// errSilent wraps another error to suppress run()'s default "error: ..."
// stderr line. Use it when a command has already printed a user-friendly
// explanation (e.g. `auth status`) and only needs the exit code mapped.
var errSilent = errors.New("silent")

// silentWrap returns an error that classify() unwraps to the wrapped
// sentinel for exit-code mapping, while errors.Is(err, errSilent) is true
// so run() suppresses the duplicate stderr.
func silentWrap(err error) error {
	if err == nil {
		return nil
	}
	return silentErr{err: err}
}

type silentErr struct{ err error }

func (s silentErr) Error() string { return s.err.Error() }
func (s silentErr) Unwrap() error { return s.err }
func (s silentErr) Is(target error) bool {
	return target == errSilent
}

func classify(err error) int {
	switch {
	case errors.Is(err, errNoResults):
		return ExitNoResults
	case errors.Is(err, client.ErrUnauthorized):
		return ExitAuth
	case errors.Is(err, client.ErrBadRequest):
		return ExitUsage
	case errors.Is(err, client.ErrNotFound),
		errors.Is(err, client.ErrRateLimited),
		errors.Is(err, client.ErrServer):
		return ExitAPI
	default:
		return ExitAPI
	}
}
