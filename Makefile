SHELL := /bin/bash

# Pinned tool versions (run through `go run`, so nothing needs installing).
STATICCHECK   := honnef.co/go/tools/cmd/staticcheck@v0.8.2
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK   := golang.org/x/vuln/cmd/govulncheck@v1.9.0
GOIMPORTS     := golang.org/x/tools/cmd/goimports@v0.52.0

.PHONY: help check fmt fmt-check vet staticcheck lint vuln test race deletion-test template-test tidy-check example examples run run-tty run-demo dist clean

# The example apps: showcase uses every feature; demo is the small quickstart.
BIN       := bin
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

## help: list the targets.
help:
	@grep -hE '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | awk -F': ' '{ printf "  %-14s %s\n", $$1, $$2 }'

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

## example: build the showcase (it uses every feature) into bin/showcase.
example:
	@mkdir -p $(BIN)
	go build -trimpath -o $(BIN)/showcase ./examples/showcase
	@echo "built $(BIN)/showcase  (try: ./$(BIN)/showcase -h)"

## examples: build every example app, the showcase and the small demo, into bin/.
examples: example
	go build -trimpath -o $(BIN)/demo ./examples/demo
	@echo "built $(BIN)/demo"

## run: run the showcase without building a binary first.
run:
	go run ./examples/showcase

## run-tty: run the showcase in TTY mode, the Linux console symbol set.
run-tty:
	DASHYGOTUI_TTY=1 go run ./examples/showcase

## run-demo: run the small demo app.
run-demo:
	go run ./examples/demo

## dist: cross-compile the showcase for Linux, macOS and Windows into bin/dist/.
dist:
	@mkdir -p $(BIN)/dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; [ "$$os" = windows ] && ext=".exe"; \
		out="$(BIN)/dist/showcase-$$os-$$arch$$ext"; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$$out" ./examples/showcase || exit 1; \
	done

## clean: delete the binaries the targets above built. Only those files; bin/ itself stays.
clean:
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; [ "$$os" = windows ] && ext=".exe"; \
		rm -f "$(BIN)/dist/showcase-$$os-$$arch$$ext"; \
	done
	rm -f $(BIN)/showcase $(BIN)/demo
