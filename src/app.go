// Copyright (C) 2026 SmartFrameSorter contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type mediaKind string

const (
	imageMedia mediaKind = "image"
	videoMedia mediaKind = "video"
)

type mediaResult struct {
	kind        mediaKind
	sets        map[string]*personSet
	items       []*photo
	skipped     []string
	groups      int
	candidateNo int
}

func parseFlags() options {
	var o options
	flag.StringVar(&o.source, "source", ".", "folder containing the original images and MP4 videos")
	flag.StringVar(&o.imageOutput, "image-output", "output-images", "folder for copied and renamed images")
	flag.StringVar(&o.videoOutput, "video-output", "output-videos", "folder for copied and renamed MP4 videos")
	flag.StringVar(&o.legacyOutput, "output", "", "deprecated alias for -image-output")
	flag.StringVar(&o.ffmpegDir, "ffmpeg-dir", "", "folder containing ffmpeg.exe and ffprobe.exe")
	flag.StringVar(&o.delimiter, "delimiter", "auto", "person-name separator: auto, underscore, or space")
	flag.Float64Var(&o.threshold, "threshold", defaultThreshold, "visual grouping threshold (smaller = stricter, larger = broader)")
	flag.BoolVar(&o.dryRun, "dry-run", false, "analyze and print both proposed groupings without copying files")
	flag.Parse()
	return o
}

func run(o options) error {
	if o.threshold <= 0 || o.threshold >= 1 {
		return fmt.Errorf("threshold must be between 0 and 1 (recommended range: 0.12 to 0.26)")
	}
	if o.delimiter != "auto" && o.delimiter != "underscore" && o.delimiter != "space" {
		return fmt.Errorf("delimiter must be auto, underscore, or space")
	}
	if o.imageOutput == "" {
		o.imageOutput = "output-images"
	}
	if o.videoOutput == "" {
		o.videoOutput = "output-videos"
	}
	if o.legacyOutput != "" {
		if o.imageOutput != "output-images" {
			return errors.New("-output and -image-output cannot be used together")
		}
		fmt.Fprintln(os.Stderr, "Warning: -output is deprecated; use -image-output instead.")
		o.imageOutput = o.legacyOutput
	}
	if o.delimiter == "" {
		o.delimiter = "auto"
	}

	source, err := filepath.Abs(o.source)
	if err != nil {
		return err
	}
	imageOutput, err := resolveOutput(source, o.imageOutput)
	if err != nil {
		return err
	}
	videoOutput, err := resolveOutput(source, o.videoOutput)
	if err != nil {
		return err
	}
	if samePath(source, imageOutput) || samePath(source, videoOutput) {
		return errors.New("an output folder cannot be the source folder")
	}
	if samePath(imageOutput, videoOutput) {
		return errors.New("image and video outputs must be different folders")
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("read source folder: %w", err)
	}
	imageEntries, videoEntries := splitCandidates(entries)
	if len(imageEntries)+len(videoEntries) == 0 {
		return errors.New("no supported images or MP4 videos were found")
	}

	images := loadImages(source, imageEntries, o.delimiter)
	var videos mediaResult
	if len(videoEntries) > 0 {
		tools, findErr := findFFmpegTools(o.ffmpegDir)
		if findErr != nil {
			return findErr
		}
		videos = loadVideos(source, videoEntries, o.delimiter, tools)
	} else {
		videos = mediaResult{kind: videoMedia, sets: make(map[string]*personSet)}
	}
	processResult(&images, o.threshold)
	processResult(&videos, o.threshold)

	if o.dryRun {
		printPlan(images, o.threshold)
		printPlan(videos, o.threshold)
		return nil
	}
	if err := ensureOutputAvailable(imageOutput, "-image-output"); err != nil {
		return err
	}
	if err := ensureOutputAvailable(videoOutput, "-video-output"); err != nil {
		return err
	}
	if err := writeBothOutputs(imageOutput, images, videoOutput, videos, o.threshold); err != nil {
		return err
	}

	printResult(images, imageOutput)
	printResult(videos, videoOutput)
	return nil
}

func resolveOutput(source, output string) (string, error) {
	if !filepath.IsAbs(output) {
		output = filepath.Join(source, output)
	}
	return filepath.Abs(output)
}

func splitCandidates(entries []os.DirEntry) ([]os.DirEntry, []os.DirEntry) {
	var images, videos []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		switch {
		case supportedImageExtensions[ext]:
			images = append(images, entry)
		case ext == ".mp4":
			videos = append(videos, entry)
		}
	}
	sort.Slice(images, func(i, j int) bool { return naturalLess(images[i].Name(), images[j].Name()) })
	sort.Slice(videos, func(i, j int) bool { return naturalLess(videos[i].Name(), videos[j].Name()) })
	return images, videos
}

func loadImages(source string, candidates []os.DirEntry, delimiter string) mediaResult {
	result := mediaResult{kind: imageMedia, sets: make(map[string]*personSet), candidateNo: len(candidates)}
	for i, entry := range candidates {
		person, ok := personFromName(entry.Name(), delimiter)
		if !ok {
			result.skipped = append(result.skipped, fmt.Sprintf("%s -- no person-name separator found", entry.Name()))
			continue
		}
		fmt.Printf("[image %d/%d] Reading thumbnail: %s\n", i+1, len(candidates), entry.Name())
		path := filepath.Join(source, entry.Name())
		feature, format, err := fingerprintFile(path)
		if err != nil {
			result.skipped = append(result.skipped, fmt.Sprintf("%s -- %v", entry.Name(), err))
			continue
		}
		addItem(&result, &photo{sourcePath: path, originalName: entry.Name(), person: person, format: format, features: []fingerprint{feature}, kind: imageMedia})
	}
	return result
}

func addItem(result *mediaResult, item *photo) {
	key := strings.ToLower(item.person)
	set := result.sets[key]
	if set == nil {
		set = &personSet{displayName: item.person}
		result.sets[key] = set
	}
	item.person = set.displayName
	set.photos = append(set.photos, item)
}

func processResult(result *mediaResult, threshold float64) {
	for _, key := range sortedPeople(result.sets) {
		set := result.sets[key]
		groups := clusterPhotos(set.photos, threshold)
		sortClusters(groups, set.photos)
		assignNames(set, groups, result.kind)
		result.groups += len(groups)
		result.items = append(result.items, set.photos...)
	}
	sort.Slice(result.items, func(i, j int) bool {
		if c := strings.Compare(strings.ToLower(result.items[i].person), strings.ToLower(result.items[j].person)); c != 0 {
			return c < 0
		}
		if result.items[i].group != result.items[j].group {
			return result.items[i].group < result.items[j].group
		}
		return result.items[i].variation < result.items[j].variation
	})
}

func printResult(result mediaResult, output string) {
	label := "Images"
	if result.kind == videoMedia {
		label = "Videos"
	}
	fmt.Printf("\n%s: copied %d file(s) for %d people into %d visual groups.\n", label, len(result.items), len(result.sets), result.groups)
	fmt.Printf("Output: %s\n", output)
	fmt.Printf("Review: %s\n", filepath.Join(output, "groups.html"))
	fmt.Printf("Mapping: %s\n", filepath.Join(output, "manifest.csv"))
	if len(result.skipped) > 0 {
		fmt.Printf("Skipped: %d file(s); see skipped.txt.\n", len(result.skipped))
	}
}
