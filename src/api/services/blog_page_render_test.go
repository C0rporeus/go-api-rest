package services

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
)

func TestIsPublicBlogArticle(t *testing.T) {
	cases := []struct {
		name string
		item models.Experience
		want bool
	}{
		{"public blog", models.Experience{Visibility: constants.VisibilityPublic, Tags: []string{"kubernetes", "blog"}}, true},
		{"accented tag", models.Experience{Visibility: constants.VisibilityPublic, Tags: []string{" Artículo "}}, true},
		{"private blog", models.Experience{Visibility: constants.VisibilityPrivate, Tags: []string{"blog"}}, false},
		{"public skill", models.Experience{Visibility: constants.VisibilityPublic, Tags: []string{"skill"}}, false},
		{"substring is not enough", models.Experience{Visibility: constants.VisibilityPublic, Tags: []string{"weblog-tools"}}, false},
	}
	for _, c := range cases {
		if got := isPublicBlogArticle(c.item); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestBlogMetaDescription(t *testing.T) {
	if got := blogMetaDescription("  Resumen corto. ", "<p>ignorado</p>"); got != "Resumen corto." {
		t.Errorf("summary: got %q", got)
	}
	if got := blogMetaDescription("", "<h2>Título</h2><p>Texto &amp; más</p>"); got != "Título Texto & más" {
		t.Errorf("body fallback: got %q", got)
	}
	long := strings.Repeat("palabra ", 60)
	got := blogMetaDescription(long, "")
	if n := len([]rune(got)); n > metaDescriptionMaxLength || !strings.HasSuffix(got, "…") || strings.HasSuffix(got, " …") {
		t.Errorf("long text: got %d runes %q", n, got)
	}
}

func TestBlogShareImage(t *testing.T) {
	t.Setenv("PUBLIC_API_URL", "https://api.example.dev")
	gcs := "https://storage.googleapis.com/bucket/portfolio-images/0aef4e1f-9ddd-40fd-9aa2-6d35154b78ab.JPG?X-Goog-Signature=abc"
	want := "https://api.example.dev/api/media/0aef4e1f-9ddd-40fd-9aa2-6d35154b78ab.jpg?variant=og"

	if got := blogShareImage(models.Experience{ImageURLs: []string{" ", gcs}}); got != want {
		t.Errorf("imageUrls: got %q", got)
	}
	if got := blogShareImage(models.Experience{Body: `<p>x</p><img src="` + gcs + `"><img src="https://other/x.png">`}); got != want {
		t.Errorf("first body image: got %q", got)
	}
	if got := blogShareImage(models.Experience{Body: `<img src="https://cdn.example.com/a.png">`}); got != "https://cdn.example.com/a.png" {
		t.Errorf("external image: got %q", got)
	}
	if got := blogShareImage(models.Experience{Body: `<img src="/relative.png">`}); got != "" {
		t.Errorf("relative image: got %q", got)
	}
}

func TestBlogHeadTagsEscapesAndEmbedsSafely(t *testing.T) {
	t.Setenv("SITE_URL", "https://www.example.dev")
	item := models.Experience{
		ID:      "abc",
		Title:   `Título "con" <comillas> & más`,
		Summary: "Resumen",
		Body:    `<p>cuerpo</p></script><script>alert(1)</script>`,
	}
	head, err := blogHeadTags(item, "https://api.example.dev/api/media/x.jpg?variant=og", item)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<meta property="og:title" content="Título &#34;con&#34; &lt;comillas&gt; &amp; más">`,
		`<link rel="canonical" href="https://www.example.dev/blog/abc/">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta property="og:image:width" content="1200">`,
	} {
		if !strings.Contains(head, want) {
			t.Errorf("head does not contain %s\n%s", want, head)
		}
	}
	// The body must not be able to close the data element early.
	if strings.Count(head, "</script>") != 1 {
		t.Fatalf("embedded data broke out of its script element:\n%s", head)
	}
	raw := regexp.MustCompile(`(?s)<script id="__BLOG_ARTICLE__" type="application/json">(.*)</script>`).FindStringSubmatch(head)
	var decoded models.Experience
	if raw == nil || json.Unmarshal([]byte(raw[1]), &decoded) != nil || decoded.Body != item.Body {
		t.Fatalf("embedded article does not round-trip: %v", raw)
	}
}

func TestBlogHeadTagsWithoutImage(t *testing.T) {
	head, err := blogHeadTags(models.Experience{ID: "x", Title: "T"}, "", models.Experience{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(head, `content="summary"`) || strings.Contains(head, "og:image") {
		t.Errorf("unexpected image tags without an image:\n%s", head)
	}
}

func TestInjectIntoShell(t *testing.T) {
	shell := `<html><head><meta charset="utf-8"><title data-x="1">Artículo | Blog</title></head><body>app</body></html>`
	got := injectIntoShell(shell, "Mi <artículo>", `<meta name="description" content="d">`)
	want := `<html><head><meta charset="utf-8"><title>Mi &lt;artículo&gt;</title><meta name="description" content="d"></head><body>app</body></html>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := injectIntoShell(`<html><HEAD></HEAD></html>`, "T", "<x>"); got != `<html><HEAD><title>T</title><x></HEAD></html>` {
		t.Errorf("shell without title: got %s", got)
	}
}

func TestBlogJSONLDDescribesTheArticleAndCannotCloseTheScript(t *testing.T) {
	item := models.Experience{
		ID: "abc", Title: `</script><b>x</b>`, Tags: []string{"blog", "kubernetes"},
		CreatedAt: "2026-11-02T10:00:00Z", UpdatedAt: "2026-11-03T10:00:00Z",
	}
	got, err := blogJSONLD(item, "https://img/x.jpg", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "</script>") != 1 {
		t.Fatalf("title must not close the script element: %s", got)
	}
	for _, want := range []string{`"@type":"BlogPosting"`, `/blog/abc/`, `"datePublished":"2026-11-02T10:00:00Z"`, `"image":"https://img/x.jpg"`, `"keywords":"blog, kubernetes"`} {
		if !strings.Contains(got, want) {
			t.Errorf("JSON-LD missing %s in %s", want, got)
		}
	}
	if withoutImage, _ := blogJSONLD(models.Experience{ID: "abc"}, "", ""); strings.Contains(withoutImage, `"image"`) {
		t.Errorf("no image expected: %s", withoutImage)
	}
}

func TestBlogStaticArticleEscapesTextAndSanitizesBody(t *testing.T) {
	got := blogStaticArticle(models.Experience{
		Title:     `Tokens <JWT> & "claims"`,
		Summary:   "Resumen <b>plano</b>",
		Body:      `<h2>Sección</h2><p onclick="x()">Texto</p><script>alert(1)</script>`,
		CreatedAt: "2026-11-02T10:00:00Z",
	})
	for _, want := range []string{
		`id="` + BlogStaticElementID + `"`,
		`<h1 class="article-title">Tokens &lt;JWT&gt; &amp; &#34;claims&#34;</h1>`,
		`Resumen &lt;b&gt;plano&lt;/b&gt;`,
		`<h2>Sección</h2>`, `<p>Texto</p>`, `>2026-11-02</time>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("static article missing %q in %s", want, got)
		}
	}
	for _, banned := range []string{"<script", "onclick", "alert(1)"} {
		if strings.Contains(got, banned) {
			t.Errorf("static article must not contain %q: %s", banned, got)
		}
	}
}

func TestInjectIntoBody(t *testing.T) {
	if got := injectIntoBody(`<html><BODY class="a"><div id="__next"></div></body></html>`, "<main>x</main>"); got != `<html><BODY class="a"><main>x</main><div id="__next"></div></body></html>` {
		t.Errorf("unexpected: %s", got)
	}
	if got := injectIntoBody("sin cuerpo", "<main>x</main>"); got != "sin cuerpo" {
		t.Errorf("a shell without <body> must be returned unchanged: %s", got)
	}
}
