// Copyright (C) 2026 SmartFrameSorter contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	thumbnailSize    = 32
	maxImagePixels   = 250_000_000
	defaultThreshold = 0.18
)

var supportedImageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".bmp": true, ".tif": true, ".tiff": true, ".webp": true,
}

type options struct {
	source, imageOutput, videoOutput, legacyOutput, delimiter, ffmpegDir string
	threshold                                                            float64
	dryRun                                                               bool
}

type fingerprint struct {
	gray      []float64
	hog       []float64
	colorHist []float64
	colorGrid []float64
	phash     uint64
}

type photo struct {
	sourcePath, originalName, person, format string
	features                                 []fingerprint
	kind                                     mediaKind
	posterPNG                                []byte
	group, variation                         int
	newName                                  string
	distanceToCenter                         float64
}

type personSet struct {
	displayName string
	photos      []*photo
}

type cluster struct {
	indices []int
}

type galleryGroup struct {
	Person string
	Number int
	Items  []galleryItem
}

type galleryItem struct {
	NewName, OriginalName, MediaURL, PosterURL string
	Distance                                   string
	IsVideo                                    bool
}

func main() {
	opts := parseFlags()
	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
}

func personFromName(filename, delimiter string) (string, bool) {
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	cut := -1
	switch delimiter {
	case "underscore":
		cut = strings.IndexRune(stem, '_')
	case "space":
		for i, r := range stem {
			if unicode.IsSpace(r) {
				cut = i
				break
			}
		}
	default:
		for i, r := range stem {
			if r == '_' || unicode.IsSpace(r) {
				cut = i
				break
			}
		}
	}
	if cut <= 0 {
		return "", false
	}
	name := strings.TrimSpace(stem[:cut])
	return name, name != ""
}

func fingerprintFile(path string) (fingerprint, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fingerprint{}, "", err
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fingerprint{}, "", fmt.Errorf("unsupported or damaged image: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxImagePixels {
		return fingerprint{}, "", fmt.Errorf("image dimensions are invalid or exceed %d megapixels", maxImagePixels/1_000_000)
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fingerprint{}, "", fmt.Errorf("decode image: %w", err)
	}
	orientation := 1
	if format == "jpeg" {
		orientation = jpegEXIFOrientation(data)
	}
	thumb := makeThumbnail(img, orientation, thumbnailSize)
	return buildFingerprint(thumb, thumbnailSize), format, nil
}

func makeThumbnail(img image.Image, orientation, size int) []float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	ow, oh := w, h
	if orientation >= 5 && orientation <= 8 {
		ow, oh = h, w
	}
	out := make([]float64, size*size*3)
	for y := 0; y < size; y++ {
		fy := (float64(y)+0.5)*float64(oh)/float64(size) - 0.5
		if fy < 0 {
			fy = 0
		}
		if fy > float64(oh-1) {
			fy = float64(oh - 1)
		}
		y0 := int(math.Floor(fy))
		y1 := min(y0+1, oh-1)
		ty := fy - float64(y0)
		for x := 0; x < size; x++ {
			fx := (float64(x)+0.5)*float64(ow)/float64(size) - 0.5
			if fx < 0 {
				fx = 0
			}
			if fx > float64(ow-1) {
				fx = float64(ow - 1)
			}
			x0 := int(math.Floor(fx))
			x1 := min(x0+1, ow-1)
			tx := fx - float64(x0)
			c00 := orientedRGB(img, b, orientation, x0, y0)
			c10 := orientedRGB(img, b, orientation, x1, y0)
			c01 := orientedRGB(img, b, orientation, x0, y1)
			c11 := orientedRGB(img, b, orientation, x1, y1)
			for channel := 0; channel < 3; channel++ {
				top := c00[channel]*(1-tx) + c10[channel]*tx
				bottom := c01[channel]*(1-tx) + c11[channel]*tx
				out[(y*size+x)*3+channel] = top*(1-ty) + bottom*ty
			}
		}
	}
	return out
}

func orientedRGB(img image.Image, b image.Rectangle, orientation, x, y int) [3]float64 {
	w, h := b.Dx(), b.Dy()
	sx, sy := x, y
	switch orientation {
	case 2:
		sx = w - 1 - x
	case 3:
		sx, sy = w-1-x, h-1-y
	case 4:
		sy = h - 1 - y
	case 5:
		sx, sy = y, x
	case 6:
		sx, sy = y, h-1-x
	case 7:
		sx, sy = w-1-y, h-1-x
	case 8:
		sx, sy = w-1-y, x
	}
	r, g, bl, _ := img.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
	return [3]float64{float64(r) / 257, float64(g) / 257, float64(bl) / 257}
}

