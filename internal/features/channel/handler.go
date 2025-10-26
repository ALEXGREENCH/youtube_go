package channel

import (
	"net/http"
	"net/url"
	"strings"

	"youtube-mini/internal/features/subscriptions"
	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/features/watchlater"
	channelui "youtube-mini/internal/ui/channel"
	"youtube-mini/internal/youtube"
)

// Handler renders the channel screen.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := strings.TrimSpace(r.URL.Query().Get("id"))
		if channelID == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		info, sections, err := client.Channel(r.Context(), channelID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		queryTab := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("tab")))
		tabOrder := []channelui.Tab{
			{Key: "latest", Title: "Latest", Items: sections.Latest},
			{Key: "popular", Title: "Popular", Items: sections.Popular},
			{Key: "shorts", Title: "Shorts", Items: sections.Shorts},
			{Key: "live", Title: "Live", Items: sections.Live},
		}
		tabs := make([]channelui.Tab, 0, len(tabOrder))
		for _, tab := range tabOrder {
			if len(tab.Items) == 0 {
				continue
			}
			tabs = append(tabs, tab)
		}
		if len(tabs) == 0 {
			tabs = append(tabs, channelui.Tab{Key: "uploads", Title: "Uploads", Items: sections.Latest})
		}
		for ti := range tabs {
			for i := range tabs[ti].Items {
				item := &tabs[ti].Items[i]
				if strings.TrimSpace(item.Channel) == "" {
					item.Channel = info.Title
				}
				if strings.TrimSpace(item.ChannelID) == "" {
					item.ChannelID = info.ID
				}
			}
		}
		selectedTab := queryTab
		if selectedTab != "" {
			found := false
			for _, tab := range tabs {
				if tab.Key == selectedTab {
					found = true
					break
				}
			}
			if !found {
				selectedTab = ""
			}
		}
		if selectedTab == "" && len(tabs) > 0 {
			selectedTab = tabs[0].Key
		}

		watchLaterSet := watchlater.ReadSet(r)
		subscribedSet := subscriptions.ReadSet(r)
		activeNav := ""
		if subscribedSet[channelID] {
			activeNav = "subscriptions"
		}

		currentPath := r.URL.RequestURI()
		subscribeURL := ""
		unsubscribeURL := ""
		if channelID != "" {
			base := "id=" + url.QueryEscape(channelID) + "&return=" + url.QueryEscape(currentPath)
			if subscribedSet[channelID] {
				unsubscribeURL = "/subscriptions/remove?" + base
			} else {
				subscribeURL = "/subscriptions/add?" + base
			}
		}

		opts := channelui.PageData{
			ChannelID:      channelID,
			Title:          info.Title,
			AvatarURL:      info.AvatarURL,
			Subscribers:    info.Subscribers,
			Description:    info.Description,
			CurrentPath:    currentPath,
			Theme:          theme.FromRequest(r),
			WatchLater:     watchLaterSet,
			Subscribed:     subscribedSet,
			ActiveTab:      activeNav,
			SelectedTab:    selectedTab,
			IsSubscribed:   subscribedSet[channelID],
			SubscribeURL:   subscribeURL,
			UnsubscribeURL: unsubscribeURL,
			Tabs:           tabs,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(channelui.Render(opts)))
	}
}
