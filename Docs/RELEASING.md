# Release process

GitHub Actions publishes stable Windows x64 releases from annotated semantic-version tags. The tagged commit must already contain the intended executable, matching release-test hash, documentation, notices, and SBOM metadata.

## 1. Prepare the release commit

Choose a stable version such as `1.1.1`. Prerelease forms such as `1.1.1-rc.1` are not accepted by the automated workflow.

Build with Go 1.26.5 on Windows x64:

```powershell
$env:GOPROXY = "off"
$env:GOSUMDB = "off"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o smart-frame-sorter.exe ./src
```

Update the expected executable SHA-256 in `src/release_test.go` when it changes:

```powershell
(Get-FileHash .\smart-frame-sorter.exe -Algorithm SHA256).Hash.ToLowerInvariant()
```

Set the SmartFrameSorter package `versionInfo` and document name in `sbom.spdx.json` to the release version. Update build metadata, notices, licenses, hashes, and documentation whenever their inputs change.

## 2. Verify locally

```powershell
$env:GOPROXY = "off"
$env:GOSUMDB = "off"
$env:CGO_ENABLED = "0"
gofmt -w src\*.go
go vet ./...
go test ./...
```

Inspect the command-line help and perform a smoke test with representative images and MP4 files. Do not release if formatting, vet, tests, artifact hashes, licenses, the SBOM, FFmpeg protocol checks, or the smoke test fail.

## 3. Package locally when needed

The packaging script remains available for local verification:

```powershell
.\scripts\package-release.ps1 -Version 1.1.1
```

It refuses to overwrite any existing release output and creates:

- `release\SmartFrameSorter-1.1.1-windows-x64`, the unpacked portable bundle;
- `release\SmartFrameSorter-1.1.1-windows-x64.zip`, the portable GitHub asset;
- `release\sources\SmartFrameSorter-1.1.1-source.zip`, the corresponding project source; and
- `release\SHA256SUMS`, covering both ZIP files, the exact FFmpeg source, and every file in the unpacked bundle.

The exact FFmpeg source remains `release\sources\ffmpeg-8.1.2.tar.xz` until FFmpeg is intentionally upgraded.

## 4. Publish from an annotated tag

Commit and push the complete release state, then create an annotated tag. Its message becomes the public GitHub Release notes:

```powershell
git tag -a v1.1.1 -m "SmartFrameSorter 1.1.1`n`nDescribe the user-visible changes here."
git push origin v1.1.1
```

The `Publish release` workflow rejects lightweight tags, malformed versions, prerelease versions, empty tag messages, and SBOM version mismatches. It repeats formatting, vet, build, and all tests before packaging. Only its publication job receives `contents: write` permission.

On success, the workflow creates a GitHub Release titled `SmartFrameSorter 1.1.1` and attaches:

- `SmartFrameSorter-1.1.1-windows-x64.zip`;
- `SmartFrameSorter-1.1.1-source.zip`;
- `ffmpeg-8.1.2.tar.xz`; and
- `SHA256SUMS`.

Ordinary pushes and pull requests run CI with read-only permissions and never upload or publish binaries.

## 5. Verify published assets

Download the public assets to a clean Windows x64 environment. Verify their entries in `SHA256SUMS`, extract the portable bundle, and repeat the smoke test. Confirm the bundle contains the documentation, licenses, notices, SPDX SBOM, and FFmpeg build metadata.
