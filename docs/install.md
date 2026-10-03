# Installing tfviz

tfviz is one binary with the interface built in. End users need no Node.js, no server and no account.

## Download a release

Releases provide binaries for macOS and Linux, on arm64 and amd64. Each release includes a SHA-256 checksum file, an SBOM for each archive, and GitHub build provenance.

```bash
VERSION=0.1.0                     # choose and pin a version
OS=darwin ARCH=arm64              # or linux / amd64
base=https://github.com/danushkastanley/tfviz/releases/download/v${VERSION}
curl -fsSLO "${base}/tfviz_${VERSION}_${OS}_${ARCH}.tar.gz"
curl -fsSLO "${base}/tfviz_${VERSION}_checksums.txt"
```

Verify the archive before you run anything:

```bash
grep "tfviz_${VERSION}_${OS}_${ARCH}.tar.gz" "tfviz_${VERSION}_checksums.txt" | shasum -a 256 -c -
gh attestation verify "tfviz_${VERSION}_${OS}_${ARCH}.tar.gz" --repo danushkastanley/tfviz
```

Then extract it and put `tfviz` on your `PATH`:

```bash
tar -xzf "tfviz_${VERSION}_${OS}_${ARCH}.tar.gz" tfviz
sudo install -m 0755 tfviz /usr/local/bin/tfviz
tfviz version
```

Pin a specific version in CI. Do not pipe a remote install script into a shell.

## Build from source

You need Go 1.27+, Node 26+ and pnpm 12+.

```bash
git clone https://github.com/danushkastanley/tfviz.git
cd tfviz
make build        # web bundle first, then bin/tfviz
```

`go install` is not supported, because the binary embeds the web bundle and that has to be built first.

Windows builds will follow once their file-opening and permission behaviour has been tested.
