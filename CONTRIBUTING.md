# Contributing to SmartFrameSorter

Thank you for helping improve SmartFrameSorter.

## Before opening a change

- Search existing issues and pull requests to avoid duplicate work.
- For a substantial behavior or architecture change, open an issue first so the scope and safety implications can be discussed.
- Report vulnerabilities through the private process in [SECURITY.md](SECURITY.md), not a public issue.

## Development workflow

1. Fork the repository and create a focused branch.
2. Keep the sorter fully offline and preserve the safety guarantees documented in [Docs/DEVELOPMENT.md](Docs/DEVELOPMENT.md).
3. Add or update tests for behavior changes.
4. Format Go code with `gofmt -w .\src`.
5. Run `go test ./...` from the repository root on Windows x64.
6. Update the README and `Docs/` when user-visible behavior changes.
7. Open a pull request explaining the problem, the approach, and how the change was verified.

Keep pull requests small enough to review. Do not include personal media, generated output directories, credentials, unrelated formatting, or unreviewed binary replacements.

## Binary and dependency changes

Changes to the Go toolchain, vendored code, FFmpeg, or release binaries must include updated provenance, licenses, SBOM data, reproducible-build metadata, hashes, and compliance tests as applicable. Explain why the binary or dependency change is necessary.

## Licensing

By submitting a contribution, you agree that it is licensed under `AGPL-3.0-or-later`, the same license as SmartFrameSorter's original code. You retain copyright in your contribution; no copyright assignment is required.

Do not submit code, media, documentation, or other material unless you have the right to license it for inclusion in this project.
