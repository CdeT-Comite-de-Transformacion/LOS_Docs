# Build and release modkit

The starter kit is a standalone Go module inside this repository. Its embedded
DDD templates ship inside each binary; end users do not need Go to run modkit.
Go is required to compile and run the projects it generates.

## Continuous integration

Pull requests and branch pushes affecting the starter kit run tests and vet on
Windows, Linux, and macOS, then build amd64 and arm64 binaries with CGO disabled.
Download the six compressed packages from the workflow's artifacts. Manual
workflow runs also produce development packages without publishing a release.

## Versioning and publication

Stable releases use Semantic Versioning tags (`vMAJOR.MINOR.PATCH`). The tag is
the version source and is injected into `modkit version`. Local builds report
`dev`; CI branch builds report `dev-<commit>`. Prerelease tags are not supported.

After committing and pushing the integration, publish the first version from
the repository root:

```bash
git tag -a v0.1.0 -m "Release modkit v0.1.0"
git push origin v0.1.0
```

Tag pushes automatically test, build, package, and publish a GitHub Release with
generated release notes and `checksums.txt`. Publication requires all three
platform jobs to succeed. The workflow uses the built-in `GITHUB_TOKEN` with
`contents: write` only in the release job; no custom token is required. Repository
or organization policies must allow this permission. Use a new tag for each
release; publishing an existing release fails rather than overwriting assets.

| Platform | Architectures | Package |
| --- | --- | --- |
| Windows | amd64, arm64 | `modkit_<version>_windows_<arch>.zip` |
| Linux | amd64, arm64 | `modkit_<version>_linux_<arch>.tar.gz` |
| macOS | amd64, arm64 | `modkit_<version>_darwin_<arch>.tar.gz` |

macOS arm64 is for Apple Silicon; amd64 is for Intel. Archives contain the
binary, usage documentation, and an example configuration. Binaries are unsigned.

## Install and generate DDD modules

Download the package matching your OS and CPU from this repository's Releases
page and verify its SHA256 against `checksums.txt` before extracting it. On
Linux, use `sha256sum`; on macOS, use `shasum -a 256`; on Windows, use
`Get-FileHash -Algorithm SHA256`. Put the extracted binary in a directory on PATH.

```bash
modkit version
mkdir payments-api
cd payments-api
modkit init --name payments-api
modkit module create credit-request --create-all
modkit module validate credit-request
go mod tidy
go test ./...
```

These commands create Go source packages for domain, application,
infrastructure, and interfaces. The release packages distribute the generator;
they do not package the generated business modules as executables.

## Local verification

From `starter-kit/`:

```bash
go test ./...
go vet ./...
go build -o bin/modkit .
```

On Windows, use `go build -o bin/modkit.exe .`.
