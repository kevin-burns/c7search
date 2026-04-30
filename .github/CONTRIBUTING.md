# Contributing to c7search

## Quick setup

```bash
git clone https://github.com/kevin-burns/c7search.git
cd c7search
go test -race ./...                    # should be green
make precommit-install                 # wires the prek (or pre-commit) hook
make build                             # produces ./c7search
```

Requires Go ≥ 1.26.2 (matches `go.mod`). The pre-commit runner prefers
[`prek`](https://github.com/j178/prek) when available and falls back to
the canonical Python `pre-commit` otherwise — see the README for install
instructions.

## How to land a change

`main` is protected. **Direct pushes by non-admins are rejected.** Even
admins should default to PRs; the bypass exists for emergencies, not
convenience.

```bash
git checkout -b feat/short-description-of-change
# ... edit, test ...
make precommit-run                     # runs every hook over all files
go test -race ./...
git commit -m "feat: ..."              # conventional commit prefix
git push -u origin feat/short-description-of-change
gh pr create --fill                    # title & body from the commit
```

The PR cannot merge until **all six required checks** pass:

| Check | Workflow | What it does |
|---|---|---|
| `test (ubuntu-latest)` | `test.yml` | `go test -race ./...` on Linux |
| `test (macos-latest)` | `test.yml` | same on macOS |
| `test (windows-latest)` | `test.yml` | same on Windows |
| `lint` | `test.yml` | golangci-lint v2.11.4 (gosec, errcheck, govet, staticcheck) |
| `gitleaks` | `security.yml` | broad-spectrum secret scan over full history |
| `govulncheck` | `security.yml` | Go vulnerability database, call-graph aware |

Once green, merge via squash:

```bash
gh pr merge --squash --delete-branch
```

## Commit message style

Conventional-commit prefix in the subject line: `feat:`, `fix:`,
`docs:`, `ci:`, `chore:`, `refactor:`, `test:`, `build:`. The `.goreleaser.yaml`
groups release notes by these prefixes; using them keeps the changelog
honest.

Subject under 72 characters. Body wrapped at 72 columns. Explain why,
not just what — `git diff` shows what.

## Coverage gate

`make cover` enforces a 70% floor. Drop below and the build fails. The
project's own coverage discipline is documented in the test plan
embedded in tests (`TestPICT_*`, `TestContract_*`, fuzz targets);
read those before changing the gate threshold.

## Branch protection: re-applying the rule

If the protection rule on `main` ever gets removed (admin error, GitHub
UI mishap), reapply it with:

```bash
gh api repos/kevin-burns/c7search/branches/main/protection \
  -X PUT \
  --input - <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "test (ubuntu-latest)",
      "test (macos-latest)",
      "test (windows-latest)",
      "lint",
      "govulncheck",
      "gitleaks"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 0,
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true,
  "block_creations": false
}
JSON
```

The settings worth understanding:

- `enforce_admins: false` — the repo owner can override required checks
  and the PR requirement. Use only for emergency hotfixes (broken main,
  security advisory, etc.). Every override leaves a "Bypassed rule
  violations" line in the push output, which is the audit trail.
- `required_approving_review_count: 0` — solo project. CI is the gate.
  Bump to `1` if collaborators arrive and you want human review too.
- `required_linear_history: true` — squash or rebase, no merge commits.
- `allow_force_pushes: false` — absolute. Don't toggle this off.

## Local hooks vs CI

The two should agree. The `.pre-commit-config.yaml` pins
`golangci-lint v2.11.4`, the same version `.github/workflows/test.yml`
runs in CI. When you bump one, bump the other.

## Reporting security issues

Don't open a public issue. See [SECURITY.md](../SECURITY.md) for the
private disclosure flow.
