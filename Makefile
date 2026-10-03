SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c

GO ?= go

.PHONY: test go-test fmt-check fixtures

test: go-test

go-test: fmt-check
	$(GO) vet ./...
	$(GO) test ./...

fmt-check:
	@unformatted=$$(gofmt -l $$($(GO) list -f '{{.Dir}}' ./... 2>/dev/null) </dev/null); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

# Regenerates producer JSON for the synthetic stack. Needs terraform and tofu.
# Only `init` touches the network (provider download); no AWS calls are made.
fixtures:
	$(GO) run ./scripts/fixturegen -tool terraform -out testdata/producer/terraform-1.16
	$(GO) run ./scripts/fixturegen -tool tofu -out testdata/producer/opentofu-1.13
