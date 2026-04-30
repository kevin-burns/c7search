# Security Policy

## Supported versions

`c7search` is pre-1.0; only the latest minor release receives security
fixes. Earlier minors will not be patched — please upgrade.

| Version  | Supported |
|----------|-----------|
| Latest   | Yes       |
| Older    | No        |

## Reporting a vulnerability

Please report suspected vulnerabilities privately via GitHub Security
Advisories: <https://github.com/kevin-burns/c7search/security/advisories/new>.

If GitHub advisories are not available to you, email the maintainer
listed in `go.mod` / commit history. Please do **not** open public issues
for vulnerabilities.

What to include:

- A short description of the issue
- Reproduction steps or proof-of-concept
- Affected versions / commit SHA
- Your suggested severity, if any

Expected response: acknowledgement within 5 business days. We aim to
ship a fix within 30 days for high-severity issues.

## What's in scope

- Code execution, path traversal, or local privilege escalation in the
  CLI binary
- API key leakage via logs, error messages, or stored files
- Cache-tampering vectors
- Supply-chain weaknesses in our build/release pipeline

## What's out of scope

- Bugs in the upstream Context7 service (report those at
  <https://context7.com/contact>)
- Issues requiring a malicious local user with write access to the
  user's home directory or `$PATH`
- Vulnerabilities in third-party tools we depend on, unless we're
  pinning a vulnerable version (please open a regular PR or issue)

## Hardening posture

For users evaluating `c7search` for enterprise use:

- Single static binary, `CGO_ENABLED=0`, `-trimpath` for reproducible
  builds
- API keys masked on display (`auth status` shows prefix + last 4 only)
- Bearer tokens redacted in error messages and `--debug` traces
- Bounded `context.WithTimeout` on every external HTTP call
- Cache directory created with mode `0700`, files with mode `0600`
- `gitleaks` + `govulncheck` enforced in CI
- SHA256 checksums published with every release; cosign signing
  planned for v0.2
