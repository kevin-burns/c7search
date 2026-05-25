package client

import "time"

// Library is a single search hit from /api/v1/search. Field names mirror the
// JSON returned by Context7 verbatim — see .notes/api-probes.md for the
// captured shape.
type Library struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Branch         string    `json:"branch"`
	LastUpdateDate time.Time `json:"lastUpdateDate"`
	State          string    `json:"state"`
	TotalTokens    int       `json:"totalTokens"`
	TotalSnippets  int       `json:"totalSnippets"`
	Stars          int       `json:"stars"` // -1 = no data
	TrustScore     float64   `json:"trustScore"`
	BenchmarkScore float64   `json:"benchmarkScore"`
	Versions       []string  `json:"versions"`
	Score          float64   `json:"score"`
	VIP            bool      `json:"vip"`
	Verified       bool      `json:"verified"`
}

// searchResponse is the on-the-wire envelope; callers receive []Library.
type searchResponse struct {
	Results             []Library `json:"results"`
	SearchFilterApplied bool      `json:"searchFilterApplied"`
}

// DocsOptions tunes a get-library-docs request.
type DocsOptions struct {
	Topic  string
	Tokens int    // 0 → server default
	Format string // "txt" (default), "md" (alias of txt at the API level), or "json"
}

// Doc is the result of a docs fetch. For text/markdown formats Body holds
// the raw bytes; for JSON, Snippets is populated and Body is empty.
type Doc struct {
	LibraryID string
	Format    string // resolved format actually returned ("txt" or "json")
	Body      string
	Snippets  []DocSnippet
}

// DocSnippet matches the per-element shape returned by ?type=json.
type DocSnippet struct {
	CodeTitle       string         `json:"codeTitle"`
	CodeDescription string         `json:"codeDescription"`
	CodeLanguage    string         `json:"codeLanguage"`
	CodeTokens      int            `json:"codeTokens"`
	CodeID          string         `json:"codeId"`
	PageTitle       string         `json:"pageTitle"`
	CodeList        []DocCodeBlock `json:"codeList"`
	Relevance       float64        `json:"relevance"`
	Model           string         `json:"model"`
}

// DocCodeBlock is one (language, code) pair inside a snippet.
type DocCodeBlock struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// docsJSONResponse is the on-the-wire envelope from
// /api/v2/context?type=json. The array key is "codeSnippets" (the
// response also carries an "infoSnippets" array we don't surface) —
// confirmed against the live Context7 API. A leading "snippets" tag
// here silently decodes to an empty slice, which is what broke
// `docs --json` historically.
type docsJSONResponse struct {
	Snippets []DocSnippet `json:"codeSnippets"`
}
