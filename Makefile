SHELL := /bin/bash

GO ?= go
GOFMT ?= gofmt
BIN_NAME ?= noria
CMD_NAME ?= noria
CMD_PATH ?= ./cmd/$(CMD_NAME)
DIST_DIR ?= dist
BIN_PATH ?= $(DIST_DIR)/$(BIN_NAME)
VERSION ?= $(shell cat VERSION)
LDFLAGS ?= -s -w -X github.com/amxv/noria/internal/buildinfo.Version=$(if $(VERSION),$(VERSION),dev)

.PHONY: help fmt test test-race vet lint check docs-install docs-dev docs-check docs-build docs-preview build build-all install-local clean release-tag

help:
	@echo "Noria development and release commands"
	@echo ""
	@echo "Targets:"
	@echo "  make fmt          - format Go files"
	@echo "  make test         - run go test ./..."
	@echo "  make vet          - run go vet ./..."
	@echo "  make lint         - validate native installer and release scripts"
	@echo "  make check        - fmt + test + vet + lint"
	@echo "  make docs-install  - install docs dependencies"
	@echo "  make docs-dev      - run the Astro/ZueDocs dev server"
	@echo "  make docs-check    - validate the Astro/ZueDocs site"
	@echo "  make docs-build    - build the Astro/ZueDocs site"
	@echo "  make docs-preview  - preview the built Astro/ZueDocs site"
	@echo "  make build        - build local binary to dist/noria"
	@echo "  make build-all    - build release binaries for 6 target platforms"
	@echo "  make install-local - install CLI to ~/.local/bin/noria"
	@echo "  make clean        - remove dist artifacts"
	@echo "  make release-tag VERSION=x.y.z - create and push a tag for GitHub Actions"

fmt:
	@$(GOFMT) -w $$(find cmd internal -type f -name '*.go')

test:
	@$(GO) test ./...

test-race:
	@$(GO) test -race ./...

vet:
	@$(GO) vet ./...

lint:
	@sh -n scripts/install.sh
	@diff -q scripts/install.sh docs/public/install.sh
	@diff -q scripts/install.ps1 docs/public/install.ps1
	@python3 -m unittest discover -s scripts -p 'test_*.py'

check: fmt test vet lint

docs-install:
	@cd docs && bun install --frozen-lockfile

docs-dev:
	@cd docs && bun run dev

docs-check:
	@cd docs && bun run test:deployment
	@cd docs && bun run check

docs-build:
	@cd docs && bun run build

docs-preview:
	@cd docs && bun run preview

build:
	@mkdir -p $(DIST_DIR)
	@$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN_PATH) $(CMD_PATH)

build-all:
	@mkdir -p $(DIST_DIR)
	@set -e; for target in "darwin amd64" "darwin arm64" "linux amd64" "linux arm64" "windows amd64" "windows arm64"; do \
		set -- $$target; \
		GOOS=$$1; GOARCH=$$2; \
		EXT=""; \
		if [ "$$GOOS" = "windows" ]; then EXT=".exe"; fi; \
		echo "Building $(BIN_NAME) for $$GOOS/$$GOARCH"; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o "$(DIST_DIR)/$(BIN_NAME)_$${GOOS}_$${GOARCH}$${EXT}" $(CMD_PATH); \
	done

install-local: build
	@mkdir -p $$HOME/.local/bin
	@install -m 755 $(BIN_PATH) $$HOME/.local/bin/$(BIN_NAME)
	@echo "Installed $(BIN_NAME) to $$HOME/.local/bin/$(BIN_NAME)"

clean:
	@rm -rf $(DIST_DIR)

release-tag:
	@test -n "$(VERSION)" || (echo "Usage: make release-tag VERSION=x.y.z" && exit 1)
	@python3 scripts/release.py check "v$(VERSION)"
	@git tag "v$(VERSION)"
	@git push origin "v$(VERSION)"
