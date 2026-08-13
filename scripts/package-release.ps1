# Copyright (C) 2026 SmartFrameSorter contributors
# SPDX-License-Identifier: AGPL-3.0-or-later

[CmdletBinding()]
param(
    [string]$Version = "1.1.0"
)

$ErrorActionPreference = "Stop"
if ($Version -cnotmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') {
    throw "Version must use stable SemVer without a leading v: MAJOR.MINOR.PATCH"
}

$ProjectRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$ReleaseRoot = Join-Path $ProjectRoot "release"
$BundleRoot = Join-Path $ReleaseRoot "SmartFrameSorter-$Version-windows-x64"
$BundleArchive = Join-Path $ReleaseRoot "SmartFrameSorter-$Version-windows-x64.zip"
$SourceRoot = Join-Path $ReleaseRoot "sources"
$ProjectSource = Join-Path $SourceRoot "SmartFrameSorter-$Version-source.zip"
$Checksums = Join-Path $ReleaseRoot "SHA256SUMS"

foreach ($output in @($BundleRoot, $BundleArchive, $ProjectSource, $Checksums)) {
    if (Test-Path -LiteralPath $output) {
        throw "Release output already exists: $output"
    }
}

$required = @(
    "smart-frame-sorter.exe", "Sort Images.bat", "README.md", "LICENSE",
    "CONTRIBUTING.md", "SECURITY.md", "Docs", "THIRD_PARTY_NOTICES.md",
    "sbom.spdx.json", "ffmpeg\ffmpeg.exe",
    "ffmpeg\ffprobe.exe", "ffmpeg\BUILD-INFO.txt", "ffmpeg\COPYING.GPLv3",
    "ffmpeg\LICENSE.md", "release\sources\ffmpeg-8.1.2.tar.xz"
)
foreach ($relative in $required) {
    if (-not (Test-Path -LiteralPath (Join-Path $ProjectRoot $relative))) {
        throw "Missing release input: $relative"
    }
}

New-Item -ItemType Directory -Path $BundleRoot -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $BundleRoot "ffmpeg") -Force | Out-Null
New-Item -ItemType Directory -Path $SourceRoot -Force | Out-Null

foreach ($relative in @("smart-frame-sorter.exe", "Sort Images.bat", "README.md", "LICENSE", "CONTRIBUTING.md", "SECURITY.md", "THIRD_PARTY_NOTICES.md", "sbom.spdx.json")) {
    Copy-Item -LiteralPath (Join-Path $ProjectRoot $relative) -Destination (Join-Path $BundleRoot $relative)
}
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Docs") -Destination (Join-Path $BundleRoot "Docs") -Recurse
foreach ($relative in @("ffmpeg.exe", "ffprobe.exe", "BUILD-INFO.txt", "COPYING.GPLv3", "LICENSE.md")) {
    Copy-Item -LiteralPath (Join-Path $ProjectRoot "ffmpeg\$relative") -Destination (Join-Path $BundleRoot "ffmpeg\$relative")
}

$sourceItems = @("go.work", "go.mod", "src", ".gitignore", ".github", "README.md", "CONTRIBUTING.md", "SECURITY.md", "Docs", "LICENSE", "THIRD_PARTY_NOTICES.md", "sbom.spdx.json", "Sort Images.bat", "scripts", "third_party")
Compress-Archive -LiteralPath ($sourceItems | ForEach-Object { Join-Path $ProjectRoot $_ }) -DestinationPath $ProjectSource -CompressionLevel Optimal
Compress-Archive -Path (Join-Path $BundleRoot "*") -DestinationPath $BundleArchive -CompressionLevel Optimal

$checksumEntries = foreach ($file in Get-ChildItem -LiteralPath $BundleRoot -Recurse -File) {
    [pscustomobject]@{
        File = $file
        Name = [IO.Path]::GetRelativePath($ReleaseRoot, $file.FullName).Replace('\', '/')
    }
}
foreach ($path in @($BundleArchive, $ProjectSource, (Join-Path $SourceRoot "ffmpeg-8.1.2.tar.xz"))) {
    $file = Get-Item -LiteralPath $path
    $checksumEntries += [pscustomobject]@{ File = $file; Name = $file.Name }
}
$checksumLines = foreach ($entry in $checksumEntries | Sort-Object Name) {
    $hash = (Get-FileHash -LiteralPath $entry.File.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($entry.Name)"
}
Set-Content -LiteralPath $Checksums -Value $checksumLines -Encoding utf8NoBOM

Write-Host "Release bundle created at $BundleRoot"
Write-Host "Portable archive created at $BundleArchive"
