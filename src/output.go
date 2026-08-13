// Copyright (C) 2026 SmartFrameSorter contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ensureOutputAvailable(output, flagName string) error {
	info, err := os.Stat(output)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("output path already exists and is not a folder: %s", output)
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("output folder is not empty: %s\nMove or delete it, or choose another with %s", output, flagName)
	}
	return nil
}

func writeBothOutputs(imageOutput string, images mediaResult, videoOutput string, videos mediaResult, threshold float64) (err error) {
	imageStage, imageWasEmpty, err := buildOutputStage(imageOutput, images, threshold)
	if err != nil {
		return err
	}
	imageCommitted := false
	defer func() {
		if !imageCommitted {
			_ = os.RemoveAll(imageStage)
		}
	}()

	videoStage, videoWasEmpty, err := buildOutputStage(videoOutput, videos, threshold)
	if err != nil {
		return err
	}
	videoCommitted := false
	defer func() {
		if !videoCommitted {
			_ = os.RemoveAll(videoStage)
		}
	}()

	if err = commitStage(imageStage, imageOutput, imageWasEmpty); err != nil {
		return err
	}
	imageCommitted = true
	if err = commitStage(videoStage, videoOutput, videoWasEmpty); err != nil {
		rollbackErr := os.Rename(imageOutput, imageStage)
		imageCommitted = false
		if imageWasEmpty {
			_ = os.Mkdir(imageOutput, 0755)
		}
		if rollbackErr != nil {
			return fmt.Errorf("finalize video output: %v; rollback image output: %v", err, rollbackErr)
		}
		return fmt.Errorf("finalize video output: %w", err)
	}
	videoCommitted = true
	return nil
}

func buildOutputStage(output string, result mediaResult, threshold float64) (stage string, outputWasEmpty bool, err error) {
	stage = output + ".building"
	if _, statErr := os.Stat(stage); statErr == nil {
		return "", false, fmt.Errorf("temporary folder already exists: %s", stage)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", false, statErr
	}
	if info, statErr := os.Stat(output); statErr == nil && info.IsDir() {
		outputWasEmpty = true
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return "", false, err
	}
	if err = os.Mkdir(stage, 0755); err != nil {
		return "", false, err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(stage)
		}
	}()

	if result.kind == videoMedia && len(result.items) > 0 {
		if err = os.Mkdir(filepath.Join(stage, "thumbnails"), 0755); err != nil {
			return "", false, err
		}
	}
	for _, item := range result.items {
		if err = copyFile(item.sourcePath, filepath.Join(stage, item.newName)); err != nil {
			return "", false, err
		}
		if result.kind == videoMedia && len(item.posterPNG) > 0 {
			posterPath := filepath.Join(stage, "thumbnails", posterFilename(item))
			if err = os.WriteFile(posterPath, item.posterPNG, 0644); err != nil {
				return "", false, err
			}
		}
	}
	if err = writeManifest(filepath.Join(stage, "manifest.csv"), result); err != nil {
		return "", false, err
	}
	if err = writeGallery(filepath.Join(stage, "groups.html"), result, threshold); err != nil {
		return "", false, err
	}
	if len(result.skipped) > 0 {
		if err = os.WriteFile(filepath.Join(stage, "skipped.txt"), []byte(strings.Join(result.skipped, "\r\n")+"\r\n"), 0644); err != nil {
			return "", false, err
		}
	}
	complete = true
	return stage, outputWasEmpty, nil
}

func commitStage(stage, output string, outputWasEmpty bool) error {
	if outputWasEmpty {
		if err := os.Remove(output); err != nil {
			return fmt.Errorf("remove empty output folder: %w", err)
		}
	}
	if err := os.Rename(stage, output); err != nil {
		if outputWasEmpty {
			_ = os.Mkdir(output, 0755)
		}
		return err
	}
	return nil
}

