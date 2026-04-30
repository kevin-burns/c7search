package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Search resolves a free-form query to candidate libraries. The server has
// already sorted by relevance; we preserve that order. Returns ([]Library,
// nil) on success — empty slice is valid (legitimate "no matches" outcome).
func (c *Client) Search(ctx context.Context, query string) ([]Library, error) {
	q := url.Values{}
	q.Set("query", query)
	_, body, err := c.do(ctx, "GET", "/api/v1/search", q, "application/json")
	if err != nil {
		return nil, err
	}
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return sr.Results, nil
}

// SearchV2 calls the v2 search endpoint, which separates the library
// name from the relevance query. The skill at
// github.com/intellectronica/agent-skills/skills/context7 documents this
// shape: GET /api/v2/libs/search?libraryName=X&query=Y. Use this when the
// caller knows the library name (tighter ranking); use Search for
// free-form lookups.
func (c *Client) SearchV2(ctx context.Context, libraryName, query string) ([]Library, error) {
	q := url.Values{}
	q.Set("libraryName", libraryName)
	q.Set("query", query)
	_, body, err := c.do(ctx, "GET", "/api/v2/libs/search", q, "application/json")
	if err != nil {
		return nil, err
	}
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("decode search v2 response: %w", err)
	}
	return sr.Results, nil
}

// Verify is a cheap probe used by `c7search auth status`. Returns nil if the
// API answered (regardless of result count); returns an error wrapping
// ErrUnauthorized / ErrRateLimited / etc on failure.
func (c *Client) Verify(ctx context.Context) error {
	_, err := c.Search(ctx, "test")
	return err
}
