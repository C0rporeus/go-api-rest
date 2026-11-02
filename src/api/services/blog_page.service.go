package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository"

	"github.com/gofiber/fiber/v3"
)

const maxBlogShellBytes = 2 << 20

// fetchBlogShellFunc downloads the static article shell (front-end
// pages/blog/articulo). Injectable for tests (see shared_helpers.go).
var fetchBlogShellFunc = fetchBlogShell

// BlogPageService renders /blog/:id on every request from the database: the
// static Next.js shell with the article's title, description, canonical URL,
// Open Graph and Twitter tags, plus the article data embedded for the browser.
// Firebase Hosting rewrites /blog/** here, so crawlers that do not run
// JavaScript always see the current article, without rebuilding the site.
type BlogPageService struct {
	repo  repository.ExperienceRepository
	shell *blogShellCache
}

// NewBlogPageService creates the service.
func NewBlogPageService(repo repository.ExperienceRepository) *BlogPageService {
	return &BlogPageService{repo: repo, shell: &blogShellCache{ttl: constants.BlogShellTTL}}
}

// RenderArticle handles GET /blog/:id.
func (s *BlogPageService) RenderArticle(c fiber.Ctx) error {
	ctx := c.Context()
	shell, err := s.shell.get(ctx)
	if err != nil {
		log.Printf("blog page: shell unavailable: %v", err)
		return c.Status(fiber.StatusBadGateway).SendString("Blog temporarily unavailable")
	}

	// The HTML document loads images from the private bucket through signed
	// cross-origin URLs; the API-wide helmet defaults (COEP require-corp,
	// CORP same-origin) are meant for JSON and would block them.
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	c.Set("Cross-Origin-Embedder-Policy", "unsafe-none")
	c.Set("Cross-Origin-Resource-Policy", "same-site")
	c.Set(fiber.HeaderCacheControl, constants.BlogPageCacheControl)

	id := strings.ToLower(strings.TrimSpace(c.Params("id")))
	item, err := s.repo.GetByID(ctx, id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("blog page: loading article %q: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).SendString("Could not load the article")
	}
	if err != nil || !isPublicBlogArticle(item) {
		// The shell renders its own "not found" message; crawlers get a real 404.
		notFound := injectIntoShell(shell, "Artículo no encontrado | Blog", `<meta name="robots" content="noindex">`)
		return c.Status(fiber.StatusNotFound).SendString(notFound)
	}

	// og:image needs the canonical object URL; the embedded copy needs signed
	// URLs so the browser can load the private images.
	shareImage := blogShareImage(item)
	embedded := item
	embedded.ImageURLs = append([]string(nil), item.ImageURLs...)
	SignExperienceImageURLs(ctx, &embedded)

	head, err := blogHeadTags(item, shareImage, embedded)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Could not render the article")
	}
	return c.SendString(injectIntoShell(shell, item.Title+" | Blog", head))
}

// blogShellCache keeps the shell for a short TTL. Front-end deploys change the
// shell's asset hashes, so it must be refreshed often; on a failed refresh the
// last good copy keeps the blog up.
type blogShellCache struct {
	mu      sync.RWMutex
	html    string
	fetched time.Time
	ttl     time.Duration
}

func (sc *blogShellCache) get(ctx context.Context) (string, error) {
	sc.mu.RLock()
	cached, fresh := sc.html, time.Since(sc.fetched) < sc.ttl
	sc.mu.RUnlock()
	if cached != "" && fresh {
		return cached, nil
	}

	shell, err := fetchBlogShellFunc(ctx, constants.BlogShellURL())
	if err != nil {
		if cached != "" {
			log.Printf("blog page: refreshing shell failed, serving cached copy: %v", err)
			return cached, nil
		}
		return "", err
	}

	sc.mu.Lock()
	sc.html, sc.fetched = shell, time.Now()
	sc.mu.Unlock()
	return shell, nil
}

func fetchBlogShell(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBlogShellBytes))
	if err != nil {
		return "", err
	}
	if !strings.Contains(strings.ToLower(string(body)), "</head>") {
		return "", errors.New("blog shell is not an HTML document")
	}
	return string(body), nil
}
