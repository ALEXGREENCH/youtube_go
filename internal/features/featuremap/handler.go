package featuremap

import (
	"net/http"
	"strings"
)

type feature struct {
	Name        string
	Status      string
	Description string
}

type category struct {
	Title    string
	Features []feature
}

var catalogue = []category{
	{
		Title: "Discovery & Navigation",
		Features: []feature{
			{"Home recommendations", "Planned", "Surface personalised mixes similar to YouTube Home."},
			{"Trending / Explore", "Planned", "Regional charts and topic shelves."},
			{"Search", "Done", "Mobile-friendly search endpoint with result cards."},
			{"Channel navigation", "Stub", "Placeholder page while the full layout is built."},
			{"Playlist browsing", "Stub", "Placeholder view with roadmap for autoplay and queueing."},
			{"Shorts shelf", "Planned", "Dedicated grid for short-form clips."},
			{"Topics and hashtags", "Planned", "Browse by topic entry points."},
		},
	},
	{
		Title: "Playback Experience",
		Features: []feature{
			{"Video playback", "Done", "Retro watch page with MP4 proxy and 3GP transcoding."},
			{"Audio-only mode", "Planned", "Low bandwidth audio streaming profile."},
			{"Captions and subtitles", "Planned", "Toggleable CC tracks."},
			{"Related videos", "Planned", "Up-next queue on the watch page."},
			{"Autoplay toggle", "Planned", "Device-level autoplay preference."},
			{"Playback queue", "Planned", "Mini queue for upcoming videos."},
			{"Live and premieres", "Planned", "Special casing for live HLS manifests."},
		},
	},
	{
		Title: "Social & Interaction",
		Features: []feature{
			{"Comments and replies", "Planned", "Threaded comment feed with filters."},
			{"Likes / dislikes", "Deferred", "Requires authenticated actions."},
			{"Subscriptions feed", "Stub", "Preview page highlighting the future feed."},
			{"Notifications", "Deferred", "Depends on account integration."},
			{"Community posts", "Planned", "Channel text and image updates."},
		},
	},
	{
		Title: "Library & Personalisation",
		Features: []feature{
			{"Watch history", "Planned", "Local-first activity log."},
			{"Watch later", "Planned", "Pocket queue saved across devices."},
			{"Playlist management", "Deferred", "Write operations require authentication."},
			{"Downloads", "Planned", "Offline caching with quota guardrails."},
			{"Subscriptions highlights", "Planned", "Recent uploads from followed channels."},
		},
	},
	{
		Title: "Platform Enhancements",
		Features: []feature{
			{"Search suggestions", "Planned", "Typeahead completions."},
			{"Offline caching", "Planned", "Edge cache for low bandwidth regions."},
			{"API client", "Done", "Centralised YouTube client with TTL cache."},
			{"Observability", "Planned", "Metrics and structured logging."},
			{"Responsive retro UI", "Done", "Lightweight HTML and CSS components."},
			{"Legacy transcoding", "Done", "FFmpeg pipeline serving 3GP/MP4 outputs."},
		},
	},
}

// Handler renders an HTML checklist summarising feature coverage.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var b strings.Builder
		b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>YouTube Mini Feature Map</title><link rel="stylesheet" href="/style.css"></head><body>`)
		b.WriteString(`<div class="head"><a href="/">Back</a> Feature Coverage</div><div class="box">`)
		b.WriteString(`<p>Status legend: <b>Done</b>, <b>Stub</b>, <b>Planned</b>, <b>Deferred</b>.</p>`)

		for _, cat := range catalogue {
			b.WriteString("<h3>" + cat.Title + "</h3><ul>")
			for _, ft := range cat.Features {
				b.WriteString("<li><b>" + ft.Name + "</b> - " + ft.Status + ". " + ft.Description + "</li>")
			}
			b.WriteString("</ul>")
		}

		b.WriteString(`<hr><center><a href="/">Home</a></center></div></body></html>`)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(b.String()))
	}
}
