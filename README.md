# tfviz

Turn Terraform and OpenTofu plans and state into calm, interactive infrastructure diagrams: a single, self-contained HTML file you can open offline or download from CI.

> **Status: pre-alpha.** Nothing here is ready for real infrastructure yet. The current milestone (M0) validates the report format, the visual design and offline export using synthetic fixtures only. See [docs/implementation-plan.md](docs/implementation-plan.md).

## Principles

- **Local and private.** No account, no hosted service and no telemetry. Local-file workflows run entirely offline.
- **Sanitised by construction.** The renderer only ever receives an approved, projected report model. Raw Terraform attributes, secrets and input variables never reach the HTML.
- **Honest semantics.** State is a recorded snapshot and a plan is a proposal. Relationships show recorded associations, not observed traffic.

## Planned usage

The interface below is proposed and not yet implemented.

```bash
terraform plan -out=tfplan
terraform show -json tfplan | tfviz plan --input - --output infra-report.html
```

## Development

Requirements: Go 1.27+, Node 26+, pnpm 12+. Fixture regeneration also needs Terraform and OpenTofu.

```bash
make test     # Go and web checks
make build    # build the web bundle, then the Go binary
make e2e      # offline browser checks with Playwright
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## Licence

Apache-2.0. See [LICENSE](LICENSE).
