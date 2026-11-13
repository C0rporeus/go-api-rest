package services

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
)

// BlogStaticElementID is the id of the server-rendered copy of the article.
// It puts the title and the full text in the HTML for crawlers that do not run
// JavaScript; the front-end removes it once its own render is in place
// (components/blog/BlogArticleView).
const BlogStaticElementID = "__BLOG_STATIC__"

var bodyOpenTagPattern = regexp.MustCompile(`(?i)<body[^>]*>`)

// blogStaticHideShellStyle hides the shell's own <main> ("Cargando artículo…")
// while the static copy is in the document, so the article is not shown twice.
const blogStaticHideShellStyle = `<style>#` + BlogStaticElementID + `~#__next main{display:none}</style>`

// blogJSONLD builds the schema.org BlogPosting block for the article.
func blogJSONLD(item models.Experience, shareImage, description string) (string, error) {
	canonical := constants.SiteURL() + "/blog/" + item.ID + "/"
	author := map[string]any{"@type": "Person", "name": constants.BlogSiteName, "url": constants.SiteURL()}
	data := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         item.Title,
		"description":      description,
		"url":              canonical,
		"mainEntityOfPage": map[string]any{"@type": "WebPage", "@id": canonical},
		"datePublished":    item.CreatedAt,
		"dateModified":     item.UpdatedAt,
		"inLanguage":       "es",
		"author":           author,
		"publisher":        author,
	}
	if shareImage != "" {
		data["image"] = shareImage
	}
	if len(item.Tags) > 0 {
		data["keywords"] = strings.Join(item.Tags, ", ")
	}
	// json.Marshal escapes <, > and &, so no value can close the script element.
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return `<script type="application/ld+json">` + string(encoded) + `</script>`, nil
}

// blogStaticArticle renders the article (title, summary and sanitized body) with
// the same classes the front-end uses. Pass the copy with signed image URLs.
func blogStaticArticle(item models.Experience) string {
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<main id="` + BlogStaticElementID + `" class="public-main article-page pt-[104px] md:pt-[120px]">`)
	b.WriteString(`<article class="article-layout mx-auto w-full max-w-6xl"><header class="article-heading">`)
	b.WriteString(`<p class="article-kicker">Artículo técnico</p>`)
	b.WriteString(`<h1 class="article-title">` + esc(item.Title) + `</h1>`)
	if summary := strings.TrimSpace(item.Summary); summary != "" {
		b.WriteString(`<p class="article-summary">` + esc(summary) + `</p>`)
	}
	b.WriteString(`<p class="article-meta">Publicado <time datetime="` + esc(item.CreatedAt) + `">` + esc(dateOnly(item.CreatedAt)) + `</time></p>`)
	b.WriteString(`</header><div class="article-content">`)
	b.WriteString(sanitizer.SanitizeHTML(item.Body))
	b.WriteString(`</div><p><a href="/blog/">Todos los artículos</a></p></article></main>`)
	return b.String()
}

// dateOnly keeps the YYYY-MM-DD part of an RFC 3339 timestamp.
func dateOnly(timestamp string) string {
	if i := strings.IndexByte(timestamp, 'T'); i > 0 {
		return timestamp[:i]
	}
	return timestamp
}

// injectIntoBody inserts markup right after the opening <body> tag, before the
// application root. A shell without <body> is returned unchanged.
func injectIntoBody(shell, markup string) string {
	loc := bodyOpenTagPattern.FindStringIndex(shell)
	if loc == nil {
		return shell
	}
	return shell[:loc[1]] + markup + shell[loc[1]:]
}
