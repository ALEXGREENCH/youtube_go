package ui

import (
	"fmt"
	"net/url"
	"strings"

	"youtube-mini/internal/youtube"
)

// SearchPageOptions controls RenderSearch output.
type SearchPageOptions struct {
	Query       string
	CurrentPath string
	Results     []youtube.SearchResult
	WatchLater  map[string]bool
	Subscribed  map[string]bool
}

// RenderSearch renders the search results page with queue/watch-later actions.
func RenderSearch(opts SearchPageOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<!DOCTYPE html><html><head><meta charset="utf-8">
<title>Results: %s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body><div class="head"><a href="/">Back</a> YouTube Mini</div>
<div class="box">
<form action="/search" method="get"><input id="search-input" class="search-input" type="text" name="q" size="18" value="%s" list="suggestions" autofocus><input type="submit" value="Go"></form>
</div>
<div class="box"><b>Results for:</b> %s</div>`,
		Escape(opts.Query), Escape(opts.Query), Escape(opts.Query))

	if len(opts.Results) == 0 {
		b.WriteString("<p>No matches found.</p>")
	} else {
		for _, it := range opts.Results {
			queueURL := "/queue/add?id=" + url.QueryEscape(it.ID) + "&return=" + url.QueryEscape(opts.CurrentPath)
			watchLater := `<a href="/watchlater/add?id=` + url.QueryEscape(it.ID) + `&return=` + url.QueryEscape(opts.CurrentPath) + `">Watch later</a>`
			if opts.WatchLater != nil && opts.WatchLater[it.ID] {
				watchLater = `<a href="/watchlater/remove?id=` + url.QueryEscape(it.ID) + `&return=` + url.QueryEscape(opts.CurrentPath) + `">Remove from Watch later</a>`
			}

			subscribe := ""
			if it.ChannelID != "" {
				if opts.Subscribed != nil && opts.Subscribed[it.ChannelID] {
					subscribe = `<a href="/subscriptions/remove?id=` + url.QueryEscape(it.ChannelID) + `&return=` + url.QueryEscape(opts.CurrentPath) + `">Unsubscribe</a>`
				} else {
					subscribe = `<a href="/subscriptions/add?id=` + url.QueryEscape(it.ChannelID) + `&return=` + url.QueryEscape(opts.CurrentPath) + `">Subscribe</a>`
				}
			}

			fmt.Fprintf(&b, `<div class="vid">
<table cellspacing="0" cellpadding="2"><tr valign="top">
<td>
	<a href="/watch?v=%s">
		<div style="position:relative;display:inline-block;">
			<img src="%s" width="96" height="54" alt="">
			<div class="badge">%s</div>
		</div>
	</a>
</td>
<td>
	<b><a href="/watch?v=%s">%s</a></b><br>
	<small><a href="/channel?id=%s">%s</a></small><br>
	<small>%s</small><br>
	<small><a href="%s">Add to queue</a></small><br>
	<small>%s</small>`,
				Escape(it.ID), Escape(it.Thumbnail), Escape(it.Duration),
				Escape(it.ID), Escape(it.Title),
				Escape(it.ChannelID), Escape(it.Channel),
				Escape(it.Meta), queueURL, watchLater)
			if subscribe != "" {
				fmt.Fprintf(&b, `<br><small>%s</small>`, subscribe)
			}
			b.WriteString(`</td>
</tr></table>
</div>`)
		}
	}

	b.WriteString(`<hr><center><a href="/">Home</a></center>`)
	AppendSuggestionScript(&b)
	b.WriteString(`</body></html>`)
	return b.String()
}
