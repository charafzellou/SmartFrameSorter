# User guide

## Requirements

- Windows x64.
- A ready-to-run SmartFrameSorter bundle containing `smart-frame-sorter.exe`, `Sort Images.bat`, and the `ffmpeg` folder.
- Enough free disk space for the copied media and locally generated video posters.

SmartFrameSorter is portable: it does not require installation, an account, or an internet connection.

## Prepare filenames

SmartFrameSorter derives the person from the filename prefix before the first underscore or whitespace character. It compares a person's files only with other files belonging to that person.

| Filename | Person detected in automatic mode |
| --- | --- |
| `alice_beach_01.jpg` | `alice` |
| `Bob studio 02.png` | `Bob` |
| `Jane Doe_beach_03.mp4` | `Jane` |

For a person's name that contains spaces, use an underscore as the separator and run with `-delimiter underscore`. For example, `Jane Doe_beach_03.mp4` is then assigned to `Jane Doe`.

For a person's name that contains underscores, use whitespace as the separator and run with `-delimiter space`.

Files without the selected separator are skipped. Person matching is case-insensitive, while the first spelling encountered is retained for display and output names.

## Run the sorter

The simplest workflow is:

1. Put the complete SmartFrameSorter bundle in the folder containing the media.
2. Double-click `Sort Images.bat`.
3. Wait for both the image and video summaries.
4. Open `output-images\groups.html` and `output-videos\groups.html` in a browser to review the results.

The batch file keeps the source directory as the working folder and pauses when the run finishes so errors remain visible.

For command-line use, open PowerShell in the media folder:

```powershell
.\smart-frame-sorter.exe
```

Use `-dry-run` before a large sort to preview the proposed names without creating output folders:

```powershell
.\smart-frame-sorter.exe -dry-run
```

## Understand the output

Images and videos are numbered independently and written to separate folders. A copied filename has the form `person_X_Y.ext`:

- `person` is a filesystem-safe form of the detected person's name.
- `X` is the visual group number for that person and media type.
- `Y` is the variation number inside that group.
- `ext` preserves a suitable media extension. MP4 files remain MP4 files.

Each output folder also contains:

- `manifest.csv`, mapping original filenames to copied filenames and group numbers.
- `groups.html`, a local visual gallery that does not load remote content.
- `thumbnails`, containing local video poster frames when videos were processed.
- `skipped.txt`, when one or more candidates could not be processed.

The source files remain in place and are never renamed or recompressed.

## Tune visual grouping

The default threshold is `0.18`. Lower values are stricter and tend to produce more groups; higher values are broader and tend to merge more files.

```powershell
.\smart-frame-sorter.exe -threshold 0.15
.\smart-frame-sorter.exe -threshold 0.22
```

The usual useful range is `0.12` to `0.26`. Start with a dry run and change the value in small steps.

The same threshold is applied to images and videos, but their group numbering remains independent.

## Output safety

SmartFrameSorter checks both output paths before writing. It refuses to run if either output directory already contains files or folders. Empty existing output directories are allowed.

New output is first written to sibling directories ending in `.building`. SmartFrameSorter removes its staging data if preparation fails and rolls back the first output if the second cannot be finalized.

To run again, move, rename, or intentionally remove the previous output folders first. SmartFrameSorter never clears them automatically.

## Troubleshooting

### No supported media was found

Only files directly inside the source directory are read. Move supported media out of subfolders or provide another folder with `-source`.

### Files were skipped because no person-name separator was found

Add an underscore or a space after the person's name, or choose the matching `-delimiter` mode. See the [command-line reference](CLI_REFERENCE.md).

### FFmpeg is missing

Keep the bundled `ffmpeg` folder beside `smart-frame-sorter.exe`, or pass a directory containing both `ffmpeg.exe` and `ffprobe.exe`:

```powershell
.\smart-frame-sorter.exe -ffmpeg-dir D:\Tools\ffmpeg
```

FFmpeg is required only when MP4 candidates are present.

### An output folder is not empty

Choose unused output paths or move the existing results. The refusal is deliberate overwrite protection.

### Similar files split into too many groups

Increase the threshold slightly. If different scenes are merged, decrease it. SmartFrameSorter compares visual structure and color; it does not understand poses, people, or backgrounds semantically.

