# tfviz: repository instructions

Read [docs/implementation-plan.md](docs/implementation-plan.md) before you make a substantive change. It is the product source of truth.

## Non-negotiables

- **No raw attributes downstream.** Only the sanitised report model (`schema/report.v1.schema.json`) may reach the renderer or the frontend. Never add a generic `map[string]any` or `attributes` field to the report.
- **Sensitivity wins.** A field that is marked sensitive is exported with no value payload, and any relationship derived from it is suppressed.
- **Canaries.** Every fixture secret is listed in `testdata/canaries.txt`. Tests must prove that none of them appears in report bytes, emitted JSON, stdout or stderr.
- **Offline HTML.** Reports make no network requests and work from `file://` under the generated CSP. Never use `unsafe-eval` or `unsafe-inline`.
- **Determinism.** Report output is ordered deterministically. Time is injected, never read implicitly in core logic.
- **File size.** Hand-written source files stay under 400 lines; split files by responsibility.
- **No new dependencies** without an explicit decision.

## Commands

- `make test`: runs Go vet and tests, then web typecheck, lint and unit tests.
- `make build`: builds the web bundle and copies it into `internal/report/assets/dist`, then builds the Go binary.
- `make e2e`: runs Playwright under `file://` in Chromium, Firefox and WebKit.
- `make fixtures`: regenerates producer JSON with Terraform and OpenTofu. It uses fake credentials and makes no AWS calls.

## Git

Use `feature/*` branches and PRs. Commits and PRs carry no AI attribution.
