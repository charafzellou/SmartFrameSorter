# Command-line reference

## Syntax

```powershell
.\smart-frame-sorter.exe [options]
```

All paths may be absolute or relative. Relative output paths are resolved inside the source folder.

## Options

| Option | Default | Description |
| --- | --- | --- |
| `-source <folder>` | `.` | Folder containing original images and MP4 videos. Only its direct children are scanned. |
| `-image-output <folder>` | `output-images` | Destination for copied and renamed images. |
| `-video-output <folder>` | `output-videos` | Destination for copied and renamed MP4 videos. |
| `-ffmpeg-dir <folder>` | Auto-detected | Folder containing both `ffmpeg.exe` and `ffprobe.exe`. |
| `-delimiter <mode>` | `auto` | Person-name separator: `auto`, `underscore`, or `space`. |
| `-threshold <number>` | `0.18` | Visual grouping threshold between 0 and 1. Smaller is stricter; the recommended range is `0.12` to `0.26`. |
| `-dry-run` | Off | Analyze and print proposed groupings without creating output. |
| `-output <folder>` | Not set | Deprecated alias for `-image-output`. |

`-output` cannot be combined with a non-default `-image-output`.

The source folder, image output, and video output must be three distinct paths. Image and video output paths must also differ from each other.

## Delimiter modes

- `auto` uses whichever appears first in the filename stem: `_` or whitespace.
- `underscore` uses the first `_`, allowing spaces inside the person's name.
- `space` uses the first whitespace character, allowing underscores inside the person's name.

The extension is excluded before the person name is parsed.

## FFmpeg lookup order

When MP4 candidates exist, SmartFrameSorter looks for both tools in this order:

1. The directory supplied with `-ffmpeg-dir`.
2. An `ffmpeg` folder beside `smart-frame-sorter.exe`.
3. The folder containing `smart-frame-sorter.exe`.
4. An `ffmpeg` folder under the current working directory.
5. `ffmpeg.exe` and `ffprobe.exe` available on `PATH`.

Images can be processed without FFmpeg when there are no MP4 candidates.

## Examples

```powershell
# Sort another source directory
.\smart-frame-sorter.exe -source D:\Media\ToSort

# Preview stricter matching
.\smart-frame-sorter.exe -source D:\Media\ToSort -threshold 0.15 -dry-run

# Preserve a full name containing spaces
.\smart-frame-sorter.exe -delimiter underscore

# Write outputs outside the source folder
.\smart-frame-sorter.exe `
  -image-output D:\Sorted\Images `
  -video-output D:\Sorted\Videos
```

