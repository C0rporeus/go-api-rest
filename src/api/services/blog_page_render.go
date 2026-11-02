package services

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
)

// BlogArticleDataElementID is the id of the <script type="application/json">
// element that carries the article to the browser, so the page renders without
// a second request. The front-end reads it (components/blog/BlogArticleView).
const BlogArticleDataElementID = "__BLOG_ARTICLE__"

const metaDescriptionMaxLength = 160

// blogKindTags mirror front-end/utils/content-kind.ts: an item is a blog article
// when one of its tags is one of these (accent- and case-insensitive).
var blogKindTags = map[string]bool{"blog": true, "articulo": true, "article": true, "post": true, "entrada": true}

var (
	titleTagPattern   = regexp.MustCompile(`(?is)<title[^>]*>.*?</title>`)
	htmlTagPattern    = regexp.MustCompile(`(?s)<[^>]*>`)
	firstImgPattern   = regexp.MustCompile(`(?i)<img\b[^>]*\bsrc=["']([^"']+)["']`)
	gcsUploadPattern  = regexp.MustCompile(`(?i)^https://storage\.googleapis\.com/[^/]+/portfolio-images/([0-9a-f-]{36}\.(?:jpe?g|png|gif|webp))(?:[?#].*)?$`)
	whitespacePattern = regexp.MustCompile(`\s+`)
	accentReplacer    = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")
)

// isPublicBlogArticle reports whether the item may be rendered as a public article.
func isPublicBlogArticle(item models.Experience) bool {
	if item.Visibility != constants.VisibilityPublic {
		return false
	}
	for _, tag := range item.Tags {
		if blogKindTags[accentReplacer.Replace(strings.ToLower(strings.TrimSpace(tag)))] {
			return true
		}
	}
	return false
}

// blogMetaDescription returns the summary, or the body text, cut at a word
// boundary to about maxLength characters.
func blogMetaDescription(summary, body string) string {
	text := strings.TrimSpace(summary)
	if text == "" {
		text = html.UnescapeString(htmlTagPattern.ReplaceAllString(body, " "))
	}
	text = strings.TrimSpace(whitespacePattern.ReplaceAllString(text, " "))
	if utf8.RuneCountInString(text) <= metaDescriptionMaxLength {
		return text
	}
	runes := []rune(text)[:metaDescriptionMaxLength-1]
	cut := string(runes)
	if i := strings.LastIndex(cut, " "); i > len(cut)*6/10 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

// blogShareImage returns a permanent image URL for og:image. Uploaded images live
// in a private bucket and are only reachable through expiring signed URLs, so
// they are served through GET /api/media/:name?variant=og (1200x630 JPEG).
// Must be called with canonical (unsigned) URLs.
func blogShareImage(item models.Experience) string {
	candidate := ""
	for _, u := range item.ImageURLs {
		if strings.TrimSpace(u) != "" {
			candidate = strings.TrimSpace(u)
			break
		}
	}
	if candidate == "" {
		if m := firstImgPattern.FindStringSubmatch(item.Body); m != nil {
			candidate = html.UnescapeString(m[1])
		}
	}
	if m := gcsUploadPattern.FindStringSubmatch(candidate); m != nil {
		return constants.PublicAPIURL() + "/api/media/" + strings.ToLower(m[1]) + "?variant=og"
	}
	if strings.HasPrefix(candidate, "https://") || strings.HasPrefix(candidate, "http://") {
		return candidate
	}
	return ""
}

// blogHeadTags builds the per-article <head> markup: description, canonical,
// Open Graph, Twitter card and the embedded article data.
func blogHeadTags(item models.Experience, shareImage string, embedded models.Experience) (string, error) {
	canonical := constants.SiteURL() + "/blog/" + item.ID + "/"
	description := blogMetaDescription(item.Summary, item.Body)
	attr := html.EscapeString

	var b strings.Builder
	meta := func(kind, key, value string) {
		b.WriteString(`<meta ` + kind + `="` + key + `" content="` + attr(value) + `">`)
	}
	meta("name", "description", description)
	b.WriteString(`<link rel="canonical" href="` + attr(canonical) + `">`)
	meta("property", "og:type", "article")
	meta("property", "og:site_name", constants.BlogSiteName)
	meta("property", "og:locale", "es_LA")
	meta("property", "og:title", item.Title)
	meta("property", "og:description", description)
	meta("property", "og:url", canonical)
	meta("property", "article:published_time", item.CreatedAt)
	meta("property", "article:modified_time", item.UpdatedAt)
	meta("name", "twitter:title", item.Title)
	meta("name", "twitter:description", description)
	if shareImage != "" {
		meta("name", "twitter:card", "summary_large_image")
		meta("property", "og:image", shareImage)
		meta("property", "og:image:alt", item.Title)
		meta("name", "twitter:image", shareImage)
		if strings.HasSuffix(shareImage, "?variant=og") {
			meta("property", "og:image:type", "image/jpeg")
			meta("property", "og:image:width", "1200")
			meta("property", "og:image:height", "630")
		}
	} else {
		meta("name", "twitter:card", "summary")
	}

	// json.Marshal escapes <, > and &, so the body cannot close the script element.
	data, err := json.Marshal(embedded)
	if err != nil {
		return "", err
	}
	b.WriteString(`<script id="` + BlogArticleDataElementID + `" type="application/json">`)
	b.Write(data)
	b.WriteString(`</script>`)
	return b.String(), nil
}

// injectIntoShell replaces the shell's <title> and appends head markup before </head>.
func injectIntoShell(shell, title, head string) string {
	titleTag := "<title>" + html.EscapeString(title) + "</title>"
	if loc := titleTagPattern.FindStringIndex(shell); loc != nil {
		shell = shell[:loc[0]] + titleTag + shell[loc[1]:]
	} else {
		head = titleTag + head
	}
	if i := strings.Index(strings.ToLower(shell), "</head>"); i >= 0 {
		return shell[:i] + head + shell[i:]
	}
	return head + shell
}
