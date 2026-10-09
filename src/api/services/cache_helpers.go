package services

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// --- HTTP / caching helpers ---

func buildCollectionETag(items []userModel.Experience) string {
	// GCS signed URLs contain time-varying query parameters. They are an
	// access mechanism, not content, so exclude their signatures from the
	// validator; otherwise every request looks like a content change and a
	// client can never reuse its cached representation with a 304 response.
	canonical := make([]userModel.Experience, len(items))
	copy(canonical, items)
	gcsPrefix := constants.GCSURLPrefix()
	if gcsPrefix != "" {
		for i := range canonical {
			canonical[i].ImageURLs = append([]string(nil), items[i].ImageURLs...)
			for j, imageURL := range canonical[i].ImageURLs {
				if strings.HasPrefix(imageURL, gcsPrefix) {
					canonical[i].ImageURLs[j] = stripQueryParams(imageURL)
				}
			}
			canonical[i].Body = StripBodySignedParams(canonical[i].Body)
		}
	}

	payload, err := json.Marshal(canonical)
	if err != nil {
		return ""
	}
	sum := sha1.Sum(payload)
	return "W/\"" + hex.EncodeToString(sum[:]) + "\""
}

func matchesIfNoneMatchHeader(ifNoneMatch, etag string) bool {
	if etag == "" {
		return false
	}
	if strings.TrimSpace(ifNoneMatch) == "*" {
		return true
	}

	for _, candidate := range strings.Split(ifNoneMatch, ",") {
		if strings.TrimSpace(candidate) == etag {
			return true
		}
	}
	return false
}

func setPublicCollectionCacheHeaders(c fiber.Ctx, etag string) {
	c.Set("Cache-Control", constants.PublicCollectionCacheControl)
	if etag != "" {
		c.Set("ETag", etag)
	}
}
