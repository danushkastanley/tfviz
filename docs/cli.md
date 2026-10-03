# Command line

tfviz reads the JSON that Terraform or OpenTofu exports with `show -json`, and writes one self-contained HTML report. The report opens directly from disk and makes no network requests.

## Generate a plan review

```bash
terraform plan -out=tfplan
terraform show -json tfplan | tfviz plan --input - --output infra-report.html
```

OpenTofu works the same way:

```bash
tofu plan -out=tfplan
tofu show -json tfplan | tfviz plan --input - --output infra-report.html
```

Piping avoids writing the exported JSON to disk. The export can contain plaintext secrets: tfviz removes them from the report, but it cannot sanitise the producer's own plan or state files.

## Generate a state report

```bash
terraform show -json | tfviz state --input - --output state-report.html
```

A state report shows Terraform's recorded snapshot. It does not prove that the infrastructure in AWS is currently identical.

## Options

| Option | Commands | Meaning |
| --- | --- | --- |
| `--input <file\|->` | plan, state | Exported JSON file, or `-` for standard input. Required. |
| `--output <file>` | plan, state | HTML report to write. Required. Written atomically with permissions `0600`. |
| `--title <text>` | plan, state | Report title. |
| `--force` | plan, state | Replace the output file if it exists. Without it, an existing file is never touched. |
| `--strict` | plan, state | Fail if anything cannot be fully interpreted: unsupported resource types, unrecognised actions, a plan the producer reported as incomplete, skipped deposed objects, or a newer JSON format. No report is written. |

`tfviz version` prints the version.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The report was written. A plan that changes infrastructure is still a success. |
| `1` | Processing failed, for example an unreadable input file or an output that could not be written. |
| `2` | Invalid arguments, unsupported input, an output that already exists without `--force`, or a `--strict` failure. |

## What tfviz prints

On success it prints one line to standard error with counts only. For example:

```text
Wrote infra-report.html: 39 resources (38 fully supported, 1 with limited detail). Changes: 3 to add, 6 to change, 2 to replace, 1 to destroy, 1 read during apply. 1 unresolved reference(s).
```

It never prints input content. Error messages explain what to do next and never echo the document.

## Inputs it rejects, with guidance

- **Raw `.tfstate` files.** Export them with `terraform show -json` first.
- **Terraform's streaming `-json` log output.** Save the plan with `-out` and export it instead.
- **Encrypted OpenTofu state or plans.** Export them with `tofu show -json` using your encryption configuration.
- **A plan passed to `state`, or state passed to `plan`.**
- **JSON format major versions other than 1.**
- **Inputs over the size limits.** These are 512 MiB, nesting deeper than 128 levels, or more than 50,000 resources.
