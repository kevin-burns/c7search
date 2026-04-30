package client

import "errors"

// Sentinel errors. Wrap underlying details with fmt.Errorf("...: %w", ErrFoo)
// so callers can errors.Is on them.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrRateLimited  = errors.New("rate limited")
	ErrBadRequest   = errors.New("bad request")
	ErrServer       = errors.New("server error")
)
