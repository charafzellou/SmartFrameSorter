// Copyright (C) 2026 SmartFrameSorter contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	videoToolTimeout = 45 * time.Second
	maxFrameBytes    = 12 << 20
)

type ffmpegTools struct {
	ffmpeg  string
	ffprobe string
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("decoder output exceeded %d bytes", b.limit)
	}
	return b.Buffer.Write(p)
}

func findFFmpegTools(explicitDir string) (ffmpegTools, error) {
	var dirs []string
	if explicitDir != "" {
		absolute, err := filepath.Abs(explicitDir)
		if err != nil {
			return ffmpegTools{}, err
		}
		dirs = append(dirs, absolute)
	}
	if executable, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(executable), "ffmpeg"), filepath.Dir(executable))
	}
	if working, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(working, "ffmpeg"))
	}

	seen := make(map[string]bool)
	for _, dir := range dirs {
		key := strings.ToLower(filepath.Clean(dir))
		if seen[key] {
			continue
		}
		seen[key] = true
		tools := ffmpegTools{ffmpeg: filepath.Join(dir, "ffmpeg.exe"), ffprobe: filepath.Join(dir, "ffprobe.exe")}
		if regularFile(tools.ffmpeg) && regularFile(tools.ffprobe) {
			return tools, validateFFmpegTools(tools)
		}
	}
	ffmpegPath, ffmpegErr := exec.LookPath("ffmpeg.exe")
	ffprobePath, ffprobeErr := exec.LookPath("ffprobe.exe")
	if ffmpegErr == nil && ffprobeErr == nil {
		tools := ffmpegTools{ffmpeg: ffmpegPath, ffprobe: ffprobePath}
		return tools, validateFFmpegTools(tools)
	}
	return ffmpegTools{}, errors.New("MP4 files were found, but ffmpeg.exe and ffprobe.exe are missing; restore the bundled ffmpeg folder or use -ffmpeg-dir")
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func validateFFmpegTools(tools ffmpegTools) error {
	for name, path := range map[string]string{"ffmpeg": tools.ffmpeg, "ffprobe": tools.ffprobe} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, path, "-version")
		output, err := cmd.Output()
		cancel()
		if err != nil {
			return fmt.Errorf("run %s at %s: %w", name, path, err)
		}
		if !bytes.Contains(bytes.ToLower(output), []byte(name+" version")) {
			return fmt.Errorf("unexpected %s executable at %s", name, path)
		}
	}
	return nil
}

func loadVideos(source string, candidates []os.DirEntry, delimiter string, tools ffmpegTools) mediaResult {
	result := mediaResult{kind: videoMedia, sets: make(map[string]*personSet), candidateNo: len(candidates)}
	for i, entry := range candidates {
		person, ok := personFromName(entry.Name(), delimiter)
		if !ok {
			result.skipped = append(result.skipped, fmt.Sprintf("%s -- no person-name separator found", entry.Name()))
			continue
		}
		fmt.Printf("[video %d/%d] Sampling frames: %s\n", i+1, len(candidates), entry.Name())
		path := filepath.Join(source, entry.Name())
		features, poster, err := fingerprintVideo(path, tools)
		if err != nil {
			result.skipped = append(result.skipped, fmt.Sprintf("%s -- %v", entry.Name(), err))
			continue
		}
		addItem(&result, &photo{sourcePath: path, originalName: entry.Name(), person: person, format: "mp4", features: features, kind: videoMedia, posterPNG: poster})
	}
	return result
}

func fingerprintVideo(path string, tools ffmpegTools) ([]fingerprint, []byte, error) {
	duration, err := probeDuration(path, tools.ffprobe)
	if err != nil {
		return nil, nil, err
	}
	timestamps := []float64{duration * 0.10, duration * 0.50, duration * 0.90}
	var features []fingerprint
	var poster []byte
	var failures []string
	for i, timestamp := range timestamps {
		frame, extractErr := extractFrame(path, timestamp, tools.ffmpeg)
		if extractErr != nil {
			failures = append(failures, fmt.Sprintf("%.3fs: %v", timestamp, extractErr))
			continue
		}
		feature, decodeErr := fingerprintFrame(frame)
		if decodeErr != nil {
			failures = append(failures, fmt.Sprintf("%.3fs: %v", timestamp, decodeErr))
			continue
		}
		features = append(features, feature)
		if poster == nil || i == 1 {
			poster = append([]byte(nil), frame...)
		}
	}
	if len(features) == 0 {
		frame, fallbackErr := extractFrame(path, 0, tools.ffmpeg)
		if fallbackErr == nil {
			if feature, decodeErr := fingerprintFrame(frame); decodeErr == nil {
				return []fingerprint{feature}, frame, nil
			}
		}
		return nil, nil, fmt.Errorf("could not decode a video frame (%s)", strings.Join(failures, "; "))
	}
	return features, poster, nil
}

func probeDuration(path, ffprobePath string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), videoToolTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error", "-protocol_whitelist", "file,pipe",
		"-select_streams", "v:0", "-show_entries", "stream=duration:format=duration", "-of", "json", path,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return 0, fmt.Errorf("ffprobe timed out after %s", videoToolTimeout)
	}
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %s", conciseToolError(err, stderr.String()))
	}
	var response struct {
		Streams []struct {
			Duration string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return 0, fmt.Errorf("read ffprobe duration: %w", err)
	}
	values := []string{response.Format.Duration}
	for _, stream := range response.Streams {
		values = append(values, stream.Duration)
	}
	duration := 0.0
	for _, raw := range values {
		value, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr == nil && !math.IsNaN(value) && !math.IsInf(value, 0) && value > duration {
			duration = value
		}
	}
	if duration <= 0 {
		return 0, errors.New("video duration is missing or zero")
	}
	return duration, nil
}

func extractFrame(path string, timestamp float64, ffmpegPath string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), videoToolTimeout)
	defer cancel()
	stdout := &limitedBuffer{limit: maxFrameBytes}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-protocol_whitelist", "file,pipe", "-ss", fmt.Sprintf("%.6f", math.Max(0, timestamp)), "-i", path,
		"-map", "0:v:0", "-frames:v", "1", "-an", "-sn", "-dn",
		"-vf", "scale=640:640:force_original_aspect_ratio=decrease",
		"-q:v", "3", "-f", "image2", "-c:v", "mjpeg", "pipe:1",
	)
	cmd.Stdout = stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("ffmpeg timed out after %s", videoToolTimeout)
	}
	if err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %s", conciseToolError(err, stderr.String()))
	}
	if stdout.Len() == 0 {
		return nil, errors.New("ffmpeg returned no frame")
	}
	return stdout.Bytes(), nil
}

func fingerprintFrame(data []byte) (fingerprint, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fingerprint{}, fmt.Errorf("decode extracted frame: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxImagePixels {
		return fingerprint{}, errors.New("extracted frame has invalid dimensions")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fingerprint{}, fmt.Errorf("decode extracted frame: %w", err)
	}
	return buildFingerprint(makeThumbnail(img, 1, thumbnailSize), thumbnailSize), nil
}

func conciseToolError(err error, stderr string) string {
	message := strings.TrimSpace(stderr)
	if message == "" {
		message = err.Error()
	}
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 400 {
		message = message[:400] + "…"
	}
	return message
}
