# Third-Party Notices

SmartFrameSorter's original source code is licensed under `AGPL-3.0-or-later`. The following components are separate third-party works and retain their respective licenses.

## Go standard library

- Project: The Go Programming Language
- Version used for the provided build: Go 1.26.5
- Source: https://go.dev/src/
- License: BSD-style Go license

The Go standard library is statically incorporated into `smart-frame-sorter.exe`.

## golang.org/x/image

- Project: Go supplementary image libraries
- Version: v0.34.0, vendored subset
- Source: https://github.com/golang/image/tree/v0.34.0
- Local source: `third_party/x_image`
- License: BSD-style Go license

The following license text applies to both Go components above:

> Copyright 2009 The Go Authors.
>
> Redistribution and use in source and binary forms, with or without
> modification, are permitted provided that the following conditions are
> met:
>
> * Redistributions of source code must retain the above copyright
> notice, this list of conditions and the following disclaimer.
> * Redistributions in binary form must reproduce the above
> copyright notice, this list of conditions and the following disclaimer
> in the documentation and/or other materials provided with the
> distribution.
> * Neither the name of Google LLC nor the names of its
> contributors may be used to endorse or promote products derived from
> this software without specific prior written permission.
>
> THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
> "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
> LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
> A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
> OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
> SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
> LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
> DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
> THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
> (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
> OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

The same text is preserved at `third_party/x_image/LICENSE`.

## FFmpeg

- Project: FFmpeg
- Version: 8.1.2
- Pinned upstream commit: `38b88335f9`
- Source archive: `release/sources/ffmpeg-8.1.2.tar.xz`
- Source SHA-256: `464beb5e7bf0c311e68b45ae2f04e9cc2af88851abb4082231742a74d97b524c`
- Source: https://ffmpeg.org/releases/ffmpeg-8.1.2.tar.xz
- License for this custom build: `GPL-3.0-or-later`

FFmpeg is not linked into SmartFrameSorter. The separately distributed `ffmpeg.exe` and `ffprobe.exe` programs are invoked through command-line arguments and pipes. They remain under their own GPL terms and are not relicensed as AGPL.

This build uses no external libraries, uses no `--enable-nonfree` components, and was configured with networking disabled. Exact configuration and reproduction instructions are in `ffmpeg/BUILD-INFO.txt` and `scripts/build-ffmpeg-windows.sh`. Its GPL text and upstream component notices are preserved in the `ffmpeg` folder.

Binary releases must provide equivalent access to the exact corresponding FFmpeg source archive and build script alongside the executables.
