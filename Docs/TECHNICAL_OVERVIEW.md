# Technical overview

## Processing pipeline

SmartFrameSorter performs one local pass over the direct child files of the source directory:

1. Split candidates into supported images and `.mp4` videos.
2. Parse the person name from each filename.
3. Decode a compact visual representation in memory.
4. Compare only files with the same person and media type.
5. Cluster files using average-linkage distance and the selected threshold.
6. Assign deterministic group and variation numbers.
7. Build both outputs in staging directories.
8. Finalize both outputs, or roll back if finalization fails.

Images and videos share the clustering stage but have separate candidate sets, group numbering, output directories, manifests, and galleries.

## Image fingerprints

Each supported image is decoded locally and resized to a small normalized representation. The fingerprint combines:

- normalized grayscale structure;
- an edge-orientation histogram;
- brightness information;
- an HSV color histogram;
- a spatial color grid; and
- a perceptual hash.

JPEG EXIF orientation is applied while sampling. Animated GIFs use the first decoded frame. Images with invalid dimensions or more than 250 megapixels are rejected before full processing.

## Video fingerprints

SmartFrameSorter uses `ffprobe` to determine duration and asks `ffmpeg` for frames near 10%, 50%, and 90%. Each decoded frame is converted to the same kind of visual fingerprint used for images.

Video distance uses symmetric best-frame matching rather than requiring the sampled timestamps to align exactly. This makes the comparison more tolerant of short trims and timing shifts. If none of the three samples can be decoded, SmartFrameSorter attempts a frame at the beginning before skipping the video.

Each FFmpeg or FFprobe process has a 45-second timeout. Decoded frame output is bounded, and the invoked protocol whitelist is restricted to `file` and `pipe`.

## Clustering and naming

Distances are calculated only within one person's set. Clusters are combined using average linkage while their distance remains within the configured threshold. Group order and variation order are deterministic for the same inputs and program version.

The resulting name is `person_X_Y.ext`. Unsafe filename characters in the person component are replaced so the output is valid on Windows.

The system performs visual similarity matching. It does not identify people from pixels and does not semantically recognize a pose, location, event, or background. Results should be reviewed in `groups.html`.

## Data handling and privacy

- The Go program contains no network client and performs no network requests.
- Media bytes, names, fingerprints, and generated posters remain on the local machine.
- The HTML galleries contain local relative paths and no remote resources.
- The custom FFmpeg build is configured with networking disabled and only `file` and `pipe` protocols enabled.
- FFmpeg is launched as a separate process and is not linked into SmartFrameSorter.

## Filesystem safety

The originals are opened for reading and copied to new files. SmartFrameSorter does not rename, move, edit, recompress, or delete source media.

Before writing, the program verifies that the two output paths are distinct from the source and from each other. A non-empty output stops the entire run. Each output is built at `<output>.building`, then renamed into place. Cleanup and rollback prevent a successful-looking partial result when either side fails.

## Reproducibility and supply chain

The repository vendors the required `golang.org/x/image` v0.34.0 subset and replaces the module path locally, so the Go build does not require package downloads.

FFmpeg 8.1.2 is built from the pinned commit `38b88335f9`. The reproduction script verifies the exact source archive SHA-256 before building. Build configuration, executable hashes, license text, third-party notices, and an SPDX 2.3 SBOM are stored in the repository.

See [development and testing](DEVELOPMENT.md) and [the release process](RELEASING.md) for verification commands.
