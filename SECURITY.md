# Security policy

tfviz reads Terraform/OpenTofu plans and state, which can contain plaintext secrets. Preventing disclosure is a core product requirement, not an optional hardening step.

## Reporting a vulnerability

Please report suspected vulnerabilities privately through [GitHub security advisories](https://github.com/danushkastanley/tfviz/security/advisories/new). Do not open a public issue.

Include the tfviz version, the command you ran and a minimal synthetic input that reproduces the problem. **Never send real plans, state files or secrets.**

## Scope

These are in scope:

- Secret or sensitive values appearing in generated reports, logs, errors or the local explorer.
- Script injection or unsafe rendering of labels in the HTML report.
- Local explorer access from other browser origins.
- Unbounded resource use from crafted inputs.

## Supported versions

tfviz is pre-alpha. Fixes land on `main` only until the first tagged release.
