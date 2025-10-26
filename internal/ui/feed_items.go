package ui

import (
	"fmt"
	"net/url"
	"strings"

	"youtube-mini/internal/youtube"
)

// FeedItemsOptions configures rendering of feed items.
type FeedItemsOptions struct {
	Items          []youtube.FeedItem
	CurrentPath    string
	WatchLater     map[string]bool
	Subscribed     map[string]bool
	ShowQueue      bool
	ShowWatchLater bool
	ShowSubscribe  bool
}

type feedAction struct {
	href  string
	label string
}

// RenderFeedItems renders a list of feed cards.
func RenderFeedItems(opts FeedItemsOptions) string {
	if len(opts.Items) == 0 {
		return ""
	}
	if opts.WatchLater == nil {
		opts.WatchLater = map[string]bool{}
	}
	if opts.Subscribed == nil {
		opts.Subscribed = map[string]bool{}
	}
	sb := &strings.Builder{}
	for _, it := range opts.Items {
		watchURL := "/watch?v=" + url.QueryEscape(it.ID)
		channelURL := ""
		if it.ChannelID != "" {
			channelURL = "/channel?id=" + url.QueryEscape(it.ChannelID)
		}

		actions := make([]feedAction, 0, 3)
		if opts.ShowQueue {
			actions = append(actions, feedAction{
				href:  "/queue/add?id=" + url.QueryEscape(it.ID) + "&return=" + url.QueryEscape(opts.CurrentPath),
				label: "Add to queue",
			})
		}
		if opts.ShowWatchLater {
			if opts.WatchLater[it.ID] {
				actions = append(actions, feedAction{
					href:  "/watchlater/remove?id=" + url.QueryEscape(it.ID) + "&return=" + url.QueryEscape(opts.CurrentPath),
					label: "Remove from Watch later",
				})
			} else {
				actions = append(actions, feedAction{
					href:  "/watchlater/add?id=" + url.QueryEscape(it.ID) + "&return=" + url.QueryEscape(opts.CurrentPath),
					label: "Watch later",
				})
			}
		}
		if opts.ShowSubscribe && it.ChannelID != "" {
			if opts.Subscribed[it.ChannelID] {
				actions = append(actions, feedAction{
					href:  "/subscriptions/remove?id=" + url.QueryEscape(it.ChannelID) + "&return=" + url.QueryEscape(opts.CurrentPath),
					label: "Unsubscribe",
				})
			} else {
				actions = append(actions, feedAction{
					href:  "/subscriptions/add?id=" + url.QueryEscape(it.ChannelID) + "&return=" + url.QueryEscape(opts.CurrentPath),
					label: "Subscribe",
				})
			}
		}

		sb.WriteString(`<article class="feed-card">`)
		fmt.Fprintf(sb, `<a class="feed-thumb" href="%s">`, EscapeAttr(watchURL))
		fmt.Fprintf(sb, `<img src="%s" width="168" height="94" alt="">`, EscapeAttr(ProxiedImage(it.Thumbnail)))
		if it.Duration != "" {
			fmt.Fprintf(sb, `<span class="badge">%s</span>`, Escape(it.Duration))
		}
		sb.WriteString(`</a>`)

		sb.WriteString(`<div class="feed-body">`)
		fmt.Fprintf(sb, `<h3 class="feed-title"><a href="%s">%s</a></h3>`, EscapeAttr(watchURL), Escape(it.Title))
		if it.Channel != "" {
			if channelURL != "" {
				fmt.Fprintf(sb, `<div class="feed-channel"><a href="%s">%s</a></div>`, EscapeAttr(channelURL), Escape(it.Channel))
			} else {
				fmt.Fprintf(sb, `<div class="feed-channel">%s</div>`, Escape(it.Channel))
			}
		}
		if it.Meta != "" {
			fmt.Fprintf(sb, `<div class="feed-meta">%s</div>`, Escape(it.Meta))
		}
		if len(actions) > 0 {
			sb.WriteString(`<div class="feed-actions">`)
			for i, action := range actions {
				if i > 0 {
					sb.WriteString(`<span class="feed-dot">&middot;</span>`)
				}
				fmt.Fprintf(sb, `<a href="%s">%s</a>`, EscapeAttr(action.href), Escape(action.label))
			}
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div></article>`)
	}
	return sb.String()
}

// EscapeAttr escapes a string for use in attribute values.
func EscapeAttr(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}
