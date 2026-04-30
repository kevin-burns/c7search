package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DocsV2 fetches documentation via the v2 unified context endpoint:
// GET /api/v2/context?libraryId=X&query=Y&type=txt|json. The libraryId
// goes in the querystring (not the URL path) and `Topic` becomes the
// semantic `query` parameter. This matches the public skill at
// github.com/intellectronica/agent-skills/skills/context7 and the path
// the upstream MCP server speaks. Prefer this over Docs (v1) for new
// callers.
//
// Library IDs are passed through as-is (leading slash preserved if
// present). The endpoint accepts both "/vercel/next.js" and "vercel/next.js".
func (c *Client) DocsV2(ctx context.Context, id string, opts DocsOptions) (Doc, error) {
	libraryID := strings.TrimSpace(id)
	if libraryID == "" {
		return Doc{}, fmt.Errorf("empty library id: %w", ErrBadRequest)
	}
	if !strings.HasPrefix(libraryID, "/") {
		libraryID = "/" + libraryID
	}
	format := normalizeFormat(opts.Format)

	q := url.Values{}
	q.Set("libraryId", libraryID)
	q.Set("type", format)
	if opts.Topic != "" {
		q.Set("query", opts.Topic)
	}
	if opts.Tokens > 0 {
		q.Set("tokens", strconv.Itoa(opts.Tokens))
	}

	accept := "text/plain"
	if format == "json" {
		accept = "application/json"
	}

	_, body, err := c.do(ctx, "GET", "/api/v2/context", q, accept)
	if err != nil {
		return Doc{}, err
	}

	doc := Doc{LibraryID: libraryID, Format: format}
	if format == "json" {
		var dr docsJSONResponse
		if err := json.Unmarshal(body, &dr); err != nil {
			return Doc{}, fmt.Errorf("decode docs v2 response: %w", err)
		}
		doc.Snippets = dr.Snippets
	} else {
		doc.Body = string(body)
	}
	return doc, nil
}

// Docs fetches documentation for a library. The format value is normalized:
// "" or "text"/"txt"/"md" → "txt"; "json" → "json".
//
// Deprecated: prefer DocsV2 for new callers. This v1 path is kept for
// backwards compatibility with existing tests and the v1-grandfathered URL.
func (c *Client) Docs(ctx context.Context, id string, opts DocsOptions) (Doc, error) {
	id = NormalizeLibraryID(id)
	if id == "" {
		return Doc{}, fmt.Errorf("empty library id: %w", ErrBadRequest)
	}

	format := normalizeFormat(opts.Format)

	q := url.Values{}
	q.Set("type", format)
	if opts.Topic != "" {
		q.Set("topic", opts.Topic)
	}
	if opts.Tokens > 0 {
		q.Set("tokens", strconv.Itoa(opts.Tokens))
	}

	accept := "text/plain"
	if format == "json" {
		accept = "application/json"
	}

	_, body, err := c.do(ctx, "GET", "/api/v1/"+strings.TrimPrefix(id, "/"), q, accept)
	if err != nil {
		return Doc{}, err
	}

	doc := Doc{LibraryID: id, Format: format}
	if format == "json" {
		var dr docsJSONResponse
		if err := json.Unmarshal(body, &dr); err != nil {
			return Doc{}, fmt.Errorf("decode docs response: %w", err)
		}
		doc.Snippets = dr.Snippets
	} else {
		doc.Body = string(body)
	}
	return doc, nil
}

// NormalizeLibraryID accepts "/owner/repo", "owner/repo", or " owner/repo "
// and returns the canonical "owner/repo" form: no surrounding whitespace,
// no leading slashes (any number — typo'd "//owner/repo" or even mixed
// slash/space prefixes get cleaned). The Context7 API is case-insensitive
// on lookup, so we preserve user casing.
func NormalizeLibraryID(id string) string {
	// Iterate strip-whitespace then strip-leading-slashes until the
	// result is stable. Uses unicode.IsSpace via TrimSpace so it covers
	// every whitespace codepoint (\f, \v, NBSP, etc.) rather than a
	// hand-rolled cutset.
	prev := ""
	for id != prev {
		prev = id
		id = strings.TrimSpace(id)
		id = strings.TrimLeft(id, "/")
	}
	return id
}

func normalizeFormat(f string) string {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "json":
		return "json"
	case "", "txt", "text", "md", "markdown":
		return "txt"
	default:
		return "txt"
	}
}
