#!/usr/bin/env bash
# Copyright (C) 2026 SmartFrameSorter contributors
# SPDX-License-Identifier: AGPL-3.0-or-later

set -euo pipefail

FFMPEG_VERSION="8.1.2"
FFMPEG_COMMIT="38b88335f9"
FFMPEG_ARCHIVE_SHA256="464beb5e7bf0c311e68b45ae2f04e9cc2af88851abb4082231742a74d97b524c"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SOURCE_ARCHIVE="${1:-${PROJECT_DIR}/release/sources/ffmpeg-${FFMPEG_VERSION}.tar.xz}"
OUTPUT_DIR="${2:-${PROJECT_DIR}/ffmpeg}"

if [[ ! -f "${SOURCE_ARCHIVE}" ]]; then
  echo "Missing source archive: ${SOURCE_ARCHIVE}" >&2
  exit 1
fi

actual_sha="$(sha256sum "${SOURCE_ARCHIVE}" | awk '{print $1}')"
if [[ "${actual_sha}" != "${FFMPEG_ARCHIVE_SHA256}" ]]; then
  echo "FFmpeg source SHA-256 mismatch: expected ${FFMPEG_ARCHIVE_SHA256}, got ${actual_sha}" >&2
  exit 1
fi

for tool in make x86_64-w64-mingw32-gcc x86_64-w64-mingw32-ar x86_64-w64-mingw32-strip nasm; do
  command -v "${tool}" >/dev/null || { echo "Missing build tool: ${tool}" >&2; exit 1; }
done

build_root="$(mktemp -d)"
trap 'rm -rf "${build_root}"' EXIT
tar -xf "${SOURCE_ARCHIVE}" -C "${build_root}"
source_dir="${build_root}/ffmpeg-${FFMPEG_VERSION}"
install_root="${build_root}/install-root"

cd "${source_dir}"
./configure \
  --prefix=/ffmpeg-smart-frame-sorter \
  --target-os=mingw32 \
  --arch=x86_64 \
  --cross-prefix=x86_64-w64-mingw32- \
  --enable-cross-compile \
  --pkg-config=/bin/false \
  --disable-autodetect \
  --disable-everything \
  --disable-doc \
  --disable-debug \
  --disable-network \
  --disable-iconv \
  --disable-zlib \
  --disable-bzlib \
  --disable-lzma \
  --disable-schannel \
  --enable-gpl \
  --enable-version3 \
  --enable-ffmpeg \
  --enable-ffprobe \
  --enable-avcodec \
  --enable-avformat \
  --enable-avfilter \
  --enable-swscale \
  --enable-protocol=file,pipe \
  --enable-demuxer=mov \
  --enable-decoder=h264,hevc,av1,mpeg4,mpeg2video,mpeg1video,mjpeg,vp8,vp9,prores,ffv1,rawvideo,dnxhd \
  --enable-parser=h264,hevc,av1,mpeg4video,mpegvideo,mjpeg,vp8,vp9 \
  --enable-filter=scale,format \
  --enable-encoder=mjpeg \
  --enable-muxer=image2 \
  --extra-cflags=-O2 \
  --extra-ldflags=-static

make -j"$(nproc)"
make install DESTDIR="${install_root}"
binary_dir="${install_root}/ffmpeg-smart-frame-sorter/bin"
x86_64-w64-mingw32-strip "${binary_dir}/ffmpeg.exe" "${binary_dir}/ffprobe.exe"

mkdir -p "${OUTPUT_DIR}"
cp "${binary_dir}/ffmpeg.exe" "${OUTPUT_DIR}/ffmpeg.exe"
cp "${binary_dir}/ffprobe.exe" "${OUTPUT_DIR}/ffprobe.exe"
cp "${SCRIPT_DIR}/ffmpeg-build-info.txt" "${OUTPUT_DIR}/BUILD-INFO.txt"

echo "Built ${OUTPUT_DIR}/ffmpeg.exe and ${OUTPUT_DIR}/ffprobe.exe"
