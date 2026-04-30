VERSION ?= dev

.PHONY: build test test-race lint vuln secrets-scan cover cover-html fuzz tidy clean precommit-install

build:
	go build -trimpath -ldflags "-s -w -X github.com/kevin-burns/c7search/internal/version.Version=$(VERSION)" -o c7search .

test:
	go test ./...

# Race detector enabled. Slower (~2x) but catches concurrency bugs that
# only surface under load. CI uses this target.
test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

# Go's official vuln DB; call-graph-aware so noise is low.
vuln:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Same scan CI and pre-commit run. Useful when adding fixtures.
secrets-scan:
	gitleaks detect --source . --config .gitleaks.toml --no-banner

# Coverage gate. Fails the build if total coverage dips below the
# threshold. The seam at root.go (var newAPIClient / var newCacheStore)
# lets cobra commands run against an httptest server, so 70% is honest.
COVER_MIN := 70.0
cover:
	@go test -coverprofile=coverage.out ./... > /dev/null
	@total=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
	echo "coverage: $${total}%"; \
	awk -v p="$$total" -v m="$(COVER_MIN)" \
	    'BEGIN { exit (p+0 < m+0) }' || \
	    { echo "FAIL: coverage $${total}% below threshold $(COVER_MIN)%"; exit 1; }

cover-html: cover
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

# Opt-in fuzz harness. CI doesn't run this -- use it locally before
# touching parsers that eat untrusted input.
fuzz:
	go test -fuzz=FuzzNormalizeLibraryID -fuzztime=30s ./internal/client/...
	go test -fuzz=FuzzExtractTopic     -fuzztime=30s ./internal/cli/...

tidy:
	go mod tidy

clean:
	rm -f c7search c7search.exe coverage.out coverage.html
	rm -rf dist/

precommit-install:
	pre-commit install
