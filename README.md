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
make test     # Go and web checks (builds the web bundle first)
make build    # web bundle plus the Go binary in bin/
make e2e      # renders reports and opens them offline via file:// in three browsers
```

In M0 the only binary is `tfviz-spike`. It renders a report model JSON file to HTML:

```bash
make build
bin/tfviz-spike -in testdata/reports/aws-review.sample.json -out out/sample-report.html
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## Licence

Apache-2.0. See [LICENSE](LICENSE).
