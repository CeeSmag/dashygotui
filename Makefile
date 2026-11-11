SHELL := /bin/bash

# Pinned tool versions (run through `go run`, so nothing needs installing).
STATICCHECK   := honnef.co/go/tools/cmd/staticcheck@v0.8.2
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK   := golang.org/x/vuln/cmd/govulncheck@v1.9.0
GOIMPORTS     := golang.org/x/tools/cmd/goimports@v0.52.0

.PHONY: check fmt fmt-check vet staticcheck lint vuln test race deletion-test template-test tidy-check

## check: the full gate required by RULES.md before any task is done.
check: fmt-check vet staticcheck lint vuln race deletion-test template-test tidy-check

## fmt: rewrite files with gofmt and goimports.
fmt:
	gofmt -w .
	go run $(GOIMPORTS) -local github.com/CeeSmag/dashygotui -w .

fmt-check:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	@out="$$(go run $(GOIMPORTS) -local github.com/CeeSmag/dashygotui -l .)"; \
	if [ -n "$$out" ]; then echo "goimports needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

staticcheck:
	go run $(STATICCHECK) ./...

lint:
	go run $(GOLANGCI_LINT) run ./...

vuln:
	go run $(GOVULNCHECK) ./...

test:
	go test ./...

## race: tests with the race detector; includes the architecture guard tests.
race:
	go test -race -count=1 ./...

## deletion-test: every addon file must be removable without breaking the build.
deletion-test:
	./scripts/deletion-test.sh

## template-test: every addon template must compile and register cleanly.
template-test:
	./scripts/template-test.sh

tidy-check:
	go mod tidy -diff
