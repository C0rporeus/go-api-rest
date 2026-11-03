package services

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func noisyImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	seed := uint32(1)
	next := func() uint8 {
		seed = seed*1664525 + 1013904223
		return uint8(seed >> 24)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: next(), G: next(), B: next(), A: 255})
		}
	}
	return img
}

func TestDownscaleImageShrinksLargeJPEG(t *testing.T) {
	var in bytes.Buffer
	if err := jpeg.Encode(&in, noisyImage(2400, 1200), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	out := downscaleImage("image/jpeg", in.Bytes())
	if len(out) >= in.Len() {
		t.Fatalf("expected smaller output: %d >= %d", len(out), in.Len())
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != maxImageDimension || cfg.Height != 600 {
		t.Fatalf("unexpected size %dx%d", cfg.Width, cfg.Height)
	}
}

func TestDownscaleImageShrinksTallPNG(t *testing.T) {
	var in bytes.Buffer
	if err := png.Encode(&in, noisyImage(800, 2400)); err != nil {
		t.Fatal(err)
	}
	out := downscaleImage("image/png", in.Bytes())
	cfg, err := png.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Height != maxImageDimension || cfg.Width != 400 {
		t.Fatalf("unexpected size %dx%d", cfg.Width, cfg.Height)
	}
}

func TestDownscaleImageLeavesSmallInvalidAndOtherFormatsUntouched(t *testing.T) {
	var small bytes.Buffer
	if err := jpeg.Encode(&small, noisyImage(400, 300), nil); err != nil {
		t.Fatal(err)
	}
	if got := downscaleImage("image/jpeg", small.Bytes()); !bytes.Equal(got, small.Bytes()) {
		t.Fatal("small image must be unchanged")
	}
	junk := []byte("not an image")
	if got := downscaleImage("image/jpeg", junk); !bytes.Equal(got, junk) {
		t.Fatal("invalid payload must be unchanged")
	}
	if got := downscaleImage("image/gif", junk); !bytes.Equal(got, junk) {
		t.Fatal("unsupported format must be unchanged")
	}
}
