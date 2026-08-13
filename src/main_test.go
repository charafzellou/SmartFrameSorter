// Copyright (C) 2026 SmartFrameSorter contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPersonFromName(t *testing.T) {
	tests := []struct {
		file, mode, want string
		ok               bool
	}{
		{"alice_beach-1.jpeg", "auto", "alice", true},
		{"bob house edit.png", "auto", "bob", true},
		{"Jane Doe_beach.jpg", "underscore", "Jane Doe", true},
		{"Jane_Doe beach.jpg", "space", "Jane_Doe", true},
		{"noseparator.jpg", "auto", "", false},
	}
	for _, tt := range tests {
		got, ok := personFromName(tt.file, tt.mode)
		if got != tt.want || ok != tt.ok {
			t.Errorf("personFromName(%q,%q)=(%q,%v), want (%q,%v)", tt.file, tt.mode, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFingerprintDistance(t *testing.T) {
	a := testPattern(false)
	b := testPattern(false)
	c := testPattern(true)
	fa := buildFingerprint(makeThumbnail(a, 1, thumbnailSize), thumbnailSize)
	fb := buildFingerprint(makeThumbnail(b, 1, thumbnailSize), thumbnailSize)
	fc := buildFingerprint(makeThumbnail(c, 1, thumbnailSize), thumbnailSize)
	if d := fingerprintDistance(fa, fb); d > 1e-9 {
		t.Fatalf("identical images have distance %f", d)
	}
	if d := fingerprintDistance(fa, fc); d < 0.05 {
		t.Fatalf("different images unexpectedly close: %f", d)
	}
}

func TestVideoDistanceUsesTimingTolerantFrameMatching(t *testing.T) {
	red := image.NewUniform(color.RGBA{240, 30, 20, 255})
	blue := image.NewUniform(color.RGBA{20, 50, 240, 255})
	green := image.NewUniform(color.RGBA{20, 220, 80, 255})
	fingerprintFor := func(img image.Image) fingerprint {
		return buildFingerprint(makeThumbnail(img, 1, thumbnailSize), thumbnailSize)
	}
	a := &photo{features: []fingerprint{fingerprintFor(red), fingerprintFor(blue), fingerprintFor(green)}}
	b := &photo{features: []fingerprint{fingerprintFor(green), fingerprintFor(red), fingerprintFor(blue)}}
	if distance := photoDistance(a, b); distance > 1e-9 {
		t.Fatalf("reordered equivalent frames have distance %f", distance)
	}
}

func TestNaturalLess(t *testing.T) {
	if !naturalLess("alice_2.jpg", "alice_10.jpg") {
		t.Fatal("expected numeric natural ordering")
	}
	if naturalLess("alice_10.jpg", "alice_2.jpg") {
		t.Fatal("natural ordering is not antisymmetric")
	}
}

func TestRunCopiesAndGroupsWithoutChangingSources(t *testing.T) {
	dir := t.TempDir()
	inputs := map[string]image.Image{
		"alice_poseA_edit1.png": testPattern(false),
		"alice_poseA_edit2.png": testPattern(false),
		"alice_poseB_edit1.png": testPattern(true),
		"alice_poseB_edit2.png": testPattern(true),
		"bob_poseA_edit1.png":   testPattern(false),
	}
	originalBytes := make(map[string][]byte)
	for name, img := range inputs {
		var data bytes.Buffer
		if err := png.Encode(&data, img); err != nil {
			t.Fatal(err)
		}
		originalBytes[name] = append([]byte(nil), data.Bytes()...)
		if err := os.WriteFile(filepath.Join(dir, name), data.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}

	out := filepath.Join(dir, "result")
	videoOut := filepath.Join(dir, "video-result")
	if err := run(options{source: dir, imageOutput: out, videoOutput: videoOut, delimiter: "auto", threshold: defaultThreshold}); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(out, "manifest.csv"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	for _, expected := range []string{"alice_1_1.png", "alice_1_2.png", "alice_2_1.png", "alice_2_2.png", "bob_1_1.png"} {
		if !strings.Contains(text, expected) {
			t.Errorf("manifest does not contain %q\n%s", expected, text)
		}
	}
	for name, before := range originalBytes {
		after, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("source file %s changed", name)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "groups.html")); err != nil {
		t.Fatalf("gallery was not created: %v", err)
	}
}

func TestRunMixedImagesAndMP4s(t *testing.T) {
	tools, err := findFFmpegTools(testFFmpegDir())
	if err != nil {
		t.Skipf("FFmpeg integration test requires local tools: %v", err)
	}
	dir := t.TempDir()
	patternA, patternB := filepath.Join(dir, "pattern-a.png"), filepath.Join(dir, "pattern-b.png")
	writePNG(t, patternA, testPattern(false))
	writePNG(t, patternB, testPattern(true))
	writePNG(t, filepath.Join(dir, "alice_image.png"), testPattern(false))

	videoSources := map[string]string{
		"alice_sceneA_edit1.mp4": patternA,
		"alice_sceneA_edit2.MP4": patternA,
		"alice_sceneB_edit1.mp4": patternB,
		"bob_sceneA_edit1.mp4":   patternA,
	}
	fixtureFFmpeg := testFixtureFFmpeg()
	if fixtureFFmpeg == "" {
		t.Skip("MP4 fixture generation requires a development FFmpeg build")
	}
	for name, source := range videoSources {
		makeTestMP4(t, fixtureFFmpeg, source, filepath.Join(dir, name))
	}
	shortVideo := filepath.Join(dir, "carol_short.mp4")
	makeTestMP4Duration(t, fixtureFFmpeg, patternA, shortVideo, "0.2")
	videoSources["carol_short.mp4"] = patternA
	corruptPath := filepath.Join(dir, "alice_corrupt.mp4")
	if err := os.WriteFile(corruptPath, []byte("not a video"), 0644); err != nil {
		t.Fatal(err)
	}
	originals := make(map[string][]byte)
	for name := range videoSources {
		originals[name], err = os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
	}

	imageOutput, videoOutput := filepath.Join(dir, "images"), filepath.Join(dir, "videos")
	err = run(options{source: dir, imageOutput: imageOutput, videoOutput: videoOutput, ffmpegDir: filepath.Dir(tools.ffmpeg), delimiter: "auto", threshold: defaultThreshold})
	if err != nil {
		t.Fatal(err)
	}
	imageManifest := readTestFile(t, filepath.Join(imageOutput, "manifest.csv"))
	if !strings.Contains(imageManifest, "alice_1_1.png") || strings.Contains(imageManifest, ".mp4") {
		t.Fatalf("image output was not independent:\n%s", imageManifest)
	}
	videoManifest := readTestFile(t, filepath.Join(videoOutput, "manifest.csv"))
	for _, expected := range []string{"alice_1_1.mp4", "alice_1_2.mp4", "alice_2_1.mp4", "bob_1_1.mp4", "carol_1_1.mp4"} {
		if !strings.Contains(videoManifest, expected) {
			t.Errorf("video manifest does not contain %q\n%s", expected, videoManifest)
		}
	}
	if !strings.Contains(readTestFile(t, filepath.Join(videoOutput, "skipped.txt")), "alice_corrupt.mp4") {
		t.Error("corrupt MP4 was not recorded as skipped")
	}
	gallery := readTestFile(t, filepath.Join(videoOutput, "groups.html"))
	if !strings.Contains(gallery, "<video controls") || !strings.Contains(gallery, "thumbnails/") {
		t.Error("video gallery does not include playable videos and local posters")
	}
	posters, err := filepath.Glob(filepath.Join(videoOutput, "thumbnails", "*.jpeg"))
	if err != nil || len(posters) != 5 {
		t.Fatalf("expected 5 posters, got %d (%v)", len(posters), err)
	}
	for originalName, before := range originals {
		after, readErr := os.ReadFile(filepath.Join(dir, originalName))
		if readErr != nil || !bytes.Equal(before, after) {
			t.Errorf("source video %s changed", originalName)
		}
	}
}

func TestMP4RequiresFFmpegButImagesDoNot(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "alice_image.png"), testPattern(false))
	if err := run(options{source: dir, imageOutput: filepath.Join(dir, "images"), videoOutput: filepath.Join(dir, "videos"), ffmpegDir: filepath.Join(dir, "missing"), delimiter: "auto", threshold: defaultThreshold}); err != nil {
		t.Fatalf("image-only run unexpectedly required FFmpeg: %v", err)
	}
}

func TestDryRunCreatesNoOutputs(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "alice_image.png"), testPattern(false))
	imageOutput, videoOutput := filepath.Join(dir, "images"), filepath.Join(dir, "videos")
	if err := run(options{source: dir, imageOutput: imageOutput, videoOutput: videoOutput, delimiter: "auto", threshold: defaultThreshold, dryRun: true}); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{imageOutput, videoOutput} {
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Errorf("dry run unexpectedly created %s", output)
		}
	}
}

