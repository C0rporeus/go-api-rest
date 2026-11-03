package services

import (
	"path"
	"strings"
)

// thumbWidth is the longest side of the card thumbnail served next to every
// cover image (<name>-w640.<ext>). Cards render at ≤ 370 CSS px, so 640 px
// covers 1x–1.7x screens; larger screens pick the full image through srcset.
const (
	thumbWidth  = 640
	thumbSuffix = "-w640"
)

// thumbObjectPath returns the object path of the thumbnail for an image path
// or URL: "dir/uuid.jpg" -> "dir/uuid-w640.jpg". It returns "" when the input
// already is a thumbnail or has no extension.
func thumbObjectPath(objectPath string) string {
	ext := path.Ext(objectPath)
	if ext == "" {
		return ""
	}
	stem := strings.TrimSuffix(objectPath, ext)
	if strings.HasSuffix(stem, thumbSuffix) {
		return ""
	}
	return stem + thumbSuffix + ext
}
