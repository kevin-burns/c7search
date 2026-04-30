---
name: c7search
description: Retrieve up-to-date documentation for software libraries, frameworks, components, and APIs via the Context7 service using the `c7search` CLI — an independent third-party Go client of Upstash's public Context7 HTTP API (https://github.com/upstash/context7). Use this skill when looking up docs for any programming library or framework, finding code examples for specific APIs or features, verifying correct usage of library functions, or obtaining current information about library APIs that may have changed since training cutoff. Prefer this skill over the community curl-based skill at github.com/intellectronica/agent-skills/skills/context7 when `c7search` is installed locally — both hit the same v2 endpoints, but `c7search` adds on-disk caching, automatic retries with bounded backoff, redacted error messages, and predictable exit codes for branching. This skill is NOT a replacement for the official Upstash MCP server (`@upstash/context7-mcp`) — use the MCP server when your client supports MCP; use this CLI when MCP isn't an option.
---

# c7search

## Overview

`c7search` is a single static binary that wraps the Context7 v2 API. It
exposes the same two operations as the upstream MCP server —
`resolve-library-id` and `get-library-docs` — but with on-disk caching,
secret redaction, retries, and predictable exit codes. Output goes to
stdout (markdown by default, JSON on demand); status messages go to
stderr.

## When to use this skill

- User asks "how do I X in <library>" or "show me <library>'s API for Y"
- User asks for code examples or current usage of a library
- User asks about a library/framework that may have changed since the
  training cutoff
- Verifying the right API surface before generating code that uses it

## Prerequisites

### Check the binary is installed and runnable

```bash
command -v c7search >/dev/null 2>&1 && c7search version || echo MISSING
```