func buildFingerprint(rgb []float64, size int) fingerprint {
	grayRaw := make([]float64, size*size)
	for i := range grayRaw {
		r, g, b := rgb[i*3], rgb[i*3+1], rgb[i*3+2]
		grayRaw[i] = 0.299*r + 0.587*g + 0.114*b
	}
	gray := normalize(grayRaw)
	return fingerprint{
		gray:      gray,
		hog:       edgeHistogram(grayRaw, size),
		colorHist: hsvHistogram(rgb),
		colorGrid: spatialColorGrid(rgb, size),
		phash:     perceptualHash(grayRaw, size),
	}
}

func normalize(values []float64) []float64 {
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	std := math.Sqrt(variance / float64(len(values)))
	if std < 1 {
		std = 1
	}
	out := make([]float64, len(values))
	for i, v := range values {
		out[i] = (v - mean) / std
	}
	return out
}

func perceptualHash(gray []float64, size int) uint64 {
	coeff := make([]float64, 0, 63)
	for v := 0; v < 8; v++ {
		for u := 0; u < 8; u++ {
			if u == 0 && v == 0 {
				continue
			}
			sum := 0.0
			for y := 0; y < size; y++ {
				cy := math.Cos((float64(2*y+1) * float64(v) * math.Pi) / (2 * float64(size)))
				for x := 0; x < size; x++ {
					cx := math.Cos((float64(2*x+1) * float64(u) * math.Pi) / (2 * float64(size)))
					sum += gray[y*size+x] * cx * cy
				}
			}
			coeff = append(coeff, sum)
		}
	}
	ordered := append([]float64(nil), coeff...)
	sort.Float64s(ordered)
	median := ordered[len(ordered)/2]
	var hash uint64
	for i, v := range coeff {
		if v > median {
			hash |= uint64(1) << i
		}
	}
	return hash
}

func edgeHistogram(gray []float64, size int) []float64 {
	const cells, bins = 4, 6
	out := make([]float64, cells*cells*bins)
	for y := 1; y < size-1; y++ {
		for x := 1; x < size-1; x++ {
			gx := gray[y*size+x+1] - gray[y*size+x-1]
			gy := gray[(y+1)*size+x] - gray[(y-1)*size+x]
			mag := math.Hypot(gx, gy)
			angle := math.Atan2(gy, gx)
			if angle < 0 {
				angle += math.Pi
			}
			if angle >= math.Pi {
				angle -= math.Pi
			}
			bin := min(int(angle/math.Pi*bins), bins-1)
			cx, cy := min(x*cells/size, cells-1), min(y*cells/size, cells-1)
			out[(cy*cells+cx)*bins+bin] += mag
		}
	}
	return unitNormalize(out)
}

func hsvHistogram(rgb []float64) []float64 {
	const hueBins, satBins, valBins = 8, 3, 3
	out := make([]float64, hueBins*satBins*valBins)
	for i := 0; i < len(rgb); i += 3 {
		r, g, b := rgb[i]/255, rgb[i+1]/255, rgb[i+2]/255
		mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
		d := mx - mn
		h := 0.0
		if d > 1e-9 {
			switch mx {
			case r:
				h = math.Mod((g-b)/d, 6)
			case g:
				h = (b-r)/d + 2
			default:
				h = (r-g)/d + 4
			}
			h /= 6
			if h < 0 {
				h++
			}
		}
		s := 0.0
		if mx > 0 {
			s = d / mx
		}
		hi := min(int(h*hueBins), hueBins-1)
		si := min(int(s*satBins), satBins-1)
		vi := min(int(mx*valBins), valBins-1)
		out[(hi*satBins+si)*valBins+vi]++
	}
	for i := range out {
		out[i] /= float64(len(rgb) / 3)
	}
	return out
}

func spatialColorGrid(rgb []float64, size int) []float64 {
	const cells = 4
	out := make([]float64, cells*cells*3)
	counts := make([]int, cells*cells)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			idx := (y*size + x) * 3
			r, g, b := rgb[idx], rgb[idx+1], rgb[idx+2]
			yy := 0.299*r + 0.587*g + 0.114*b
			cb := 128 - 0.168736*r - 0.331264*g + 0.5*b
			cr := 128 + 0.5*r - 0.418688*g - 0.081312*b
			cell := min(y*cells/size, cells-1)*cells + min(x*cells/size, cells-1)
			out[cell*3] += yy / 255
			out[cell*3+1] += cb / 255
			out[cell*3+2] += cr / 255
			counts[cell]++
		}
	}
	for cell, count := range counts {
		for c := 0; c < 3; c++ {
			out[cell*3+c] /= float64(count)
		}
	}
	return out
}

