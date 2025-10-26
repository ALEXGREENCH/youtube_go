package ui

import (
	"fmt"
	"strings"

	"youtube-mini/internal/youtube"
)

// HomePageData represents the unified home/search page content.
type HomePageData struct {
	Theme         string
	ActiveTab     string
	Query         string
	CurrentPath   string
	HomeItems     []youtube.FeedItem
	SearchResults []youtube.SearchResult
	WatchLater    map[string]bool
	Subscribed    map[string]bool
}

// RenderHomePage renders the combined home/search experience.
func RenderHomePage(data HomePageData) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<!DOCTYPE html><html><head><meta charset="utf-8">
<title>YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body class="%s">`, ThemeBodyClass(data.Theme))

	b.WriteString(RenderHeader(data.ActiveTab, data.Theme, data.CurrentPath, data.Query))
	b.WriteString(`<main class="page">`)

	if strings.TrimSpace(data.Query) != "" {
		renderSearchResults(&b, data)
	} else {
		if len(data.HomeItems) == 0 {
			b.WriteString(`<p class="empty">No recommendations yet. Try searching for something!</p>`)
		} else {
			itemsHTML := RenderFeedItems(FeedItemsOptions{
				Items:          data.HomeItems,
				CurrentPath:    data.CurrentPath,
				WatchLater:     data.WatchLater,
				Subscribed:     data.Subscribed,
				ShowQueue:      true,
				ShowWatchLater: true,
				ShowSubscribe:  true,
			})
			b.WriteString(itemsHTML)
		}
	}

	b.WriteString(`<hr><div class="footer-link"><a href="/features">Feature roadmap</a></div>`)
	b.WriteString(`</main>`)
	AppendSuggestionScript(&b)
	b.WriteString(`</body></html>`)
	return b.String()
}

func renderSearchResults(b *strings.Builder, data HomePageData) {
	fmt.Fprintf(b, `<div class="box"><b>Results for:</b> %s</div>`, Escape(data.Query))
	if len(data.SearchResults) == 0 {
		b.WriteString(`<p>No matches found.</p>`)
		return
	}

	results := make([]youtube.FeedItem, 0, len(data.SearchResults))
	for _, it := range data.SearchResults {
		results = append(results, youtube.FeedItem{
			ID:        it.ID,
			Title:     it.Title,
			Channel:   it.Channel,
			ChannelID: it.ChannelID,
			Thumbnail: it.Thumbnail,
			Duration:  it.Duration,
			Meta:      it.Meta,
		})
	}
	b.WriteString(RenderFeedItems(FeedItemsOptions{
		Items:          results,
		CurrentPath:    data.CurrentPath,
		WatchLater:     data.WatchLater,
		Subscribed:     data.Subscribed,
		ShowQueue:      true,
		ShowWatchLater: true,
		ShowSubscribe:  true,
	}))
}
