package home

import (
	"net/http"
	"strings"

	"youtube-mini/internal/features/subscriptions"
	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/features/watchlater"
	"youtube-mini/internal/ui"
	"youtube-mini/internal/youtube"
)

// Handler serves the unified home/search page.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		themeID := theme.FromRequest(r)
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	activeTab := "home"

		var feedItems []youtube.FeedItem
		var searchResults []youtube.SearchResult
		var err error

	if query != "" {
		activeTab = ""
		searchResults, err = client.Search(r.Context(), query)
	} else {
			switch tab {
			case "explore":
				feedItems, err = client.Trending(r.Context())
				activeTab = "explore"
			case "home", "":
				feedItems, err = client.Home(r.Context())
				activeTab = "home"
			default:
				feedItems, err = client.Home(r.Context())
				activeTab = "home"
			}
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		opts := ui.HomePageData{
			Theme:        themeID,
			ActiveTab:    activeTab,
			Query:        query,
			CurrentPath:  r.URL.RequestURI(),
			HomeItems:    feedItems,
			SearchResults: searchResults,
			WatchLater:   watchlater.ReadSet(r),
			Subscribed:   subscriptions.ReadSet(r),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderHomePage(opts)))
	}
}
