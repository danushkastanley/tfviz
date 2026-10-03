# Tested compatibility

tfviz claims compatibility only with what it tests. Other versions may work, but they are not promised.

## Producers

| Producer | Versions tested | Inputs |
| --- | --- | --- |
| Terraform | 1.16.4 | Plan JSON (format 1.2), state JSON (format 1.0), version-4 `.tfstate` |
| OpenTofu | 1.13.1 | Plan JSON (format 1.2), state JSON (format 1.0), version-4 `.tfstate` |

- **Format versions.** JSON format major version 1 is accepted. A newer minor version is read with a warning (and fails under `--strict`). Any other major version is rejected.
- **Encrypted OpenTofu state.** Export it with `tofu show -json` first.

## Providers

| Provider | Version tested | Coverage |
| --- | --- | --- |
| `hashicorp/aws` | 6.67.0 | See [support.md](support.md) |

Resources from any other provider appear with limited detail and no placement.

## Browsers

Reports are tested from `file://` with no network access, in the Playwright 1.63 builds of Chromium, Firefox and WebKit, on macOS and Linux. Branded Chrome, Firefox and Safari are expected to behave the same, but have not yet been verified on separate devices.

## Platforms

Release binaries are built for macOS and Linux, on arm64 and amd64. Windows is planned.

## Fixtures

Every claim above is backed by synthetic fixtures in [testdata](../testdata/README.md). They are regenerated with `make fixtures` and compared against golden reports in CI.