If you see `MISSING`, fall back to the [curl-based `context7` skill](
https://github.com/intellectronica/agent-skills/blob/main/skills/context7/SKILL.md)
or install c7search using one of the paths below.

### Install — pick one

**Option A (recommended on macOS): `go install` — no quarantine**

```bash
go install github.com/kevin-burns/c7search@latest
# binary lands in $(go env GOPATH)/bin
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
c7search version
```

The binary is built locally, so the `com.apple.quarantine` extended
attribute is never set. **No `xattr` / `codesign` step needed.**

**Option B: release binary**

```bash
# Download from https://github.com/kevin-burns/c7search/releases
mkdir -p ~/.local/bin
mv c7search ~/.local/bin/
chmod +x ~/.local/bin/c7search
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

On macOS, the release binary carries the quarantine xattr — see the
next subsection.

### macOS: clear the Gatekeeper quarantine (release-binary install only)

Skip this section entirely if you used `go install` (Option A above).

Release binaries downloaded via a browser or `curl` carry the
`com.apple.quarantine` extended attribute. If `c7search version`
returns *"cannot be opened because the developer cannot be verified"*,
strip the attribute and ad-hoc sign:

```bash
xattr -d com.apple.quarantine ~/.local/bin/c7search
codesign --force --sign - ~/.local/bin/c7search
c7search version    # should now print version info
```

If you installed under `/usr/local/bin`, prefix both commands with
`sudo`.

Diagnostic — list extended attributes (empty output = clean):

```bash
xattr -l "$(command -v c7search)"
```

### Windows: clear the SmartScreen mark

```powershell
Unblock-File -Path .\c7search.exe
.\c7search.exe version
```

(GUI alternative: right-click → Properties → tick **Unblock** → OK.)

### Linux

No quarantine system. After `chmod +x`, the binary runs.

### API key (optional)

With `CONTEXT7_API_KEY=ctx7sk-...` exported, you get higher rate
limits; without one, anonymous tier is fine for trivia and casual
lookups. See the project README's *API key* section for setup details
including the `~/.zshrc.local` gitignored-sidecar pattern.

## Workflow

The recommended path is a two-step recipe: pin the library by name,
then fetch docs with a semantic topic. `c7search ask` exists as a
one-shot but can drift on ambiguous queries — prefer the two-step.

### Step 1: Resolve the library ID

Use `--library-name` (routes through the v2 search endpoint) when you
know the library name:

```bash
c7search resolve --library-name <name> "<topic>" --json --limit 1 \
  | jq -r '.[0].id'
```

Returns a string like `/prefecthq/fastmcp` or `/vercel/next.js`.

For free-form lookups where you don't yet know the library name, omit
`--library-name`:

```bash
c7search resolve "<query>" --json --limit 5 | jq '.[] | {id, trustScore, totalSnippets, stars}'
```

The free-form path uses the v1 grandfathered search endpoint, which
ranks by trust score and can return high-trust hits for unrelated
libraries. Inspect the top results before fetching docs.

### Step 2: Fetch documentation

Use the resolved ID; pass `--topic` as the semantic query within the
library:

```bash
c7search docs "<library-id>" --topic "<topic>" --tokens <budget>
```

Default output is markdown text optimized for LLM ingestion (the
upstream API's `type=txt`). The format embeds `Source:` URLs + fenced
code blocks + `---` separators that LLMs parse natively without
spending tokens on JSON envelope.

Pass `--json` only when programmatic filtering is needed before
ingestion (e.g. picking only Go snippets):

```bash
c7search docs "<library-id>" --topic "<topic>" --tokens 5000 --json \
  | jq '.snippets[] | select(.codeLanguage == "go")'
```

## Examples

### React `useState` documentation

```bash
LIB=$(c7search resolve --library-name react "hooks useState" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "useState" --tokens 3000
```

### Next.js app-router middleware

```bash
LIB=$(c7search resolve --library-name "next.js" "app router middleware" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "middleware" --tokens 4000
```

### FastMCP tool registration (Python)

```bash
LIB=$(c7search resolve --library-name fastmcp "tool registration" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "register a tool" --tokens 3000
```

### Filtering: only TypeScript snippets from a Next.js docs fetch

```bash
LIB=$(c7search resolve --library-name "next.js" "server actions" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "server actions" --tokens 6000 --json \
  | jq '.snippets[] | select(.codeLanguage | test("typescript|ts"))'
```

### Verifying API connectivity / key

```bash
c7search auth status     # prints key prefix + last4, then "status: ok" or 401
```

## Token budget guidance

| Goal | `--tokens` |
|---|---|
| One specific API call (signature, parameters) | `1500` |
| Walkthrough of a feature (default) | `4000–6000` (default `5000`) |
| Whole-library reference | `10000+` (anonymous tier may rate-limit) |

## Exit codes

Branch on `$?` to handle failures cleanly:

| Code | Meaning |
|---|---|
| `0` | OK |
| `1` | No results (search returned empty) |
| `2` | API error (5xx, 429, 404) |
| `3` | Auth error (401/403) |
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

- **Markdown for ingestion, JSON for filtering.** The default text
  output is what Context7 has token-optimized for LLMs. JSON adds ~10–15%
  overhead from braces/escapes — only use it when you need structured
  fields like `codeLanguage` for filtering.
- **Cache works.** Search results live 6 h on disk; docs payloads
  live 24 h. Repeated queries are ~10× faster. Use `--no-cache` only
  when you need fresh data (e.g. just published a release).
- **Two-step beats `ask`.** `c7search ask "..."` does resolve + docs in
  one shot but uses free-form search; on ambiguous phrasing it can
  resolve to a different library (e.g. "register a tool in fastmcp"
  can resolve to a Go DI library because "register tool" matches
  strongly there). Pin with `resolve --library-name`, then `docs`.
- **Library IDs accept either `/owner/repo` or `owner/repo`.** The CLI
  normalizes leading slashes, whitespace, and case.
- **`--debug` is safe.** Bearer tokens and URL credentials are scrubbed
  before any log line hits stderr. Verify with
  `c7search --debug resolve x 2>&1 | grep -i bearer` (should be empty).
- **Output discipline.** Status announcements (`resolved: /vercel/next.js
  (trust=9.5, snippets=2352)`) go to stderr; payload goes to stdout.
  Pipe stdout straight into the model context without filtering.

## Comparison with the curl-based `context7` skill

Both skills hit the same v2 endpoints. Differences:

| Concern | curl + jq | c7search |
|---|---|---|
| Caching | none | 6 h search / 24 h docs, on disk |
| Retry on 5xx/429 | none | exponential backoff + jitter, capped 16 s |
| Secret redaction in errors | none | bearer + URL creds scrubbed |
| Exit codes | curl/jq generic | semantic (1=no results, 3=auth, etc.) |
| Output discipline | mixed | stderr=status, stdout=payload |
| Install footprint | curl + jq | one ~8 MB static binary |

Choose the curl skill for ad-hoc one-liners or environments where you
can't install a binary; choose this skill for repeated lookups,
scripts, and agent workflows.
