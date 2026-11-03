package services

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	"golang.org/x/image/draw"
)

const (
	// maxImageDimension limits the longest side of stored images. Covers render at
	// ≤ 800 CSS px and article bodies at ≤ 900 px; 1200 px also matches Open Graph, so larger
	// originals only cost bandwidth.
	maxImageDimension = 1200
	// jpegQuality balances visual quality and weight for photos/illustrations.
	jpegQuality = 78
	// maxDecodePixels guards against decompression bombs.
	maxDecodePixels = 25_000_000
)

// downscaleImage shrinks JPEG/PNG images whose longest side exceeds
// maxImageDimension. JPEGs are re-encoded at jpegQuality. Other formats,
// images already small enough, malformed payloads and results that are not
// smaller than the original are returned unchanged.
func downscaleImage(contentType string, original []byte) []byte {
	return downscaleTo(contentType, original, maxImageDimension)
}

// downscaleTo is downscaleImage with an explicit longest-side limit.
func downscaleTo(contentType string, original []byte, limit int) []byte {
	kind := strings.ToLower(strings.TrimSpace(contentType))
	if kind != "image/jpeg" && kind != "image/png" {
		return original
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(original))
	if err != nil || config.Width <= 0 || config.Height <= 0 ||
		int64(config.Width)*int64(config.Height) > maxDecodePixels {
		return original
	}
	if config.Width <= limit && config.Height <= limit {
		return original
	}
	src, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return original
	}
	resized := resizeToFit(src, limit)

	var out bytes.Buffer
	if kind == "image/jpeg" {
		err = jpeg.Encode(&out, resized, &jpeg.Options{Quality: jpegQuality})
	} else {
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, resized)
	}
	if err != nil || out.Len() >= len(original) {
		return original
	}
	return out.Bytes()
}

// resizeToFit scales src so its longest side equals limit, keeping aspect ratio.
func resizeToFit(src image.Image, limit int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	newWidth, newHeight := limit, height*limit/width
	if height > width {
		newWidth, newHeight = width*limit/height, limit
	}
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}
