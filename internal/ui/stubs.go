package ui

import "fmt"

// RenderChannel placeholder content for channel view.
func RenderChannel(id string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8">
<title>Channel %s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body><div class="head"><a href="/">Back</a> Channels</div>
<div class="box"><h3>Channel %s</h3>
<p>Full channel profiles are on the roadmap. Expect videos, playlists and community tabs soon.</p>
</div>
<hr><center><a href="/">Home</a></center></body></html>`, Escape(id), Escape(id))
}

// RenderPlaylist placeholder content for playlists.
func RenderPlaylist(id string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8">
<title>Playlist %s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body><div class="head"><a href="/">Back</a> Playlists</div>
<div class="box"><h3>Playlist %s</h3>
<p>The playlist view is under construction. Upcoming: ordered videos, autoplay and Watch Later shortcuts.</p>
</div>
<hr><center><a href="/">Home</a></center></body></html>`, Escape(id), Escape(id))
}

// RenderSubscriptions placeholder content for subscriptions feed.
func RenderSubscriptions() string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8">
<title>Subscriptions - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
</head><body><div class="head"><a href="/">Back</a> Subscriptions</div>
<div class="box"><h3>Subscription feed</h3>
<p>A personalised feed of fresh uploads and live streams from your channels will land here shortly.</p>
</div>
<hr><center><a href="/">Home</a></center></body></html>`
}