func TestNonEmptyOutputPreventsAnyOutput(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "alice_image.png"), testPattern(false))
	imageOutput, videoOutput := filepath.Join(dir, "images"), filepath.Join(dir, "videos")
	if err := os.Mkdir(imageOutput, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageOutput, "keep.txt"), []byte("user data"), 0644); err != nil {
		t.Fatal(err)
	}
	err := run(options{source: dir, imageOutput: imageOutput, videoOutput: videoOutput, delimiter: "auto", threshold: defaultThreshold})
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected non-empty output error, got %v", err)
	}
	if _, err := os.Stat(videoOutput); !os.IsNotExist(err) {
		t.Error("video output was created despite image output preflight failure")
	}
	if got := readTestFile(t, filepath.Join(imageOutput, "keep.txt")); got != "user data" {
		t.Error("existing output content changed")
	}
}

func TestMissingFFmpegToolsAreRejected(t *testing.T) {
	dir := t.TempDir()
	err := validateFFmpegTools(ffmpegTools{ffmpeg: filepath.Join(dir, "ffmpeg.exe"), ffprobe: filepath.Join(dir, "ffprobe.exe")})
	if err == nil {
		t.Fatal("expected missing FFmpeg tools to be rejected")
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func makeTestMP4(t *testing.T, ffmpegPath, input, output string) {
	t.Helper()
	makeTestMP4Duration(t, ffmpegPath, input, output, "1.2")
}

func makeTestMP4Duration(t *testing.T, ffmpegPath, input, output, duration string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, "-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-loop", "1", "-i", input, "-t", duration, "-r", "8", "-c:v", "mpeg4", "-pix_fmt", "yuv420p", output)
	if outputBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create MP4 fixture: %v: %s", err, outputBytes)
	}
}

