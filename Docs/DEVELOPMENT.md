# Development and testing

## Prerequisites

- Go 1.24 or newer.
- Windows x64 for the complete test suite and Windows executable.
- PowerShell for release packaging.
- The checked-in `ffmpeg` tools and exact source archive for release compliance tests.

The Go dependency needed by SmartFrameSorter is vendored under `third_party/x_image`; no network download is required.

## Build offline

From the repository root:

```powershell
$env:GOPROXY = "off"
go build -trimpath -ldflags="-s -w" -o smart-frame-sorter.exe ./src
```

The release tests pin the expected SHA-256 of `smart-frame-sorter.exe`. If an intentional source or toolchain change produces a new release binary, update the expected hash in `release_test.go` as part of the reviewed release change.

## Run tests

```powershell
$env:GOPROXY = "off"
go test ./...
```

The suite covers filename parsing, image grouping, timing-tolerant video grouping, corrupt and short videos, byte-identical copying, output separation, dry runs, overwrite protection, FFmpeg configuration, licenses, the SBOM, and pinned artifact hashes.

Before submitting changes, format and retest:

```powershell
gofmt -w src\*.go
go test ./...
```

Review `git diff` after formatting so unrelated work is not included.

## Repository layout

| Path | Purpose |
| --- | --- |
| `src/main.go` | Image fingerprints, distance calculations, clustering, naming, and filename parsing. |
| `src/app.go` | CLI options and top-level image/video workflow. |
| `src/video.go` | FFmpeg discovery, probing, frame extraction, and video fingerprints. |
| `src/output.go` | Staging, copying, manifests, galleries, and rollback. |
| `src/main_test.go` | Functional and safety tests. |
| `src/release_test.go` | License, SBOM, FFmpeg, source, and binary compliance checks. |
| `third_party/x_image` | Vendored image codecs and their license. |
| `ffmpeg` | Bundled Windows tools, build metadata, and license files. |
| `scripts` | FFmpeg reproduction and release packaging scripts. |
| `release/sources` | Exact corresponding FFmpeg source and generated project source archives. |
| `Docs` | User, technical, development, and release documentation. |

## Rebuild FFmpeg

The custom FFmpeg build requires Linux or WSL with MinGW-w64, Make, NASM, `tar`, and `sha256sum`. From that environment:

```bash
./scripts/build-ffmpeg-windows.sh
```

The script reads `release/sources/ffmpeg-8.1.2.tar.xz`, verifies SHA-256 `464beb5e7bf0c311e68b45ae2f04e9cc2af88851abb4082231742a74d97b524c`, and writes `ffmpeg/ffmpeg.exe` and `ffmpeg/ffprobe.exe`.

If rebuilt executables differ, update `ffmpeg/BUILD-INFO.txt`, `scripts/ffmpeg-build-info.txt`, `src/release_test.go`, the SBOM if applicable, and release checksums. Confirm the license and protocol configuration again with the complete test suite.

## Design constraints

Changes should preserve these product guarantees:

- fully offline processing;
- source-folder-only discovery;
- no modification or recompression of originals;
- comparison only within one person and media type;
- distinct atomic-style image and video outputs;
- no overwriting of non-empty outputs; and
- complete corresponding source and license material for distributed binaries.

See [CONTRIBUTING.md](../CONTRIBUTING.md) before opening a pull request.
