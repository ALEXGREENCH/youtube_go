package channel

import (
	"fmt"
	"net/url"
	"strings"

	"youtube-mini/internal/ui"
	"youtube-mini/internal/youtube"
)

// PageData aggregates channel metadata and content rows.
type PageData struct {
	ChannelID      string
	Title          string
	AvatarURL      string
	Subscribers    string
	Description    string
	Tabs           []Tab
	CurrentPath    string
	Theme          string
	WatchLater     map[string]bool
	Subscribed     map[string]bool
	ActiveTab      string
	SelectedTab    string
	IsSubscribed   bool
	SubscribeURL   string
	UnsubscribeURL string
}

// Tab describes a row of items.
type Tab struct {
	Key   string
	Title string
	Items []youtube.FeedItem
}

// Render builds the channel page HTML.
func Render(data PageData) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<!DOCTYPE html><html><head><meta charset="utf-8">
<title>%s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body class="%s">`, ui.Escape(data.Title), ui.ThemeBodyClass(data.Theme))

	b.WriteString(ui.RenderHeader(data.ActiveTab, data.Theme, data.CurrentPath, ""))
	b.WriteString(`<main class="page channel-page">`)
	b.WriteString(`<div class="box">`)
	if data.AvatarURL != "" {
		fmt.Fprintf(&b, `<img src="%s" width="64" height="64" style="border-radius:50%%" alt=""> `, ui.EscapeAttr(ui.ProxiedImage(data.AvatarURL)))
	}
	fmt.Fprintf(&b, `<b>%s</b>`, ui.Escape(data.Title))
	if data.Subscribers != "" {
		fmt.Fprintf(&b, `<br><small>%s</small>`, ui.Escape(data.Subscribers))
	}
	if data.Description != "" {
		fmt.Fprintf(&b, `<p>%s</p>`, ui.Escape(data.Description))
	}
	if data.IsSubscribed {
		if data.UnsubscribeURL != "" {
			fmt.Fprintf(&b, `<p><a href="%s">Unsubscribe</a></p>`, ui.EscapeAttr(data.UnsubscribeURL))
		}
	} else if data.SubscribeURL != "" {
		fmt.Fprintf(&b, `<p><a class="btn" href="%s">Subscribe</a></p>`, ui.EscapeAttr(data.SubscribeURL))
	}
	b.WriteString(`</div>`)

	selected := data.SelectedTab
	activeTitle := ""
	activeItems := []youtube.FeedItem{}
	for _, tab := range data.Tabs {
		key := tab.Key
		if key == "" {
			key = strings.ToLower(strings.ReplaceAll(tab.Title, " ", "-"))
		}
		if selected == "" {
			selected = key
		}
		if key == selected && len(activeItems) == 0 {
			activeTitle = tab.Title
			activeItems = tab.Items
		}
	}
	if len(activeItems) == 0 && len(data.Tabs) > 0 {
		first := data.Tabs[0]
		selected = first.Key
		if selected == "" {
			selected = strings.ToLower(strings.ReplaceAll(first.Title, " ", "-"))
		}
		activeTitle = first.Title
		activeItems = first.Items
	}

	if len(data.Tabs) > 1 {
		b.WriteString(`<div class="channel-tabs">`)
		for _, tab := range data.Tabs {
			key := tab.Key
			if key == "" {
				key = strings.ToLower(strings.ReplaceAll(tab.Title, " ", "-"))
			}
			class := ""
			if key == selected {
				class = ` class="on"`
			}
			values := url.Values{}
			values.Set("id", data.ChannelID)
			if key != "" {
				values.Set("tab", key)
			}
			href := "/channel?" + values.Encode()
			fmt.Fprintf(&b, `<a href="%s"%s>%s</a>`, ui.EscapeAttr(href), class, ui.Escape(tab.Title))
		}
		b.WriteString(`</div>`)
	}

	if activeTitle != "" {
		fmt.Fprintf(&b, `<div class="box"><b>%s</b></div>`, ui.Escape(activeTitle))
	}

	if len(activeItems) == 0 {
		b.WriteString(`<p class="empty">No videos yet.</p>`)
	} else {
		b.WriteString(ui.RenderFeedItems(ui.FeedItemsOptions{
			Items:          activeItems,
			CurrentPath:    data.CurrentPath,
			WatchLater:     data.WatchLater,
			Subscribed:     data.Subscribed,
			ShowQueue:      true,
			ShowWatchLater: true,
			ShowSubscribe:  true,
		}))
	}

	b.WriteString(`<hr><div class="footer-link"><a href="/">Home</a></div>`)
	b.WriteString(`</main>`)
	ui.AppendSuggestionScript(&b)
	b.WriteString(`</body></html>`)
	return b.String()
}
