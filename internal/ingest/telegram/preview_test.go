package telegram

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParsePreview(t *testing.T) {
	html, err := os.ReadFile("testdata/preview.html")
	if err != nil {
		t.Fatal(err)
	}

	posts, err := ParsePreview("hrlunapark", html)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("posts = %d, want 3", len(posts))
	}

	p := posts[1] // hrlunapark/392 — the Claimsorted multi-vacancy post
	if p.MsgID != 392 {
		t.Errorf("msg_id = %d, want 392", p.MsgID)
	}
	want := time.Date(2026, 5, 28, 12, 3, 7, 0, time.UTC)
	if !p.PostedAt.Equal(want) {
		t.Errorf("posted_at = %v, want %v", p.PostedAt, want)
	}
	if !strings.Contains(p.Text, "ML & full-stack engineers") {
		t.Errorf("text lacks the tl;dr line; got %.120q", p.Text)
	}
	// Entities decoded, tags stripped, <br> became newlines.
	if strings.Contains(p.Text, "<") || strings.Contains(p.Text, "&amp;") {
		t.Errorf("text still contains markup: %.200q", p.Text)
	}
	if !strings.Contains(p.Text, "\n") {
		t.Error("text has no newlines — <br> separation lost")
	}
}

func TestParsePreviewYieldsPostsInPageOrder(t *testing.T) {
	html, _ := os.ReadFile("testdata/preview.html")
	posts, err := ParsePreview("hrlunapark", html)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(posts); i++ {
		if posts[i].MsgID <= posts[i-1].MsgID {
			t.Errorf("posts out of order: %d after %d", posts[i].MsgID, posts[i-1].MsgID)
		}
	}
}

func TestParsePreviewCapturesMessageLinks(t *testing.T) {
	// A digest post: the teaser text plus outbound vacancy links whose hrefs nodeText
	// would otherwise drop. The links must survive for the link-following step.
	const page = `<html><body>
<div class="tgme_widget_message" data-post="habr_career/75410">
  <time datetime="2026-06-13T10:19:00+00:00">10:19</time>
  <div class="tgme_widget_message_text">Работа для начинающих на Хабр Карьере.
    <a href="https://u.habr.com/PnBO7">Product manager в СберЗдоровье</a>.
    <a href="https://u.habr.com/fq3n5">Больше вакансий</a>
  </div>
</div></body></html>`

	posts, err := ParsePreview("habr_career", []byte(page))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("posts = %d, want 1", len(posts))
	}
	links := posts[0].Links
	if len(links) != 2 {
		t.Fatalf("links = %d (%+v), want 2", len(links), links)
	}
	if links[0].URL != "https://u.habr.com/PnBO7" || links[0].Text != "Product manager в СберЗдоровье" {
		t.Errorf("links[0] = %+v", links[0])
	}
	if links[1].URL != "https://u.habr.com/fq3n5" || links[1].Text != "Больше вакансий" {
		t.Errorf("links[1] = %+v", links[1])
	}
	// The plain text stays clean (no raw hrefs leaked into it).
	if strings.Contains(posts[0].Text, "u.habr.com") {
		t.Errorf("text leaked a href: %q", posts[0].Text)
	}
}

func TestParsePreviewZeroPostsIsAnError(t *testing.T) {
	// A page with no recognizable posts means the markup drifted or the channel
	// has no preview — the caller must count the channel failed, loudly.
	_, err := ParsePreview("foo", []byte("<html><body>nothing here</body></html>"))
	if err == nil {
		t.Fatal("want error for a page with zero parseable posts")
	}
}
