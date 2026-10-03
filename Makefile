SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c

GO ?= go

.PHONY: test build go-test fmt-check web web-install web-test e2e fixtures

test: web-test go-test

build: web
	$(GO) build -trimpath -o bin/tfviz ./cmd/tfviz

go-test: web fmt-check
	$(GO) vet ./...
	$(GO) test ./...

fmt-check:
	@unformatted=$$(gofmt -l $$($(GO) list -f '{{.Dir}}' ./... 2>/dev/null) </dev/null); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

# Builds the interface bundle and places it where the Go renderer embeds it.
web: web-install
	pnpm --dir web run build
	mkdir -p internal/report/html/dist
	cp web/dist/tfviz.js web/dist/tfviz.css web/dist/THIRD_PARTY_NOTICES.txt internal/report/html/dist/

web-install:
	pnpm --dir web install --frozen-lockfile

web-test: web-install
	pnpm --dir web run schema:check
	pnpm --dir web run typecheck
	pnpm --dir web run lint
	pnpm --dir web run test

# Opens generated reports over file:// in Chromium, Firefox and WebKit.
# First run: pnpm --dir web exec playwright install chromium firefox webkit
e2e: web
	pnpm --dir web run e2e

# Regenerates producer JSON for the synthetic stack. Needs terraform and tofu.
# Only `init` touches the network (provider download); no AWS calls are made.
fixtures:
	$(GO) run ./scripts/fixturegen -tool terraform -out testdata/producer/terraform-1.16
	$(GO) run ./scripts/fixturegen -tool tofu -out testdata/producer/opentofu-1.13
