package services

import (
	"regexp"
	"strings"

	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
)

const (
	// cardsViewParam selects the lightweight listing used by landing cards.
	cardsViewParam = "cards"
	// cardSummaryRunes caps the fallback summary derived from the body.
	cardSummaryRunes = 220
)

var firstImageSrcPattern = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

// toCardView drops the heavy HTML body from an experience while keeping what a
// card needs: a summary (derived from the body when empty) and a cover image
// (first body image when imageUrls is empty). Call before signing URLs.
func toCardView(item userModel.Experience) userModel.Experience {
	if len(item.ImageURLs) == 0 {
		if match := firstImageSrcPattern.FindStringSubmatch(item.Body); len(match) == 2 {
			item.ImageURLs = []string{match[1]}
		}
	}
	if strings.TrimSpace(item.Summary) == "" {
		text := strings.Join(strings.Fields(sanitizer.StripHTML(item.Body)), " ")
		if runes := []rune(text); len(runes) > cardSummaryRunes {
			text = strings.TrimSpace(string(runes[:cardSummaryRunes])) + "…"
		}
		item.Summary = text
	}
	item.Body = ""
	if prefix := constants.GCSURLPrefix(); prefix != "" && len(item.ImageURLs) > 0 && strings.HasPrefix(item.ImageURLs[0], prefix) {
		if thumb := thumbObjectPath(stripQueryParams(item.ImageURLs[0])); thumb != "" {
			item.ThumbURL = thumb
		}
	}
	return item
}

// toCardViews applies toCardView to every item.
func toCardViews(items []userModel.Experience) []userModel.Experience {
	cards := make([]userModel.Experience, len(items))
	for i, item := range items {
		cards[i] = toCardView(item)
	}
	return cards
}
