@echo off
rem Copyright (C) 2026 SmartFrameSorter contributors
rem SPDX-License-Identifier: AGPL-3.0-or-later
setlocal
cd /d "%~dp0"
echo Local Image and MP4 Sorter
echo ==========================
echo.
"%~dp0smart-frame-sorter.exe" -source "%~dp0" -image-output "output-images" -video-output "output-videos"
echo.
if errorlevel 1 (
  echo Nothing in the original folder was changed.
) else (
  echo Open output-images\groups.html to review images.
  echo Open output-videos\groups.html to review videos.
)
echo.
pause
