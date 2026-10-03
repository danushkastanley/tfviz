# Using tfviz in CI

The pattern is the same everywhere:

1. Create the plan.
2. Export it to JSON.
3. Generate the report.
4. Upload only the report.

The exported JSON can contain plaintext secrets, so pipe it straight into tfviz rather than saving it as an artefact. If your apply step needs the binary plan, keep it separately with restricted access. The report does not replace it.

These examples assume a pinned, verified tfviz binary is already installed. See [install.md](install.md).

## GitHub Actions

```yaml
- name: Plan and report
  run: |
    set -euo pipefail
    umask 077
    mkdir -p artefacts
    terraform plan -input=false -out=tfplan
    terraform show -json tfplan | tfviz plan --input - --output artefacts/infra-report.html --force
- uses: actions/upload-artifact@<pinned-sha> # pin to a commit SHA
  with:
    name: infra-report
    path: artefacts/infra-report.html
    retention-days: 14
```

## Jenkins

```groovy
pipeline {
  agent any
  stages {
    stage('Infrastructure review report') {
      steps {
        sh(script: '''#!/usr/bin/env bash
set -euo pipefail
set +x
umask 077
mkdir -p artefacts
plan_path=$(mktemp)
trap 'rm -f "$plan_path"' EXIT
terraform plan -input=false -out="$plan_path"
terraform show -json "$plan_path" | tfviz plan --input - --output artefacts/infra-report.html --force
''')
      }
    }
  }
  post {
    success {
      archiveArtifacts(artifacts: 'artefacts/infra-report.html', fingerprint: true, allowEmptyArchive: false)
    }
  }
}
```

Jenkins usually serves archived artefacts under a restrictive Content Security Policy that blocks scripts. Download the report and open it locally; that is the supported path. Don't weaken the controller's security settings just to display a diagram.

## GitLab CI

```yaml
infra-report:
  script:
    - umask 077 && mkdir -p artefacts
    - terraform plan -input=false -out=tfplan
    - terraform show -json tfplan | tfviz plan --input - --output artefacts/infra-report.html --force
  artifacts:
    paths: [artefacts/infra-report.html]
    expire_in: 2 weeks
```

## Buildkite

```yaml
steps:
  - label: "Infrastructure review report"
    command: |
      set -euo pipefail
      umask 077 && mkdir -p artefacts
      terraform plan -input=false -out=tfplan
      terraform show -json tfplan | tfviz plan --input - --output artefacts/infra-report.html --force
    artifact_paths: "artefacts/infra-report.html"
```

## OpenTofu

Replace `terraform` with `tofu`. Encrypted OpenTofu plans need your encryption configuration when you run `tofu show -json`.

## Failing the pipeline on gaps

A plan that changes infrastructure is a success (exit code 0). Add `--strict` to fail when anything cannot be fully interpreted, for example an unsupported resource type or a plan the producer reported as incomplete.

| Exit code | Meaning |
| --- | --- |
| `0` | Report written. |
| `1` | Processing or retrieval failed. |
| `2` | Invalid arguments, unsupported input, or a `--strict` failure. |

## Sharing outside the team

Reports show approved names, IDs and network details. Apply your normal artefact access and retention controls. Use `--safe-share` for audiences who shouldn't see identifying details.
