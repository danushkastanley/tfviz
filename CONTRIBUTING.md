# Contributing

Thank you for helping. tfviz handles sensitive infrastructure data, so changes are held to a high bar for privacy and correctness.

## Ground rules

- Use only synthetic infrastructure in fixtures, issues and pull requests. Never commit real plans, state, account IDs or secrets.
- Keep pull requests small and focused on one behaviour.
- Add or update tests with every behaviour change. Security-relevant code paths need canary tests.
- Do not add dependencies without discussing them in an issue first.
- Use UK English in user-facing text.

## Local setup

```bash
make test
make build
make e2e        # first run: pnpm --dir web exec playwright install
make fixtures   # needs Terraform and OpenTofu; makes no AWS API calls
```

## Layout

| Path | Responsibility |
| --- | --- |
| `cmd/tfviz/` | CLI entry point |
| `internal/input/` | Producer JSON and raw-state readers |
| `internal/projection/` | Sensitivity rules and approved metadata |
| `internal/provider/aws/` | AWS resource and relationship adapters |
| `internal/graph/` | Graph assembly, grouping and change logic |
| `internal/report/` | Report schema types, HTML export and serving |
| `schema/` | Versioned report JSON Schema (source of truth) |
| `web/` | React/TypeScript interface |
| `testdata/` | Synthetic fixtures and canaries |
