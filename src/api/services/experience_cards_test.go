package services

import (
	"strings"
	"testing"

	userModel "backend-yonathan/src/models"
)

func TestToCardViewDropsBodyAndDerivesSummaryAndCover(t *testing.T) {
	item := userModel.Experience{
		ID:   "1",
		Body: `<p>Hola   <strong>mundo</strong></p><img src="https://x/y.jpg" alt="">`,
	}
	got := toCardView(item)
	if got.Body != "" {
		t.Fatalf("body must be dropped, got %q", got.Body)
	}
	if got.Summary != "Hola mundo" {
		t.Fatalf("unexpected summary %q", got.Summary)
	}
	if len(got.ImageURLs) != 1 || got.ImageURLs[0] != "https://x/y.jpg" {
		t.Fatalf("unexpected cover %v", got.ImageURLs)
	}
}

func TestToCardViewKeepsExistingSummaryAndImages(t *testing.T) {
	item := userModel.Experience{Summary: "resumen", ImageURLs: []string{"a"}, Body: `<img src="b">`}
	got := toCardView(item)
	if got.Summary != "resumen" || len(got.ImageURLs) != 1 || got.ImageURLs[0] != "a" {
		t.Fatalf("existing values must be kept: %+v", got)
	}
}

func TestToCardViewTruncatesLongSummary(t *testing.T) {
	got := toCardView(userModel.Experience{Body: "<p>" + strings.Repeat("palabra ", 100) + "</p>"})
	if r := []rune(got.Summary); len(r) > cardSummaryRunes+1 || !strings.HasSuffix(got.Summary, "…") {
		t.Fatalf("summary not truncated: %d runes", len(r))
	}
}

func TestThumbObjectPath(t *testing.T) {
	cases := map[string]string{
		"portfolio-images/abc.jpg":             "portfolio-images/abc-w640.jpg",
		"https://h/b/portfolio-images/abc.png": "https://h/b/portfolio-images/abc-w640.png",
		"portfolio-images/abc-w640.jpg":        "",
		"portfolio-images/noext":               "",
	}
	for in, want := range cases {
		if got := thumbObjectPath(in); got != want {
			t.Errorf("thumbObjectPath(%q)=%q want %q", in, got, want)
		}
	}
}
