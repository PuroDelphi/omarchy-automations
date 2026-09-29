# Runtime packages and reproducibility

[Español](../es/packages.md) · [Installation](installation.md)

The packager produces development artifacts for Linux amd64. Publishing a preview
does not complete the deferred roadmap checks or make it a stable release.
Only verified architectures are advertised. The no-compiler setup is described
in the installation guide; this chapter explains how maintainers build packages.

## Build from source

```bash
make build
python3 scripts/package-release.py
python3 scripts/test-release-package.py
python3 scripts/test-reproducible-build.py
```

`make build` uses `-trimpath -buildvcs=false`: binaries omit local build paths and
changing repository state. Use the Go version declared in `go.mod` and dependencies
pinned by `go.sum`; reproducing a build requires the same toolchain, source,
platform and flags. Archive determinism alone does not prove compiler reproducibility.

The output is `dist/omarchy-automations-<version>-linux-amd64.tar.gz` and its
`.sha256` file. Version comes from the binaries and must match the QML contract
and manifest base version. `release.json` records version, contracts, platform
and each file's SHA256. The optional broker retains its separate protocol.

The tar.gz normalizes ordering, permissions, ownership and timestamps, including
the gzip header. Explicit input roots include binaries, UI, installers, templates,
examples, schemas and documentation. Profiles, Git metadata, caches and credentials
are excluded; symlink inputs are rejected.

## Install an extracted package

From the download directory, substitute the actual version for `<version>`:

```bash
sha256sum -c omarchy-automations-<version>-linux-amd64.tar.gz.sha256
tar -xzf omarchy-automations-<version>-linux-amd64.tar.gz
cd omarchy-automations-<version>
python3 scripts/install.py --activate
```

Checksums detect corruption; authenticity depends on obtaining them from a trusted
publisher. This is a runtime archive, not a source checkout for `make build`.
The broker has a separate optional installation procedure and is not enabled by
extracting or installing the user package. For a Git-managed Omarchy panel use
`setup.py` from that checkout instead of installing a second managed UI over it.

## What is verified

`test-release-package.py` builds two byte-identical archives, validates hashes and
runs install/update/uninstall from extracted content in a staging HOME, without
activating host services. `test-setup.py` checks prebuilt setup in managed and
Git-panel modes, preserves checkout files, and rejects corrupt or unsafe archives.

Independent compilation was also checked with Go 1.26.8 on Linux amd64: two source
trees with initially empty build caches produce identical binaries. They share
the toolchain and module cache; `GOPROXY=off` prevents downloads. This evidence does
not establish reproducibility on other toolchains or platforms.

The project uses [MIT](../../LICENSE); preserve the
[third-party notices](../third-party-notices.md) as well. Final stable-release
acceptance remains tracked in the [roadmap](../../ROADMAP.md).