func copyFile(source, destination string) (err error) {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		closeErr := out.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
		if !ok {
			_ = os.Remove(destination)
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	ok = true
	return nil
}

func writeManifest(path string, result mediaResult) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()
	w := csv.NewWriter(f)
	if err = w.Write([]string{"original_file", "media_type", "person", "group_X", "variation_Y", "new_file", "sample_frames", "distance_to_group_center"}); err != nil {
		return err
	}
	for _, item := range result.items {
		row := []string{item.originalName, string(item.kind), item.person, strconv.Itoa(item.group), strconv.Itoa(item.variation), item.newName, strconv.Itoa(len(item.features)), fmt.Sprintf("%.4f", item.distanceToCenter)}
		if err = w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func writeGallery(path string, result mediaResult, threshold float64) error {
	var groups []galleryGroup
	for _, item := range result.items {
		if len(groups) == 0 || groups[len(groups)-1].Person != item.person || groups[len(groups)-1].Number != item.group {
			groups = append(groups, galleryGroup{Person: item.person, Number: item.group})
		}
		gallery := galleryItem{
			NewName: item.newName, OriginalName: item.originalName, MediaURL: url.PathEscape(item.newName),
			Distance: fmt.Sprintf("%.3f", item.distanceToCenter), IsVideo: result.kind == videoMedia,
		}
		if gallery.IsVideo {
			gallery.PosterURL = "thumbnails/" + url.PathEscape(posterFilename(item))
		}
		groups[len(groups)-1].Items = append(groups[len(groups)-1].Items, gallery)
	}
	label := "Images"
	if result.kind == videoMedia {
		label = "Videos"
	}
	data := struct {
		Groups    []galleryGroup
		Threshold string
		Label     string
	}{groups, fmt.Sprintf("%.2f", threshold), label}
	t, err := template.New("gallery").Parse(galleryTemplate)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func posterFilename(item *photo) string {
	return strings.TrimSuffix(item.newName, filepath.Ext(item.newName)) + ".jpeg"
}

const galleryTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Label}} — SmartFrameSorter groups</title><style>
:root{color-scheme:dark;background:#101114;color:#f3f4f6;font:15px system-ui,sans-serif}body{max-width:1400px;margin:auto;padding:28px}h1{margin:0 0 6px}h2{margin:32px 0 12px;padding-top:18px;border-top:1px solid #343740}.note{color:#abb0bb}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(180px,1fr));gap:14px}.card{background:#1b1d22;border:1px solid #30333a;border-radius:10px;padding:9px;overflow:hidden}.card img,.card video{width:100%;height:220px;object-fit:contain;background:#090a0c;border-radius:6px}.new{font-weight:650;margin-top:8px;overflow-wrap:anywhere}.old,.distance{font-size:12px;color:#abb0bb;overflow-wrap:anywhere}.distance{margin-top:4px}</style></head>
<body><h1>{{.Label}} — proposed visual groups</h1><div class="note">Completely local report · grouping threshold {{.Threshold}} · X is the group, Y is the variation.</div>
{{if not .Groups}}<p>No valid {{.Label}} were found in this run.</p>{{end}}
{{range .Groups}}<h2>{{.Person}} — group {{.Number}}</h2><div class="grid">{{range .Items}}<div class="card">{{if .IsVideo}}<video controls preload="none" poster="{{.PosterURL}}"><source src="{{.MediaURL}}" type="video/mp4"></video>{{else}}<a href="{{.MediaURL}}"><img loading="lazy" src="{{.MediaURL}}" alt="{{.NewName}}"></a>{{end}}<div class="new">{{.NewName}}</div><div class="old">was {{.OriginalName}}</div><div class="distance">distance from group center: {{.Distance}}</div></div>{{end}}</div>{{end}}
</body></html>`

func printPlan(result mediaResult, threshold float64) {
	label := string(result.kind) + "s"
	fmt.Printf("\nDry run — %s at threshold %.2f: %d valid files, %d visual groups.\n\n", label, threshold, len(result.items), result.groups)
	for _, item := range result.items {
		fmt.Printf("%-36s -> %s\n", item.originalName, item.newName)
	}
	if len(result.skipped) > 0 {
		fmt.Printf("\nSkipped %s:\n", label)
		for _, line := range result.skipped {
			fmt.Printf("  %s\n", line)
		}
	}
}
