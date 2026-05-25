---
name: c7search
description: Retrieve up-to-date documentation and code examples for any software library, framework, or API via the Context7 service, using the `c7search` CLI. Use this skill whenever looking up how to use a library or framework, finding current code examples for a specific API or feature, verifying the correct signature or usage of a library function, or checking library behavior that may have changed since the training cutoff — even when the user doesn't name Context7 or c7search explicitly. `c7search` is a single Go binary wrapping Context7's public v2 API with on-disk caching, retries, and predictable exit codes.
---

# c7search

`c7search` wraps the Context7 v2 documentation API in a single static
binary. It exposes two operations — resolve a library name to an ID, then
fetch docs for that ID — with on-disk caching, secret redaction, retries,
and semantic exit codes. Payload goes to **stdout** (markdown by default,
JSON on demand); status messages go to **stderr**, so you can pipe stdout
straight into the model context without filtering.

## When to use

- "How do I X in `<library>`" / "show me `<library>`'s API for Y"
- Need current code examples or usage for a library or framework
- A library may have changed since the training cutoff
- Verifying the right API surface before generating code that uses it

## Check the binary is installed

```bash
command -v c7search >/dev/null 2>&1 && c7search version || echo MISSING
```

If `MISSING`: the quickest install (and the only one needing no extra
steps on macOS) is `go install github.com/kevin-burns/c7search@latest`,
then ensure `$(go env GOPATH)/bin` is on `$PATH`. For release binaries
and per-OS notes (macOS Gatekeeper, Windows SmartScreen), see the install
section of the README: <https://github.com/kevin-burns/c7search#install>.
If you can't install a binary, fall back to the curl-based `context7`
skill (same endpoints, no caching/retries):
<https://github.com/intellectronica/agent-skills/blob/main/skills/context7/SKILL.md>

An API key is optional (`CONTEXT7_API_KEY=ctx7sk-...` lifts rate limits);
the anonymous tier is fine for casual lookups. `c7search auth status`
prints the key source and connectivity.

## Workflow: resolve, then fetch

Prefer this two-step recipe. It uses the v2 endpoints exclusively and has
predictable resolution. (`c7search ask "..."` does both in one shot but
uses free-form v1 search and can drift to a different library on ambiguous
phrasing — e.g. "register a tool in fastmcp" can resolve to a Go DI
library because "register tool" matches it strongly there.)

**Step 1 — resolve the library ID.** Use `--library-name` when you know
the name; the positional argument becomes the relevance topic:

```bash
c7search resolve --library-name <name> "<topic>" --json --limit 1 | jq -r '.[0].id'
```

Returns an ID like `/vercel/next.js`. For free-form lookups where you
don't know the library name, omit `--library-name` and inspect the top
results before fetching — that path ranks by trust score and can surface
high-trust hits for unrelated libraries:

```bash
c7search resolve "<query>" --json --limit 5 | jq '.[] | {id, trustScore, totalSnippets, stars}'
```

**Step 2 — fetch docs.** Pass the resolved ID and a semantic `--topic`:

```bash
c7search docs "<library-id>" --topic "<topic>" --tokens <budget>
```

## Examples

**Two-step lookup (Next.js app-router middleware):**

```bash
LIB=$(c7search resolve --library-name "next.js" "app router middleware" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "middleware" --tokens 4000
```

**Filter by language before ingesting (only TypeScript snippets):**

```bash
LIB=$(c7search resolve --library-name "next.js" "server actions" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "server actions" --tokens 6000 --json \
  | jq '.snippets[] | select(.codeLanguage | test("typescript|ts"))'
```

## Output format

Default markdown is what Context7 token-optimizes for LLMs (`Source:`
headers + fenced code blocks + `---` separators). Feed it straight into
context. Use `--json` **only** when you need to filter or sort snippets by
a structured field (`codeLanguage`, `codeTitle`) before ingesting — it
adds ~10–15% token overhead from the JSON envelope.

## Token budget (`--tokens`)

| Goal | Budget |
|---|---|
| One specific API call (signature, parameters) | `1500` |
| Walkthrough of a feature (default) | `4000–6000` (default `5000`) |
| Whole-library reference | `10000+` (anonymous tier may rate-limit) |

## Exit codes — branch on `$?`

| Code | Meaning |
|---|---|
| `0` | OK |
| `1` | No results (search returned empty) |
| `2` | API error (5xx, 429, 404) |
| `3` | Auth error (401/403) — check `CONTEXT7_API_KEY` |
| `4` | Usage / bad request |

```bash
if ! out=$(c7search docs "$LIB" --topic "$TOPIC" 2>&1); then
  case $? in
    1) echo "no docs found";;
    2) echo "API problem; retry later";;
    3) echo "check CONTEXT7_API_KEY";;
    *) echo "unknown error: $out";;
  esac
fi
```

## Tips

- **Cache works.** Search results live 6 h on disk, docs payloads 24 h;
  repeated queries are ~10× faster. Use `--no-cache` only when you need
  fresh data (e.g. a release you just published).
- **Library IDs accept `/owner/repo` or `owner/repo`.** Leading slashes,
  whitespace, and case are normalized.
- **`--debug` is safe.** Bearer tokens and URL credentials are scrubbed
  before any log line reaches stderr.
