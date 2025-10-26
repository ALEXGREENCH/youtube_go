package watch

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"youtube-mini/internal/features/history"
	"youtube-mini/internal/features/queue"
	"youtube-mini/internal/features/settings"
	"youtube-mini/internal/features/subscriptions"
	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/features/watchlater"
	"youtube-mini/internal/ui"
	"youtube-mini/internal/youtube"
)

// Handler renders the watch page with related videos, queue controls, watch later, and autoplay.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("v"))
		if id == "" {
			http.Error(w, "usage: /watch?v=<id>", http.StatusBadRequest)
			return
		}

		queueIDs := queue.Read(r)
		if r.URL.Query().Get("dequeue") == "1" {
			queue.Remove(w, r, id)
			queueIDs = removeID(queueIDs, id)
		}

		video, err := client.GetVideo(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		history.Record(w, r, id)

		relatedItems, autoplayRelated, _ := client.Next(r.Context(), id)

		autoplayEnabled := settings.ReadAutoplay(r)
		nextFromQueue := nextQueueID(queueIDs, id)
		autoplayURL, autoplaySource := computeAutoplayTarget(nextFromQueue, autoplayRelated, autoplayEnabled)
		if autoplayURL != "" && strings.Contains(autoplayURL, id) {
			autoplayURL = ""
		}
		if autoplayURL != "" && autoplaySource == "queue" {
			autoplayURL = autoplayURL + "&dequeue=1"
		}

		returnPath := r.URL.RequestURI()

		relatedEntries := make([]ui.RelatedEntry, 0, len(relatedItems))
		for _, item := range relatedItems {
			addURL := "/queue/add?id=" + url.QueryEscape(item.ID) + "&return=" + url.QueryEscape(returnPath)
			relatedEntries = append(relatedEntries, ui.RelatedEntry{Item: item, AddToQueueURL: addURL})
		}

		queueEntries := buildQueueEntries(queueIDs, id, returnPath, client, r)

		subscribedSet := subscriptions.ReadSet(r)
		watchLaterSet := watchlater.ReadSet(r)

		subscribeURL := ""
		unsubscribeURL := ""
		if video.ChannelID != "" {
			base := "id=" + url.QueryEscape(video.ChannelID) + "&return=" + url.QueryEscape(returnPath)
			if subscribedSet[video.ChannelID] {
				unsubscribeURL = "/subscriptions/remove?" + base
			} else {
				subscribeURL = "/subscriptions/add?" + base
			}
		}

		watchLaterURL := ""
		if watchLaterSet[id] {
			watchLaterURL = "/watchlater/remove?id=" + url.QueryEscape(id) + "&return=" + url.QueryEscape(returnPath)
		} else {
			watchLaterURL = "/watchlater/add?id=" + url.QueryEscape(id) + "&return=" + url.QueryEscape(returnPath)
		}

		data := ui.WatchPageData{
			Theme:       theme.FromRequest(r),
			CurrentPath: returnPath,
			Video:       video,
			StreamURL:   fmt.Sprintf("/stream/%s.mp4", video.ID),
			AudioURL:    fmt.Sprintf("/stream/audio/%s.mp3", video.ID),
			TranscodeLinks: []ui.Link{
				{Label: "MP4 240p (AAC)", URL: fmt.Sprintf("/stream/ffmpeg/%s.mp4?aac", video.ID)},
				{Label: "3GP 144p (Retro)", URL: fmt.Sprintf("/stream/ffmpeg/%s.mp4?retro", video.ID)},
				{Label: "3GP 96p (Edge)", URL: fmt.Sprintf("/stream/ffmpeg/%s.mp4?edge", video.ID)},
			},
			Captions:          video.Captions,
			AutoplayEnabled:   autoplayEnabled,
			AutoplayToggleURL: "/settings/autoplay?return=" + url.QueryEscape(returnPath),
			AutoplayNextURL:   autoplayURL,
			AutoplaySource:    autoplaySource,
			Related:           relatedEntries,
			Queue:             queueEntries,
			QueueClearURL:     "/queue/clear?return=" + url.QueryEscape(returnPath),
			Subscribed:        subscribedSet[video.ChannelID],
			SubscribeURL:      subscribeURL,
			UnsubscribeURL:    unsubscribeURL,
			InWatchLater:      watchLaterSet[id],
			WatchLaterURL:     watchLaterURL,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderWatch(data)))
	}
}

func removeID(list []string, id string) []string {
	filtered := make([]string, 0, len(list))
	for _, it := range list {
		if it == id {
			continue
		}
		filtered = append(filtered, it)
	}
	return filtered
}

func nextQueueID(queueIDs []string, current string) string {
	for _, id := range queueIDs {
		if id == current {
			continue
		}
		return id
	}
	return ""
}

func computeAutoplayTarget(queueID, relatedID string, autoplayEnabled bool) (string, string) {
	if !autoplayEnabled {
		return "", ""
	}
	if queueID != "" {
		return "/watch?v=" + url.QueryEscape(queueID), "queue"
	}
	if relatedID != "" {
		return "/watch?v=" + url.QueryEscape(relatedID), "related"
	}
	return "", ""
}

func buildQueueEntries(queueIDs []string, currentID, returnPath string, client *youtube.Client, r *http.Request) []ui.QueueEntry {
	entries := make([]ui.QueueEntry, 0, len(queueIDs))
	ctx := r.Context()
	seen := make(map[string]struct{})
	for _, id := range queueIDs {
		if id == currentID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		video, err := client.GetVideo(ctx, id)
		if err != nil {
			continue
		}
		item := youtube.FeedItem{
			ID:        id,
			Title:     video.Title,
			Channel:   video.Author,
			ChannelID: video.ChannelID,
			Thumbnail: fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", id),
			Meta:      "Queued",
		}
		removeURL := "/queue/remove?id=" + url.QueryEscape(id) + "&return=" + url.QueryEscape(returnPath)
		entries = append(entries, ui.QueueEntry{Item: item, RemoveURL: removeURL})
		if len(entries) >= 10 {
			break
		}
	}
	return entries
}
