package subscriptions

import (
	"fmt"
	"net/http"
	"strings"

	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/ui"
	"youtube-mini/internal/youtube"
)

// Handler renders the subscriptions feed based on stored channel ids.
func Handler(client *youtube.Client, readWatchLater func(*http.Request) map[string]bool) http.HandlerFunc {
	if readWatchLater == nil {
		readWatchLater = func(*http.Request) map[string]bool { return nil }
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ids := Read(r)
		items := make([]youtube.FeedItem, 0, len(ids)*5)
		channelTitles := make(map[string]string, len(ids))
		ctx := r.Context()
		for _, id := range ids {
			channelItems, err := client.ChannelFeed(ctx, id)
			if err != nil || len(channelItems) == 0 {
				continue
			}
			needsTitle := false
			for i := range channelItems {
				if strings.TrimSpace(channelItems[i].Channel) == "" {
					needsTitle = true
					break
				}
			}
			title := channelTitles[id]
			if needsTitle && title == "" {
				if info, _, err := client.Channel(ctx, id); err == nil {
					title = info.Title
					channelTitles[id] = title
				}
			}
			for i := range channelItems {
				item := &channelItems[i]
				if strings.TrimSpace(item.Channel) == "" && title != "" {
					item.Channel = title
				}
				if strings.TrimSpace(item.ChannelID) == "" {
					item.ChannelID = id
				}
			}
			for i, item := range channelItems {
				items = append(items, item)
				if i >= 4 {
					break
				}
			}
		}

		opts := ui.FeedPageOptions{
			Theme:       theme.FromRequest(r),
			Title:       "Subscriptions",
			Subtitle:    fmt.Sprintf("Showing updates from %d channels", len(ids)),
			ActiveTab:   "subscriptions",
			CurrentPath: r.URL.RequestURI(),
			Items:       items,
			Subscribed:  ReadSet(r),
			WatchLater:  readWatchLater(r),
			EmptyText:   "Subscribe to channels from the watch page to see them here.",
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderFeedPage(opts)))
	}
}

// AddHandler subscribes to a channel and redirects back.
func AddHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		Add(w, r, id)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

// RemoveHandler unsubscribes from a channel.
func RemoveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		Remove(w, r, id)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

func redirectTarget(r *http.Request) string {
	target := r.URL.Query().Get("return")
	if strings.TrimSpace(target) == "" {
		target = "/subscriptions"
	}
	return target
}
