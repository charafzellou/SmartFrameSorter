# SmartFrameSorter

SmartFrameSorter is a fully offline Windows tool that groups visually similar images and MP4 videos within each person's files. It copies and renames the results without moving, changing, recompressing, or deleting the originals.

The sorter uses perceptual image fingerprints and sampled video frames. It does **not** use cloud services, facial recognition, or semantic AI.

## Highlights

- Keeps each person's media separate, based on the filename prefix before `_` or a space.
- Processes images and MP4 videos independently.
- Reads only files directly inside the source folder; it never scans subfolders.
- Copies original media byte-for-byte into safely staged output folders.
- Produces a CSV mapping and a completely local HTML review gallery for each media type.
- Ships with a network-disabled FFmpeg build for offline MP4 frame extraction.
- Refuses to overwrite non-empty output folders and rolls back failed finalization.

## Get SmartFrameSorter

Download ready-to-run Windows x64 bundles from the project's [GitHub Releases page](https://github.com/charafzellou/SmartFrameSorter/releases). Keep `smart-frame-sorter.exe`, `Sort Images.bat`, and the bundled `ffmpeg` folder together.

To build from source, see the [development guide](Docs/DEVELOPMENT.md). Go application sources live in [`src/`](src/), with the root `go.work` workspace coordinating the module.

## Quick start

1. Put the SmartFrameSorter bundle in the folder containing the media to sort.
2. Name each media file with the person's name first, followed by `_` or a space. Examples: `alice_beach_edit1.jpeg` and `bob house 04.mp4`.
3. Double-click `Sort Images.bat`.

SmartFrameSorter creates two sibling folders:

- `output-images` for grouped images.
- `output-videos` for grouped MP4 files.

Each output contains `manifest.csv` and a local `groups.html` gallery. Invalid or unusable files are recorded in `skipped.txt`. Copied media uses the name `person_X_Y.ext`, where `X` is the visual group and `Y` is the variation within that group.

## Command-line examples

```powershell
# Preview proposed names without creating output folders
.\smart-frame-sorter.exe -dry-run

# Use stricter matching
.\smart-frame-sorter.exe -threshold 0.15

# Choose output folders
.\smart-frame-sorter.exe -image-output sorted-images -video-output sorted-videos

# Treat only underscores as person-name separators
.\smart-frame-sorter.exe -delimiter underscore

# Use another local FFmpeg installation
.\smart-frame-sorter.exe -ffmpeg-dir C:\path\to\ffmpeg
```

The default threshold is `0.18`; the usual useful range is `0.12` to `0.26`. Smaller values create stricter groups.

## Supported media

Images: JPEG/JPG, PNG, WebP, TIFF, BMP, and GIF. Animated GIFs use the first frame, and JPEG EXIF rotation is respected.

Videos: MP4 containers with common H.264, HEVC, AV1, MPEG-4, MPEG-1/2, MJPEG, VP8/9, ProRes, FFV1, raw-video, or DNxHD video. Unsupported or damaged files are skipped and reported.

## Documentation

- [Documentation index](Docs/README.md)
- [User guide](Docs/USER_GUIDE.md)
- [Command-line reference](Docs/CLI_REFERENCE.md)
- [Technical overview](Docs/TECHNICAL_OVERVIEW.md)
- [Development and testing](Docs/DEVELOPMENT.md)
- [Release process](Docs/RELEASING.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## Offline and safety guarantees

Media, thumbnails, filenames, and fingerprints are never uploaded. The Go program performs no network requests. The bundled FFmpeg 8.1.2 executables were compiled with network support disabled and expose only the `file` and `pipe` protocols.

Originals are copied byte-for-byte. Both outputs are prepared in staging folders, and the run refuses to overwrite either non-empty output. See the [technical overview](Docs/TECHNICAL_OVERVIEW.md) for the comparison and output pipeline.

## License

SmartFrameSorter's original code is licensed under [GNU AGPL-3.0-or-later](LICENSE). Bundled dependencies retain their own licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and the [SPDX SBOM](sbom.spdx.json).

The bundled FFmpeg programs are separate command-line works licensed under `GPL-3.0-or-later`. Codec patent obligations vary by jurisdiction and intended use; commercial distributors should obtain jurisdiction-specific legal advice.
