package services

import (
	"bytes"
	"image"
	"image/jpeg"
	"sync"

	// Registered decoders for image.Decode.
	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"

	xdraw "golang.org/x/image/draw"
)

// Open Graph cards are rendered at 1.91:1; 1200x630 is the size every platform accepts.
const (
	ogImageWidth   = 1200
	ogImageHeight  = 630
	ogJPEGQuality  = 82
	ogCacheEntries = 64
)

// ogImageCache keeps rendered variants per object; uploads are immutable (UUID names).
var (
	ogImageCache   = map[string][]byte{}
	ogImageCacheMu sync.RWMutex
)

// renderOGImage center-crops the source to 1.91:1 and scales it to 1200x630 JPEG,
// keeping previews light (< ~300 KB) for Facebook, LinkedIn and WhatsApp.
func renderOGImage(original []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, ogImageWidth, ogImageHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, coverCropRect(src.Bounds()), xdraw.Src, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: ogJPEGQuality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// coverCropRect returns the largest 1.91:1 region of b. Portrait images keep the
// upper third, where titles and subjects usually are, instead of the exact center.
func coverCropRect(b image.Rectangle) image.Rectangle {
	w, h := b.Dx(), b.Dy()
	if w*ogImageHeight > h*ogImageWidth {
		cropW := h * ogImageWidth / ogImageHeight
		x0 := b.Min.X + (w-cropW)/2
		return image.Rect(x0, b.Min.Y, x0+cropW, b.Max.Y)
	}
	cropH := w * ogImageHeight / ogImageWidth
	y0 := b.Min.Y + (h-cropH)/3
	return image.Rect(b.Min.X, y0, b.Max.X, y0+cropH)
}

// cachedOGImage returns a rendered variant, computing it at most once per object.
func cachedOGImage(key string, original []byte) ([]byte, error) {
	ogImageCacheMu.RLock()
	cached, ok := ogImageCache[key]
	ogImageCacheMu.RUnlock()
	if ok {
		return cached, nil
	}
	rendered, err := renderOGImage(original)
	if err != nil {
		return nil, err
	}
	ogImageCacheMu.Lock()
	if len(ogImageCache) >= ogCacheEntries {
		ogImageCache = map[string][]byte{}
	}
	ogImageCache[key] = rendered
	ogImageCacheMu.Unlock()
	return rendered, nil
}
