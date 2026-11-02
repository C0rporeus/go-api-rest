package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository/memory"

	"github.com/gofiber/fiber/v3"
)

const testBlogShell = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Artículo | Blog</title><script src="/_next/static/chunks/app.js"></script></head><body><div id="__next"></div></body></html>`

func stubBlogShell(t *testing.T, fn func(context.Context, string) (string, error)) {
	t.Helper()
	original := fetchBlogShellFunc
	fetchBlogShellFunc = fn
	t.Cleanup(func() { fetchBlogShellFunc = original })
}

func newBlogTestApp(t *testing.T, items ...models.Experience) *fiber.App {
	t.Helper()
	repo := memory.NewExperienceRepository()
	for _, item := range items {
		if err := repo.Create(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	app := fiber.New()
	app.Get("/blog/:id", NewBlogPageService(repo).RenderArticle)
	return app
}

func getBlogPage(t *testing.T, app *fiber.App, path string) (*http.Response, string) {
	t.Helper()
	res, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func TestRenderArticleBuildsPageFromTheDatabase(t *testing.T) {
	stubBlogShell(t, func(context.Context, string) (string, error) { return testBlogShell, nil })
	app := newBlogTestApp(t, models.Experience{
		ID: "55ba8524-46fb-4076-8559-16766d9b149d", Title: "Políticas como código", Summary: "Resumen del artículo",
		Body: "<p>Hola</p>", Tags: []string{"blog"}, Visibility: constants.VisibilityPublic,
		CreatedAt: "2026-11-02T00:00:00Z", UpdatedAt: "2026-11-02T00:00:00Z",
	})

	// Trailing slash as served by Firebase Hosting (trailingSlash: true).
	res, body := getBlogPage(t, app, "/blog/55BA8524-46fb-4076-8559-16766d9b149d/")
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, body = %s", res.StatusCode, body)
	}
	for _, want := range []string{
		"<title>Políticas como código | Blog</title>",
		`<meta property="og:title" content="Políticas como código">`,
		`<meta name="description" content="Resumen del artículo">`,
		`id="__BLOG_ARTICLE__"`,
		`/_next/static/chunks/app.js`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page does not contain %s", want)
		}
	}
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q", got)
	}
	if got := res.Header.Get("Cache-Control"); got != constants.BlogPageCacheControl {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := res.Header.Get("Cross-Origin-Embedder-Policy"); got != "unsafe-none" {
		t.Errorf("COEP = %q, images from the bucket would be blocked", got)
	}
}

func TestRenderArticleNotFoundForMissingPrivateOrNonBlogItems(t *testing.T) {
	stubBlogShell(t, func(context.Context, string) (string, error) { return testBlogShell, nil })
	app := newBlogTestApp(t,
		models.Experience{ID: "private", Title: "Borrador", Tags: []string{"blog"}, Visibility: constants.VisibilityPrivate},
		models.Experience{ID: "skill", Title: "Capacidad", Tags: []string{"skill"}, Visibility: constants.VisibilityPublic},
	)
	for _, path := range []string{"/blog/missing", "/blog/private", "/blog/skill"} {
		res, body := getBlogPage(t, app, path)
		if res.StatusCode != fiber.StatusNotFound {
			t.Errorf("%s: status = %d", path, res.StatusCode)
		}
		if !strings.Contains(body, `name="robots" content="noindex"`) || strings.Contains(body, "Borrador") {
			t.Errorf("%s: not-found page leaks content or is indexable: %s", path, body)
		}
	}
}

func TestRenderArticleBadGatewayWithoutShell(t *testing.T) {
	stubBlogShell(t, func(context.Context, string) (string, error) { return "", errors.New("down") })
	app := newBlogTestApp(t)
	if res, _ := getBlogPage(t, app, "/blog/x"); res.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("status = %d, want 502", res.StatusCode)
	}
}

func TestBlogShellCacheRefreshesAndFallsBackToLastGoodCopy(t *testing.T) {
	calls := 0
	fail := false
	stubBlogShell(t, func(context.Context, string) (string, error) {
		calls++
		if fail {
			return "", errors.New("front-end deploy in progress")
		}
		return testBlogShell, nil
	})
	cache := &blogShellCache{ttl: time.Hour}

	for range 2 {
		if got, err := cache.get(context.Background()); err != nil || got != testBlogShell {
			t.Fatalf("get = %q, %v", got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("fetched %d times within the TTL, want 1", calls)
	}

	cache.fetched = time.Now().Add(-2 * time.Hour) // expire
	fail = true
	if got, err := cache.get(context.Background()); err != nil || got != testBlogShell {
		t.Fatalf("expired + failed refresh should serve the cached copy, got %q, %v", got, err)
	}
	if calls != 2 {
		t.Fatalf("expected a refresh attempt after expiry, calls = %d", calls)
	}
}

func TestFetchBlogShell(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok/":
			_, _ = io.WriteString(w, testBlogShell)
		case "/not-html/":
			_, _ = io.WriteString(w, "plain text")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	if got, err := fetchBlogShell(context.Background(), srv.URL+"/ok/"); err != nil || got != testBlogShell {
		t.Errorf("ok: %q, %v", got, err)
	}
	for _, path := range []string{"/not-html/", "/missing/"} {
		if _, err := fetchBlogShell(context.Background(), srv.URL+path); err == nil {
			t.Errorf("%s: expected an error", path)
		}
	}
}