func unitNormalize(values []float64) []float64 {
	norm := 0.0
	for _, v := range values {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return values
	}
	for i := range values {
		values[i] /= norm
	}
	return values
}

func fingerprintDistance(a, b fingerprint) float64 {
	if equalFingerprint(a, b) {
		return 0
	}
	phash := float64(bits.OnesCount64(a.phash^b.phash)) / 63
	dot := 0.0
	for i := range a.gray {
		dot += a.gray[i] * b.gray[i]
	}
	gray := clamp((1-dot/float64(len(a.gray)))/2, 0, 1)
	dot = 0
	for i := range a.hog {
		dot += a.hog[i] * b.hog[i]
	}
	hog := clamp(1-dot, 0, 1)
	hellinger := 0.0
	for i := range a.colorHist {
		d := math.Sqrt(a.colorHist[i]) - math.Sqrt(b.colorHist[i])
		hellinger += d * d
	}
	hellinger = clamp(math.Sqrt(hellinger)/math.Sqrt2, 0, 1)
	colorGrid := 0.0
	for i := 0; i < len(a.colorGrid); i += 3 {
		colorGrid += 0.4 * math.Abs(a.colorGrid[i]-b.colorGrid[i])
		colorGrid += 0.3 * math.Abs(a.colorGrid[i+1]-b.colorGrid[i+1])
		colorGrid += 0.3 * math.Abs(a.colorGrid[i+2]-b.colorGrid[i+2])
	}
	colorGrid /= float64(len(a.colorGrid) / 3)
	return 0.30*phash + 0.30*gray + 0.25*hog + 0.10*hellinger + 0.05*colorGrid
}

func equalFingerprint(a, b fingerprint) bool {
	if a.phash != b.phash || len(a.gray) != len(b.gray) || len(a.hog) != len(b.hog) || len(a.colorHist) != len(b.colorHist) || len(a.colorGrid) != len(b.colorGrid) {
		return false
	}
	for i := range a.gray {
		if a.gray[i] != b.gray[i] {
			return false
		}
	}
	for i := range a.hog {
		if a.hog[i] != b.hog[i] {
			return false
		}
	}
	for i := range a.colorHist {
		if a.colorHist[i] != b.colorHist[i] {
			return false
		}
	}
	for i := range a.colorGrid {
		if a.colorGrid[i] != b.colorGrid[i] {
			return false
		}
	}
	return true
}

// photoDistance compares one image fingerprint or two timing-tolerant video
// frame sets. The symmetric nearest-neighbour average prevents a shared intro
// frame from making otherwise unrelated videos look identical.
func photoDistance(a, b *photo) float64 {
	if len(a.features) == 0 || len(b.features) == 0 {
		return 1
	}
	if len(a.features) == 1 && len(b.features) == 1 {
		return fingerprintDistance(a.features[0], b.features[0])
	}
	total := 0.0
	for _, fa := range a.features {
		best := math.Inf(1)
		for _, fb := range b.features {
			best = math.Min(best, fingerprintDistance(fa, fb))
		}
		total += best
	}
	for _, fb := range b.features {
		best := math.Inf(1)
		for _, fa := range a.features {
			best = math.Min(best, fingerprintDistance(fa, fb))
		}
		total += best
	}
	return total / float64(len(a.features)+len(b.features))
}

func clusterPhotos(photos []*photo, threshold float64) []cluster {
	n := len(photos)
	if n == 0 {
		return nil
	}
	distances := make([][]float64, n)
	for i := range distances {
		distances[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			d := photoDistance(photos[i], photos[j])
			distances[i][j], distances[j][i] = d, d
		}
	}
	clusters := make([]cluster, n)
	for i := range clusters {
		clusters[i] = cluster{indices: []int{i}}
	}
	for {
		bestI, bestJ, best := -1, -1, math.Inf(1)
		for i := 0; i < len(clusters); i++ {
			for j := i + 1; j < len(clusters); j++ {
				d := averageLinkage(clusters[i], clusters[j], distances)
				if d < best {
					bestI, bestJ, best = i, j, d
				}
			}
		}
		if bestI < 0 || best > threshold {
			break
		}
		clusters[bestI].indices = append(clusters[bestI].indices, clusters[bestJ].indices...)
		clusters = append(clusters[:bestJ], clusters[bestJ+1:]...)
	}
	for i := range clusters {
		sort.Slice(clusters[i].indices, func(a, b int) bool {
			return naturalLess(photos[clusters[i].indices[a]].originalName, photos[clusters[i].indices[b]].originalName)
		})
	}
	return clusters
}

func averageLinkage(a, b cluster, distances [][]float64) float64 {
	total := 0.0
	for _, i := range a.indices {
		for _, j := range b.indices {
			total += distances[i][j]
		}
	}
	return total / float64(len(a.indices)*len(b.indices))
}

