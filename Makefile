# Version information from git
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# ldflags to inject version info
LDFLAGS := -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)

.PHONY: test install build setup-hooks

test:
	go test ./...

# Full security audit: dependency vulns + code patterns + secrets
audit:
	@echo "=== govulncheck ==="
	govulncheck ./...
	@echo ""
	@echo "=== gosec ==="
	gosec -quiet -severity=medium ./... || true
	@echo ""
	@echo "=== gitleaks ==="
	gitleaks detect --source . -v 2>/dev/null || echo "gitleaks not installed (go install github.com/gitleaks/gitleaks/v8@latest)"
	@echo ""
	@echo "=== Audit complete ==="


build:
	go build -ldflags="$(LDFLAGS)" -o templar ./cmd/templar

install:
	go build -ldflags="$(LDFLAGS)" -o ${GOBIN}/templar ./cmd/templar

setup-hooks:
	git config core.hooksPath .githooks
