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
	payload, err := json.Marshal(items)
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