func testFFmpegDir() string {
	if value := os.Getenv("SMARTFRAMESORTER_TEST_FFMPEG_DIR"); value != "" {
		return value
	}
	if working, err := os.Getwd(); err == nil {
		root := working
		if filepath.Base(filepath.Clean(root)) == "src" {
			root = filepath.Dir(root)
		}
		if regularFile(filepath.Join(root, "ffmpeg", "ffmpeg.exe")) && regularFile(filepath.Join(root, "ffmpeg", "ffprobe.exe")) {
			return filepath.Join(root, "ffmpeg")
		}
	}
	if regularFile(`C:\ffmpeg\bin\ffmpeg.exe`) && regularFile(`C:\ffmpeg\bin\ffprobe.exe`) {
		return `C:\ffmpeg\bin`
	}
	return ""
}

func testFixtureFFmpeg() string {
	if value := os.Getenv("SMARTFRAMESORTER_TEST_FIXTURE_FFMPEG"); value != "" && regularFile(value) {
		return value
	}
	if regularFile(`C:\ffmpeg\bin\ffmpeg.exe`) {
		return `C:\ffmpeg\bin\ffmpeg.exe`
	}
	return ""
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func testPattern(invert bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			on := (x < 32) != (y < 24)
			if invert {
				on = !on
			}
			if on {
				img.Set(x, y, color.RGBA{230, 50, 30, 255})
			} else {
				img.Set(x, y, color.RGBA{20, 80, 220, 255})
			}
		}
	}
	return img
}
