# Security model

tfviz reads documents that can contain secrets: Terraform and OpenTofu plans and state. This page describes what tfviz protects, how it does so, and where the limits are.

## Trust boundaries

```text
plan/state (secret-bearing)
   │  internal/input: the only code that sees raw values
   ▼
snapshot (sensitive values reduced to payload-free markers)
   │  internal/projection + provider adapters: approved fields only
   ▼
sanitised report model (schema/report.v1.schema.json)
   │  internal/report/html or the local explorer
   ▼
HTML report / browser
```

## Controls

| Risk | Control | Evidence |
| --- | --- | --- |
| Secrets in the report | The reader turns sensitive values into markers with no payload. Only allowlisted fields are exported. A deny guard withholds secret-shaped names even if an adapter allowlists them. Tags, descriptions, policies, user data, environment variables and secret contents are withheld | Canary tests scan every report, stdout and stderr for planted secrets |
| Secrets revealed by relationships | A sensitive reference suppresses its relationship | Graph test |
| Script injection from labels | Report data is serialised JSON with `<`, `>`, `&` and U+2028/9 escaped. The interface renders text only; `innerHTML` and `dangerouslySetInnerHTML` are banned by lint | Hostile-label tests in Go and in three browsers, plus fuzzing |
| Network access from reports | The CSP allows only the inlined script and style hashes, `img-src data:`, and `connect-src 'none'` | Playwright checks for zero requests and confirms that an injected script is blocked |
| Hostile icon files (`--icons`) | Only regular `.svg` files under the chosen folder, size-limited and checked for an `<svg>` root. They are embedded as `data:image/svg+xml` and shown only through `<img>`, where browsers run no scripts and load nothing | Loader tests, schema pattern, and a browser test under the CSP |
| Malicious input | Size, depth and resource limits, a single document only, and errors that never echo input | Fuzz tests |
| Other sites reading the local explorer | Loopback only, a Host check, a one-time token exchanged for a SameSite=Strict cookie, no CORS, no-store, and CSRF protection on refresh | Server tests, plus a cross-origin test in three browsers |
| Credential exposure | S3 credentials stay inside the AWS SDK. tfviz never starts an SSO login and calls no secret-value APIs | S3 tests in an isolated environment |
| Identifying details when sharing | `--safe-share` replaces them with consistent local stand-ins, and the mapping is never written | Safe-share leak tests |
| Supply chain | Pinned dependencies and actions, a minimum release age for npm packages, licence allowlists, checksums, SBOMs and build provenance | CI and the release workflow |

## Limits

- **Normal reports are sensitive.** They show approved names, IDs, ARNs and network details by design, so protect them like other infrastructure artefacts.
- **Safe-share is not anonymity.** The architecture itself may be confidential.
- **Names can carry secrets.** No heuristic can prove that an arbitrary name or identifier is harmless.
- **No guaranteed erasure.** tfviz releases raw input once it is parsed, but Go cannot guarantee memory is erased.
- **Snapshots are not live.** State and plans describe a recorded moment, and relationships are recorded associations, not observed traffic.

Report vulnerabilities as described in [SECURITY.md](../SECURITY.md).
