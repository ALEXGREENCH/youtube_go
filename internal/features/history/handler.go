package history

import (
	"fmt"
	"net/http"
	"time"

	"youtube-mini/internal/features/subscriptions"
	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/features/watchlater"
	"youtube-mini/internal/ui"
	"youtube-mini/internal/youtube"
)

// Handler renders the watch history page.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := Read(r)
		items := make([]youtube.FeedItem, 0, len(entries))
		for _, entry := range entries {
			video, err := client.GetVideo(r.Context(), entry.VideoID)
			if err != nil {
				continue
			}
			meta := "Watched " + relativeTime(entry.SeenAt)
			items = append(items, youtube.FeedItem{
				ID:        entry.VideoID,
				Title:     video.Title,
				Channel:   video.Author,
				ChannelID: video.ChannelID,
				Thumbnail: fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", entry.VideoID),
				Duration:  video.LengthSeconds,
				Meta:      meta,
			})
		}

		opts := ui.FeedPageOptions{
			Theme:       theme.FromRequest(r),
			Title:       "History",
			Subtitle:    "Recently watched videos",
			ActiveTab:   "history",
			CurrentPath: r.URL.RequestURI(),
			Items:       items,
			WatchLater:  watchlater.ReadSet(r),
			Subscribed:  subscriptions.ReadSet(r),
			EmptyText:   "Start watching videos and they will show up here.",
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderFeedPage(opts)))
	}
}

// ClearHandler wipes watch history.
func ClearHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Clear(w)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

func redirectTarget(r *http.Request) string {
	target := r.URL.Query().Get("return")
	if target == "" {
		target = "/history"
	}
	return target
}

func relativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}
	return t.Format("02 Jan 2006")
}