func sortClusters(groups []cluster, photos []*photo) {
	sort.Slice(groups, func(i, j int) bool {
		return naturalLess(photos[groups[i].indices[0]].originalName, photos[groups[j].indices[0]].originalName)
	})
}

func assignNames(set *personSet, groups []cluster, kind mediaKind) {
	safePerson := safeFilename(set.displayName)
	for gi, group := range groups {
		medoid := groupMedoid(group, set.photos)
		for yi, index := range group.indices {
			p := set.photos[index]
			p.group, p.variation = gi+1, yi+1
			p.distanceToCenter = photoDistance(p, set.photos[medoid])
			extension := extensionForFormat(p.format, p.originalName)
			if kind == videoMedia {
				extension = ".mp4"
			}
			p.newName = fmt.Sprintf("%s_%d_%d%s", safePerson, p.group, p.variation, extension)
		}
	}
}

func groupMedoid(group cluster, photos []*photo) int {
	bestIndex, bestTotal := group.indices[0], math.Inf(1)
	for _, i := range group.indices {
		total := 0.0
		for _, j := range group.indices {
			total += photoDistance(photos[i], photos[j])
		}
		if total < bestTotal {
			bestIndex, bestTotal = i, total
		}
	}
	return bestIndex
}

func extensionForFormat(format, original string) string {
	switch format {
	case "jpeg":
		return ".jpeg"
	case "png":
		return ".png"
	case "gif":
		return ".gif"
	case "bmp":
		return ".bmp"
	case "tiff":
		return ".tiff"
	case "webp":
		return ".webp"
	default:
		ext := strings.ToLower(filepath.Ext(original))
		if ext == "" {
			return ".img"
		}
		return ext
	}
}

func safeFilename(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\\|?*`, r) {
			return '-'
		}
		return r
	}, s)
	s = strings.Trim(s, " .")
	if s == "" {
		return "person"
	}
	reserved := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true}
	if reserved[strings.ToUpper(s)] {
		s += "-person"
	}
	return s
}

func sortedPeople(sets map[string]*personSet) []string {
	keys := make([]string, 0, len(sets))
	for key := range sets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return naturalLess(sets[keys[i]].displayName, sets[keys[j]].displayName) })
	return keys
}

func samePath(a, b string) bool {
	a = strings.TrimRight(filepath.Clean(a), `\\/`)
	b = strings.TrimRight(filepath.Clean(b), `\\/`)
	return strings.EqualFold(a, b)
}

func naturalLess(a, b string) bool {
	ar, br := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	for i, j := 0, 0; i < len(ar) && j < len(br); {
		if unicode.IsDigit(ar[i]) && unicode.IsDigit(br[j]) {
			ii, jj := i, j
			for ii < len(ar) && unicode.IsDigit(ar[ii]) {
				ii++
			}
			for jj < len(br) && unicode.IsDigit(br[jj]) {
				jj++
			}
			an := strings.TrimLeft(string(ar[i:ii]), "0")
			if an == "" {
				an = "0"
			}
			bn := strings.TrimLeft(string(br[j:jj]), "0")
			if bn == "" {
				bn = "0"
			}
			if len(an) != len(bn) {
				return len(an) < len(bn)
			}
			if an != bn {
				return an < bn
			}
			i, j = ii, jj
			continue
		}
		if ar[i] != br[j] {
			return ar[i] < br[j]
		}
		i++
		j++
	}
	return len(ar) < len(br)
}

func jpegEXIFOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	for pos := 2; pos+4 <= len(data); {
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos >= len(data) {
			break
		}
		marker := data[pos]
		pos++
		if marker == 0xd9 || marker == 0xda {
			break
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if pos+2 > len(data) {
			break
		}
		length := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		if length < 2 || pos+length > len(data) {
			break
		}
		payload := data[pos+2 : pos+length]
		if marker == 0xe1 && len(payload) >= 6 && bytes.Equal(payload[:6], []byte{'E', 'x', 'i', 'f', 0, 0}) {
			if orientation := tiffOrientation(payload[6:]); orientation >= 1 && orientation <= 8 {
				return orientation
			}
		}
		pos += length
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return 1
	}
	offset := int(order.Uint32(tiff[4:8]))
	if offset < 0 || offset+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	for i := 0; i < count; i++ {
		entry := offset + 2 + i*12
		if entry+12 > len(tiff) {
			break
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 && order.Uint16(tiff[entry+2:entry+4]) == 3 && order.Uint32(tiff[entry+4:entry+8]) >= 1 {
			return int(order.Uint16(tiff[entry+8 : entry+10]))
		}
	}
	return 1
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
