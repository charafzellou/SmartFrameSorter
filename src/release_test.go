// Copyright (C) 2026 SmartFrameSorter contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseComplianceArtifacts(t *testing.T) {
	root := repositoryRoot(t)
	required := []string{
		"LICENSE", "THIRD_PARTY_NOTICES.md", "sbom.spdx.json",
		filepath.Join("third_party", "x_image", "LICENSE"),
		filepath.Join("ffmpeg", "ffmpeg.exe"), filepath.Join("ffmpeg", "ffprobe.exe"),
		filepath.Join("ffmpeg", "COPYING.GPLv3"), filepath.Join("ffmpeg", "LICENSE.md"),
		filepath.Join("ffmpeg", "BUILD-INFO.txt"),
		filepath.Join("release", "sources", "ffmpeg-8.1.2.tar.xz"),
		filepath.Join("scripts", "build-ffmpeg-windows.sh"), filepath.Join("scripts", "package-release.ps1"),
	}
	for _, path := range required {
		if info, err := os.Stat(filepath.Join(root, path)); err != nil || !info.Mode().IsRegular() {
			t.Errorf("required release artifact is missing or not a regular file: %s (%v)", path, err)
		}
	}

	license := readTestFile(t, filepath.Join(root, "LICENSE"))
	if len(license) < 30_000 || !strings.Contains(license, "GNU AFFERO GENERAL PUBLIC LICENSE") || !strings.Contains(license, "Version 3, 19 November 2007") {
		t.Error("LICENSE is not the complete GNU AGPLv3 text")
	}
	notices := readTestFile(t, filepath.Join(root, "THIRD_PARTY_NOTICES.md"))
	for _, marker := range []string{"Go standard library", "golang.org/x/image", "FFmpeg", "GPL-3.0-or-later", "BSD-style"} {
		if !strings.Contains(notices, marker) {
			t.Errorf("THIRD_PARTY_NOTICES.md is missing %q", marker)
		}
	}
	sbom := []byte(readTestFile(t, filepath.Join(root, "sbom.spdx.json")))
	if !json.Valid(sbom) {
		t.Fatal("sbom.spdx.json is invalid JSON")
	}
	for _, marker := range [][]byte{[]byte("SPDX-2.3"), []byte("AGPL-3.0-or-later"), []byte("GPL-3.0-or-later"), []byte("BSD-3-Clause")} {
		if !bytes.Contains(sbom, marker) {
			t.Errorf("SBOM is missing %q", marker)
		}
	}

	assertSHA256(t, filepath.Join(root, "release", "sources", "ffmpeg-8.1.2.tar.xz"), "464beb5e7bf0c311e68b45ae2f04e9cc2af88851abb4082231742a74d97b524c")
	assertSHA256(t, filepath.Join(root, "smart-frame-sorter.exe"), "3f948c8cd0e87e0546074dfe16c6322a37d081745c5d2839690fb7789df97e57")
	assertSHA256(t, filepath.Join(root, "ffmpeg", "ffmpeg.exe"), "b2cca40cc829dd19b5fcfe38b0377fd2ea47f3ca9e1ebcb1d1df74b5b041f0dc")
	assertSHA256(t, filepath.Join(root, "ffmpeg", "ffprobe.exe"), "d26906fb99643693c1cc02cd62f982b1f4cf01ea89c965dd7eb2f1cdb5e20d16")
}

func TestBundledFFmpegIsPinnedGPLAndOffline(t *testing.T) {
	ffmpegPath := filepath.Join(repositoryRoot(t), "ffmpeg", "ffmpeg.exe")
	version, err := exec.Command(ffmpegPath, "-version").CombinedOutput()
	if err != nil {
		t.Fatalf("bundled FFmpeg version check failed: %v: %s", err, version)
	}
	versionText := string(version)
	for _, marker := range []string{"ffmpeg version 8.1.2", "--disable-network", "--disable-autodetect", "--disable-everything", "--enable-gpl", "--enable-version3", "--enable-protocol='file,pipe'"} {
		if !strings.Contains(versionText, marker) {
			t.Errorf("bundled FFmpeg configuration is missing %q", marker)
		}
	}
	protocols, err := exec.Command(ffmpegPath, "-protocols").CombinedOutput()
	if err != nil {
		t.Fatalf("bundled FFmpeg protocol check failed: %v: %s", err, protocols)
	}
	for _, forbidden := range []string{"http", "https", "tcp", "udp", "rtmp", "ftp", "srt"} {
		if containsTrimmedLine(strings.ToLower(string(protocols)), forbidden) {
			t.Errorf("bundled FFmpeg unexpectedly enables network protocol %q", forbidden)
		}
	}
	for _, requiredProtocol := range []string{"file", "pipe"} {
		if !containsTrimmedLine(string(protocols), requiredProtocol) {
			t.Errorf("bundled FFmpeg is missing required protocol %q", requiredProtocol)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Clean(working)) == "src" {
		return filepath.Dir(working)
	}
	return working
}

func assertSHA256(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		t.Errorf("SHA-256 mismatch for %s: got %s, want %s", path, actual, expected)
	}
}

func containsTrimmedLine(text, wanted string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == wanted {
			return true
		}
	}
	return false
}
