package watchlater

import (
	"fmt"
	"net/http"

	"youtube-mini/internal/features/subscriptions"
	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/ui"
	"youtube-mini/internal/youtube"
)

// Handler renders the watch later list.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := Read(r)
		items := make([]youtube.FeedItem, 0, len(ids))
		for _, id := range ids {
			video, err := client.GetVideo(r.Context(), id)
			if err != nil {
				continue
			}
			items = append(items, youtube.FeedItem{
				ID:        id,
				Title:     video.Title,
				Channel:   video.Author,
				ChannelID: video.ChannelID,
				Thumbnail: fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", id),
				Duration:  video.LengthSeconds,
				Meta:      "Saved for later",
			})
		}

		opts := ui.FeedPageOptions{
			Theme:       theme.FromRequest(r),
			Title:       "Watch Later",
			Subtitle:    "Videos you saved for later",
			ActiveTab:   "watchlater",
			CurrentPath: r.URL.RequestURI(),
			Items:       items,
			WatchLater:  ReadSet(r),
			Subscribed:  subscriptions.ReadSet(r),
			EmptyText:   "Use the Watch later button on any video to save it here.",
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderFeedPage(opts)))
	}
}

// AddHandler appends an id to watch later and redirects back.
func AddHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		Add(w, r, id)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

// RemoveHandler removes an id and redirects back.
func RemoveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		Remove(w, r, id)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

// ClearHandler wipes the watch later list.
func ClearHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Clear(w)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

func redirectTarget(r *http.Request) string {
	target := r.URL.Query().Get("return")
	if target == "" {
		target = "/watchlater"
	}
	return target
}
