package ui

import (
	"fmt"
	"net/url"
	"strings"

	"youtube-mini/internal/youtube"
)

// FeedPageOptions controls RenderFeedPage output.
type FeedPageOptions struct {
	Theme       string
	Title       string
	Subtitle    string
	ActiveTab   string
	CurrentPath string
	Items       []youtube.FeedItem
	Subscribed  map[string]bool
	WatchLater  map[string]bool
	EmptyText   string
}

// RenderFeedPage renders feed-style grids such as watch later or subscriptions.
func RenderFeedPage(opts FeedPageOptions) string {
	if opts.EmptyText == "" {
		opts.EmptyText = "No videos available right now."
	}

	itemsHTML := RenderFeedItems(FeedItemsOptions{
		Items:          opts.Items,
		CurrentPath:    opts.CurrentPath,
		WatchLater:     opts.WatchLater,
		Subscribed:     opts.Subscribed,
		ShowQueue:      true,
		ShowWatchLater: true,
		ShowSubscribe:  true,
	})

	var b strings.Builder
	fmt.Fprintf(&b, `<!DOCTYPE html><html><head><meta charset="utf-8">
<title>%s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body class="%s">`, Escape(opts.Title), ThemeBodyClass(opts.Theme))

	b.WriteString(RenderHeader(opts.ActiveTab, opts.Theme, opts.CurrentPath, ""))
	b.WriteString(`<main class="page">`)

	if opts.Subtitle != "" {
		fmt.Fprintf(&b, `<div class="box"><b>%s</b></div>`, Escape(opts.Subtitle))
	}

	if itemsHTML == "" {
		fmt.Fprintf(&b, `<p class="empty">%s</p>`, Escape(opts.EmptyText))
	} else {
		b.WriteString(itemsHTML)
	}

	b.WriteString(`<hr><div class="footer-link"><a href="/features">Feature roadmap</a></div>`)
	b.WriteString(`</main>`)
	AppendSuggestionScript(&b)
	b.WriteString(`</body></html>`)
	return b.String()
}

func RenderHeader(activeTab, theme, currentPath, query string) string {
	var b strings.Builder
	nextTheme := NextTheme(theme)
	label := "Switch to Light theme"
	if nextTheme == "dark" {
		label = "Switch to Dark theme"
	}
	b.WriteString(`<header class="head">`)
	b.WriteString(`<a class="head-logo" href="/"><img src="/static/youtube.png" alt="YouTube Mini"></a>`)
	b.WriteString(`<form class="head-search" action="/" method="get">`)
	b.WriteString(`<input id="search-input" class="search-input" type="text" name="q" placeholder="Search YouTube..." value="` + EscapeAttr(query) + `" list="suggestions">`)
	b.WriteString(`<button type="submit" class="search-submit" aria-label="Search">Search</button>`)
	b.WriteString(`</form>`)
	b.WriteString(`<div class="head-actions"><a class="head-theme" href="/theme?name=` + nextTheme + `&return=` + url.QueryEscape(currentPath) + `">` + Escape(label) + `</a></div>`)
	b.WriteString(`</header>`)

	b.WriteString(`<div class="tabs"><a href="/"`)
	if activeTab == "home" {
		b.WriteString(` class="on"`)
	}
	b.WriteString(`>Home</a><a href="/?tab=explore"`)
	if activeTab == "explore" {
		b.WriteString(` class="on"`)
	}
	b.WriteString(`>Explore</a><a href="/subscriptions"`)
	if activeTab == "subscriptions" {
		b.WriteString(` class="on"`)
	}
	b.WriteString(`>Subscriptions</a><a href="/watchlater"`)
	if activeTab == "watchlater" {
		b.WriteString(` class="on"`)
	}
	b.WriteString(`>Watch later</a><a href="/history"`)
	if activeTab == "history" {
		b.WriteString(` class="on"`)
	}
	b.WriteString(`>History</a></div>`)

	return b.String()
}
