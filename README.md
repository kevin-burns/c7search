# c7search

An **independent third-party Go CLI client** for the public
[Context7](https://context7.com) documentation HTTP API. It exposes the same
two operations as Upstash's official MCP server
([upstash/context7](https://github.com/upstash/context7)) —
`resolve-library-id` and `get-library-docs` — but as a single static binary
rather than an MCP transport.

Built for environments where adding new MCP servers requires approvals but
running a signed CLI does not. **This is not a replacement for, nor a fork
of, Upstash's MCP server**; it's an alternative transport for the same
public API. If your client supports MCP, use the upstream
[`@upstash/context7-mcp`](https://github.com/upstash/context7) directly.

## Why a CLI

- Locked-down enterprise: MCP server allow-lists block extensions; a CLI
  binary is fine.
- Mixed OS team: Windows colleagues get a `.exe`; macOS and Linux get a
  static binary. No Python, Node, or container runtime required.
- Pipe-friendly: status to stderr, payload to stdout, ANSI only on a TTY.
  Plays nicely with `glow`, `jq`, `less`.

## Install

### TL;DR — pick one

| You have… | Run | Quarantine? |
|---|---|---|
| Go ≥ 1.26.2 | `go install github.com/kevin-burns/c7search@latest` | **No** — recommended for macOS |
| `brew` (when tap is published) | `brew install kevin-burns/tap/c7search` | No — Homebrew clears it |
| Just a release `.tar.gz` | Extract, move onto `$PATH`, then on macOS run `xattr -d` + `codesign -` (see below) | Yes — needs the manual step |

The `go install` path is the simplest on macOS because Go-built
binaries are produced locally and never touch the
`com.apple.quarantine` xattr — Gatekeeper has nothing to flag and you
skip the `xattr` / `codesign` dance entirely.

### Where to install

`c7search` is a single static binary; pick whichever directory in your
`$PATH` you prefer:

| Location | Scope | sudo? | When to choose |
|---|---|---|---|
| `~/.local/bin` | user-only | no | XDG-style; lowest friction. Add to `PATH` if not already there. |
| `~/bin` | user-only | no | Classic per-user path. macOS adds it to `PATH` automatically when present. |
| `/usr/local/bin` | system-wide | **yes** | Shared with other CLI tools; survives user-profile resets. |
| `/opt/homebrew/bin` (Apple Silicon) `/usr/local/bin` (Intel) | system-wide | depends | If you want to drop alongside Homebrew tools. |

After install, confirm:

```bash
which c7search        # → /Users/you/.local/bin/c7search (or wherever)
c7search version      # → c7search v0.1.0 (...)
```

If `which` finds nothing, the directory isn't on `$PATH`. Add to
`~/.zshrc` (zsh) or `~/.bashrc` (bash):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### From source (recommended on macOS — no quarantine)

Requires Go ≥ 1.26.2.

```bash
go install github.com/kevin-burns/c7search@latest
# binary lands in $(go env GOBIN) or $(go env GOPATH)/bin
```

Add `$(go env GOPATH)/bin` to your `PATH` if it isn't already:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
c7search version
```

**No `xattr` / `codesign` step is needed.** The binary is built on
your machine, so it never receives the `com.apple.quarantine` extended
attribute that browsers and `curl` set on downloaded files.

#### Troubleshooting: `c7search: command not found` after `go install`

The most common cause on macOS: the install succeeded, but
`$GOPATH/bin` isn't on your `$PATH`. The binary is sitting in
`~/go/bin/c7search` and your shell can't see it. Three quick checks:

```bash
# 1. Where does Go put binaries on your machine?
go env GOBIN GOPATH
# If GOBIN is set, the binary is in $GOBIN.
# If GOBIN is empty, it's in $(go env GOPATH)/bin (typically ~/go/bin).

# 2. Does the binary actually exist?
ls -l "$(go env GOPATH)/bin/c7search"
# Expected: -rwxr-xr-x ... ~/go/bin/c7search

# 3. Is that directory on your PATH?
echo "$PATH" | tr ':' '\n' | grep -E '(/go/bin|GOBIN)'
# If empty, that's your problem — fix below.
```

If the binary exists but isn't on `$PATH`:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
which c7search        # should print ~/go/bin/c7search
c7search version
```

If the binary doesn't exist, the install failed silently. Re-run with
`-v`:

```bash
go install -v github.com/kevin-burns/c7search@v0.1.0 2>&1 | tail -20
```

Common failure modes:

- **Corporate proxy** blocking `proxy.golang.org` →
  `GOPROXY=direct go install github.com/kevin-burns/c7search@v0.1.0`
  (slower, bypasses the module proxy).
- **`~/go` owned by root** from a previous `sudo go install` →
  `sudo chown -R "$USER" ~/go`.

### From a release binary

Use this path if you don't have Go installed, or you want exactly the
artifact that shipped with a tagged release.

Download the platform archive from the
[Releases page](https://github.com/kevin-burns/c7search/releases) and verify
the SHA256:

```bash
shasum -a 256 -c checksums.txt
tar -xzf c7search-darwin-arm64.tar.gz       # adjust for your platform
mv c7search ~/.local/bin/                    # or wherever (see table above)
chmod +x ~/.local/bin/c7search
```

#### macOS: clear the Gatekeeper quarantine

Browsers and `curl` set the `com.apple.quarantine` extended attribute on
downloaded files. If you try to run the binary without clearing it,
macOS shows: *"c7search" cannot be opened because the developer cannot
be verified.*

**Recommended (no sudo, user-local install):**

```bash
# 1. Strip the quarantine xattr
xattr -d com.apple.quarantine ~/.local/bin/c7search

# 2. Ad-hoc sign so future Gatekeeper checks pass
codesign --force --sign - ~/.local/bin/c7search

# 3. Verify
c7search version
```

**System-wide install (`/usr/local/bin`, requires sudo):**

```bash
sudo xattr -d com.apple.quarantine /usr/local/bin/c7search
sudo codesign --force --sign - /usr/local/bin/c7search
c7search version
```

If `xattr -d` reports *No such xattr*, the file wasn't quarantined —
skip to the `codesign` step (or run `c7search version` directly; it may
already work).

To inspect what extended attributes are present before/after:

```bash
xattr -l ~/.local/bin/c7search
# expected after the steps above: (empty output)
```

> **Why this works:** stripping `com.apple.quarantine` opts you out of
> the first-launch Gatekeeper review for that file. The ad-hoc
> `codesign -` (the dash means "sign with no developer identity") gives
> the binary a local signature so subsequent integrity checks pass.
> This is appropriate for a binary you've SHA256-verified yourself.
> Once we publish cosign-signed releases (planned for v0.2), the
> manual `codesign` step won't be needed.

#### Windows: clear the SmartScreen mark

Edge / Chrome / curl mark downloaded `.exe`s with a Mark-of-the-Web
zone identifier. SmartScreen prompts the first time you run them.

**GUI:** Right-click `c7search.exe` → **Properties** → tick **Unblock**
→ **OK**.

**PowerShell (equivalent, scriptable):**

```powershell
Unblock-File -Path .\c7search.exe
.\c7search.exe version
```

**Linux:** no quarantine system; `chmod +x` and run.

## API key

`c7search` works anonymously against the public Context7 API — no key
required. Anonymous calls are rate-limited (~200 requests per window
per IP); set an API key to lift that ceiling.

### Get a key

Sign in at <https://context7.com> and copy your key from the dashboard.
Keys start with the prefix `ctx7sk-` (used by the redactor for safer
display).

### Provide it (three ways, in precedence order)

| Precedence | Method | When to use |
|---|---|---|
| 1. highest | `--api-key ctx7sk-...` flag | one-off override; CI jobs that pass a per-run key |
| 2. | `CONTEXT7_API_KEY` env var | normal day-to-day use |
| 3. lowest | none | anonymous tier — works for trivia and casual lookups |

```bash
# 1. per-invocation flag (overrides env)
c7search --api-key ctx7sk-XXXX auth status

# 2. current shell session only
export CONTEXT7_API_KEY=ctx7sk-XXXX
c7search auth status

# 3. persistent across sessions — add to ~/.zshrc (or ~/.bashrc)
echo 'export CONTEXT7_API_KEY=ctx7sk-XXXX' >> ~/.zshrc
source ~/.zshrc
```

If you keep your shell rc file in a public dotfiles repo, source the
key from a gitignored sidecar instead of inlining it:

```bash
# in ~/.zshrc (committed)
[ -f ~/.zshrc.local ] && source ~/.zshrc.local

# in ~/.zshrc.local (gitignored)
export CONTEXT7_API_KEY=ctx7sk-XXXX
```

### Verify it works

```bash
$ c7search auth status
key:    ctx7sk...XXXX
source: $CONTEXT7_API_KEY
status: ok
```

The `key` line shows only `prefix...last4`. If `status: invalid key`
appears, the call returned 401 — exit code is `3`.

### Confirm the key never leaks

`--debug` produces a structured request log on stderr. Bearer tokens
and URL-embedded credentials are scrubbed by a `redactHandler` before
the line is written:

```bash
c7search --debug resolve fastmcp 2>&1 | grep -i bearer  # → empty
```

If that command returns anything containing your key, **report it as a
security bug** — see [SECURITY.md](SECURITY.md). The redactor is
covered by `TestRedactHandler_ScrubsBearerAndCreds`; a regression
should fail CI before it ships.

### Cache and the API key

The on-disk cache namespaces every entry on `sha256(key)[:8]` (or
`anon` when no key is set). An agent that runs anonymously, then sets
a key mid-session and re-runs the same query, will hit the API again
rather than serve stale anonymous-tier results. The key itself is
never written to disk.

## Commands

```text
c7search resolve "next.js app router"
c7search resolve --library-name fastmcp "tool registration"     # v2 search
c7search docs /vercel/next.js --topic routing --tokens 5000
c7search ask "how do I do middleware in next.js"
c7search ask ... --json | jq .
c7search docs ... | glow -

c7search auth status        # show key source + connectivity
c7search cache path         # print the on-disk cache directory
c7search cache clear        # bust the cache
c7search version            # build version, commit, date
```

### Flags

| Flag             | Description                                                 |
|------------------|-------------------------------------------------------------|
| `--api-key`      | Override `$CONTEXT7_API_KEY` for this invocation            |
| `--format`       | `text` \| `md` \| `json` (per-command default)              |
| `--json`         | Shorthand for `--format json`                               |
| `--no-cache`     | Bypass the on-disk cache for this invocation                |
| `--debug`        | Verbose stderr logging (secrets redacted)                   |
| `--version`      | Print version and exit                                      |

`resolve`-only flags:

| Flag             | Description                                                 |
|------------------|-------------------------------------------------------------|
| `--limit N`      | Cap displayed results (`0` = unlimited; default `10`)       |
| `--library-name` | Route through `/api/v2/libs/search`; the positional argument becomes the relevance topic. Use this when you know the library by name. |

`docs` / `ask`-only flags:

| Flag             | Description                                                 |
|------------------|-------------------------------------------------------------|
| `--topic`        | Narrow to a topic (semantic query within the library)       |
| `--tokens N`     | Approximate token budget for the response (default `5000`)  |

### Exit codes

| Code | Meaning                            |
|------|------------------------------------|
| 0    | OK                                 |
| 1    | No results                         |
| 2    | API error (5xx, rate-limited, 404) |
| 3    | Auth error (401/403)               |
| 4    | Usage / bad request                |

## Using c7search from an LLM agent

`c7search` is built to be a clean substitute for the curl-and-jq recipe in
the Context7 [agent skill][skill]: same v2 endpoints (`/api/v2/libs/search`,
`/api/v2/context`), but with on-disk caching, secret redaction, retries
with bounded backoff, and predictable exit codes.

[skill]: https://github.com/intellectronica/agent-skills/blob/main/skills/context7/SKILL.md

A ready-to-install Claude skill that wraps this CLI ships at
[`skills/c7search/SKILL.md`](skills/c7search/SKILL.md). To install it
into a Claude Code session:

```bash
# user-scoped (available in every project)
mkdir -p ~/.claude/skills && cp -r skills/c7search ~/.claude/skills/

# or project-scoped (only the current repo)
mkdir -p .claude/skills && cp -r skills/c7search .claude/skills/
```

Then `/reload-plugins` (or restart the session). The skill auto-activates
when you ask Claude about a library, framework, or specific API.

### Quick recipes

**Two-step (verbose, gives you control):**

```bash
LIB=$(c7search resolve --library-name fastmcp "tool registration" --json --limit 1 \
        | jq -r '.[0].id')
c7search docs "$LIB" --topic "tool registration" --tokens 4000
```

**One-shot (only when the library name is unambiguous):**

```bash
c7search ask "how do I do app router middleware in next.js" --tokens 4000
```

`ask` runs `resolve` + `docs` and prints the resolution announcement
(`resolved: /vercel/next.js  (trust=9.5, snippets=2352)`) to **stderr**
so you can pipe stdout straight into the model context.

> **For agents: prefer the two-step recipe.** `ask` uses free-form v1
> search and picks the highest-trust hit, which can drift to a different
> library when your question phrasing matches another well-trusted entry
> (e.g. "register a tool in fastmcp" can resolve to a Go DI library
> because "register tool" matches it strongly). Use `resolve --library-name`
> to pin the library, then `docs` to fetch — that path uses the v2
> endpoints exclusively and has predictable resolution.

### Format guidance for agents

| Goal | Use | Why |
|---|---|---|
| Feed docs into a model context | default markdown (`docs`, `ask`) | Context7 returns a markdown-ish format pre-optimized for LLM token efficiency: `Source:` headers + fenced code blocks + `---` separators. JSON wrapping wastes ~10–15% tokens on braces. |
| Filter/sort snippets by language before ingesting | `--json` on `docs` | Exposes per-snippet `codeLanguage`, `codeTitle`, `codeList` so you can `jq` to e.g. only TypeScript blocks. |
| Pick a library ID from candidates | `--json` on `resolve` | Gives `id`, `trustScore`, `totalSnippets`, `stars` for ranking. |
| Show docs to a human at a terminal | default text/markdown, optionally piped through `glow` | ANSI/colour suppressed automatically when stdout isn't a TTY. |

### Stable contract for downstream scripts

These are pinned by tests; a change here is a breaking change.

- `resolve --json` → array of objects with at least `id`, `title`,
  `description`, `trustScore`, `totalSnippets`, `stars`.
- `docs --json` → object with `libraryId`, `format`, `snippets[]`. Each
  snippet has `codeTitle`, `codeLanguage`, `codeList[].language`,
  `codeList[].code`.
- Exit codes (table above) are part of the contract — branch on `$?`.

### Token budget guidance

- Trivia / one-API lookups: `--tokens 1500`
- Walkthrough of a feature: `--tokens 4000–6000` (default 5000)
- Whole-library reference: `--tokens 10000+` (anonymous tier may rate-limit)

### Authenticated vs anonymous

Anonymous works for trivia; expect rate-limiting on bursty workflows.
For a key, see [API key](#api-key) above. Quick check:

```bash
c7search auth status   # → "status: ok" or "invalid key" (exit 3)
```

## Cache

Responses are cached on disk under `os.UserCacheDir()/c7search/` —
typically:

- macOS: `~/Library/Caches/c7search/`
- Linux: `~/.cache/c7search/` (or `$XDG_CACHE_HOME/c7search/`)
- Windows: `%LOCALAPPDATA%\c7search\Cache\`

Search results live 6h; docs payloads live 24h. The cache directory is
created with mode `0700` and files with `0600`. Run `c7search cache clear`
to bust everything.

## Security

- API keys are never logged. Errors are passed through `redact.RedactURL`
  and `redact.RedactBearer` before being returned.
- `auth status` shows only `prefix...last4` — never the full key.
- Every external HTTP call runs under a bounded `context.WithTimeout`.
- Cache files: mode `0600`. Cache dir: mode `0700`. Defense in depth
  even though no secrets live there today.
- Build pipeline: `CGO_ENABLED=0`, `-trimpath`, SHA256 checksums.
  Cosign signing planned for v0.2.

See [SECURITY.md](SECURITY.md) for the disclosure policy.

## Development

```bash
make build       # build c7search in repo root
make test-race   # tests with race detector
make lint        # golangci-lint (gosec, errcheck, govet, staticcheck)
make secrets-scan
make vuln        # govulncheck
make cover       # 70% coverage gate
make fuzz        # opt-in fuzz harness for parsers
```

Quality gates also run via pre-commit hooks. The repo uses
[`prek`](https://github.com/j178/prek) when available (Rust, faster cold
start) and falls back to the canonical Python `pre-commit` otherwise:

```bash
# install one of the two runners
brew install prek                         # recommended
# or: pipx install pre-commit             # canonical fallback

make precommit-install                    # wires the git hook
make precommit-run                        # runs every hook over all files
```

Hooks configured in `.pre-commit-config.yaml`: `go fmt`, `go vet`,
`go-build`, `golangci-lint`, trailing-whitespace / end-of-file fixers,
large-file guard, private-key detector, and `gitleaks` for broad-spectrum
secret scanning.

## Acknowledgements

This CLI is a thin wrapper around the
[Context7 public API](https://context7.com). Original MCP server:
<https://github.com/upstash/context7>.

## License

MIT — see [LICENSE](LICENSE).
