package ui

import (
	"fmt"
	"strings"

	"youtube-mini/internal/youtube"
)

// WatchPageData aggregates everything required to render the watch view.
type WatchPageData struct {
	Theme             string
	CurrentPath       string
	Video             youtube.Video
	StreamURL         string
	AudioURL          string
	TranscodeLinks    []Link
	Captions          []youtube.CaptionTrack
	AutoplayEnabled   bool
	AutoplayToggleURL string
	AutoplayNextURL   string
	AutoplaySource    string
	Related           []RelatedEntry
	Queue             []QueueEntry
	QueueClearURL     string
	Subscribed        bool
	SubscribeURL      string
	UnsubscribeURL    string
	InWatchLater      bool
	WatchLaterURL     string
}

// Link represents a simple label/url pair.
type Link struct {
	Label string
	URL   string
}

// RelatedEntry wraps a feed item with queue action.
type RelatedEntry struct {
	Item          youtube.FeedItem
	AddToQueueURL string
}

// QueueEntry contains queue metadata and removal action.
type QueueEntry struct {
	Item      youtube.FeedItem
	RemoveURL string
}

// RenderWatch renders the watch page with related videos, queue, and controls.
func RenderWatch(data WatchPageData) string {
	video := data.Video
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8">`)
	fmt.Fprintf(&b, `<title>%s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
`, Escape(video.Title))
	if data.AutoplayEnabled && data.AutoplayNextURL != "" {
		fmt.Fprintf(&b, `<meta http-equiv="refresh" content="5;url=%s">`, Escape(data.AutoplayNextURL))
	}
	b.WriteString(`</head><body class="` + ThemeBodyClass(data.Theme) + `">`)
	b.WriteString(RenderHeader("", data.Theme, data.CurrentPath, ""))
	b.WriteString(`<main class="page watch-page">`)
	b.WriteString(`<div class="watch-hero box">`)
	fmt.Fprintf(&b, `<img src="%s" width="240" alt=""><br>
<b>%s</b><br>`, EscapeAttr(ProxiedImage(video.ThumbURL)), Escape(video.Title))

	if video.Author != "" {
		if video.ChannelID != "" {
			fmt.Fprintf(&b, `<small>By <a href="/channel?id=%s">%s</a></small><br>`, Escape(video.ChannelID), Escape(video.Author))
		} else {
			fmt.Fprintf(&b, `<small>By %s</small><br>`, Escape(video.Author))
		}
	}
	if video.ViewCount != "" || video.LengthSeconds != "" {
		meta := []string{}
		if video.ViewCount != "" {
			meta = append(meta, Escape(video.ViewCount)+" views")
		}
		if video.LengthSeconds != "" {
			meta = append(meta, Escape(video.LengthSeconds)+"s")
		}
		b.WriteString("<small>" + strings.Join(meta, " - ") + "</small><br>")
	}

	if data.Subscribed {
		if data.UnsubscribeURL != "" {
			fmt.Fprintf(&b, `<small><a href="%s">Unsubscribe</a></small><br>`, Escape(data.UnsubscribeURL))
		}
	} else if data.SubscribeURL != "" {
		fmt.Fprintf(&b, `<small><a href="%s">Subscribe</a></small><br>`, Escape(data.SubscribeURL))
	}

	if data.InWatchLater {
		fmt.Fprintf(&b, `<small><a href="%s">Remove from Watch later</a></small><br>`, Escape(data.WatchLaterURL))
	} else if data.WatchLaterURL != "" {
		fmt.Fprintf(&b, `<small><a href="%s">Add to Watch later</a></small><br>`, Escape(data.WatchLaterURL))
	}

	fmt.Fprintf(&b, `<a class="btn" href="%s">Play in browser</a><br>`, Escape(data.StreamURL))
	if data.AudioURL != "" {
		fmt.Fprintf(&b, `<small><a href="%s">Audio only (MP3)</a></small><br>`, Escape(data.AudioURL))
	}
	for _, link := range data.TranscodeLinks {
		fmt.Fprintf(&b, `<small><a href="%s">%s</a></small><br>`, Escape(link.URL), Escape(link.Label))
	}
	b.WriteString(`</div><hr>`)

	if data.AutoplayToggleURL != "" {
		if data.AutoplayEnabled {
			message := "Autoplay is ON"
			if data.AutoplaySource != "" {
				message += " (next: " + Escape(data.AutoplaySource) + ")"
			}
			fmt.Fprintf(&b, `<div class="box"><small>%s - <a href="%s">turn off</a></small></div>`, message, Escape(data.AutoplayToggleURL))
		} else {
			fmt.Fprintf(&b, `<div class="box"><small>Autoplay is OFF - <a href="%s">turn on</a></small></div>`, Escape(data.AutoplayToggleURL))
		}
	}

	b.WriteString(`<div class="box"><b>Available formats</b><br>`)
	if len(video.Formats) == 0 {
		b.WriteString("<small>No format metadata returned.</small>")
	} else {
		b.WriteString("<ul>")
		for _, format := range video.Formats {
			if format.URL == "" {
				continue
			}
			label := format.Quality
			if label == "" {
				label = "Stream"
			}
			if format.Bitrate != "" {
				label += " (" + format.Bitrate + "bps)"
			}
			fmt.Fprintf(&b, `<li>%s - %s</li>`, Escape(label), Escape(format.Mime))
		}
		b.WriteString("</ul>")
	}
	b.WriteString(`</div>`)

	if len(data.Captions) > 0 {
		b.WriteString(`<div class="box"><b>Captions</b><ul>`)
		for _, track := range data.Captions {
			label := track.Language
			if label == "" {
				label = track.Kind
			}
			fmt.Fprintf(&b, `<li><a href="%s">%s</a></li>`, Escape(track.URL), Escape(label))
		}
		b.WriteString(`</ul></div>`)
	}

	if len(data.Queue) > 0 {
		b.WriteString(`<div class="box"><b>Queue</b>`)
		if data.QueueClearURL != "" {
			fmt.Fprintf(&b, ` <small><a href="%s">clear all</a></small>`, Escape(data.QueueClearURL))
		}
		b.WriteString(`<br>`)
		for _, entry := range data.Queue {
			item := entry.Item
			fmt.Fprintf(&b, `<div class="vid">
<table cellspacing="0" cellpadding="2"><tr valign="top">
<td><a href="/watch?v=%s&dequeue=1"><img src="%s" width="96" height="54" alt=""></a></td>
<td><b><a href="/watch?v=%s&dequeue=1">%s</a></b><br>
<small>%s</small><br>
<small><a href="%s">remove</a></small></td>
</tr></table>
</div>`,
				Escape(item.ID), EscapeAttr(ProxiedImage(item.Thumbnail)),
				Escape(item.ID), Escape(item.Title),
				Escape(item.Meta), Escape(entry.RemoveURL))
		}
		b.WriteString(`</div>`)
	}

	if len(data.Related) > 0 {
		b.WriteString(`<div class="box"><b>Related</b></div>`)
		for _, entry := range data.Related {
			item := entry.Item
			channelID := item.ChannelID
			if channelID == "" {
				channelID = item.Channel
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
	<small><a href="%s">Add to queue</a></small>
</td>
</tr></table>
</div>`,
				Escape(item.ID), EscapeAttr(ProxiedImage(item.Thumbnail)), Escape(item.Duration),
				Escape(item.ID), Escape(item.Title),
				Escape(channelID), Escape(item.Channel),
				Escape(item.Meta), Escape(entry.AddToQueueURL))
		}
	}

	b.WriteString(`<hr><div class="footer-link"><a href="/">Home</a></div>`)
	b.WriteString(`</main>`)
	AppendSuggestionScript(&b)
	b.WriteString(`</body></html>`)
	return b.String()
}
