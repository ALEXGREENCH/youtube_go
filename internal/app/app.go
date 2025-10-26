package app

import (
	"net/http"
	"os"
	"path/filepath"

	"youtube-mini/internal/features/audio"
	"youtube-mini/internal/features/channel"
	"youtube-mini/internal/features/explore"
	"youtube-mini/internal/features/featuremap"
	"youtube-mini/internal/features/history"
	"youtube-mini/internal/features/index"
	"youtube-mini/internal/features/playlist"
	"youtube-mini/internal/features/proxy"
	"youtube-mini/internal/features/queue"
	"youtube-mini/internal/features/search"
	"youtube-mini/internal/features/settings"
	"youtube-mini/internal/features/stream"
	"youtube-mini/internal/features/style"
	"youtube-mini/internal/features/subscriptions"
	"youtube-mini/internal/features/suggest"
	"youtube-mini/internal/features/theme"
	"youtube-mini/internal/features/transcoder"
	"youtube-mini/internal/features/watch"
	"youtube-mini/internal/features/watchlater"
	"youtube-mini/internal/platform/metrics"
	"youtube-mini/internal/transcode"
	"youtube-mini/internal/youtube"
	staticfs "youtube-mini/static"
)

// App wires dependencies and exposes the HTTP handler tree.
type App struct {
	mux     *http.ServeMux
	metrics *metrics.Registry
}

// New constructs a fully wired application.
func New(youtubeClient *youtube.Client, legacy *transcode.Service) *App {
	registry := metrics.New()
	mux := http.NewServeMux()

	mux.Handle("/metrics", registry.Handler())
	mux.Handle("/style.css", registry.Wrap("style", style.Handler()))

	// Serve static assets (logo, icons). Prefer filesystem if available, otherwise embedded FS.
	var staticHandler http.Handler
	if dir := staticPath(); dir != "" {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			staticHandler = http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
		}
	}
	if staticHandler == nil {
		staticHandler = http.StripPrefix("/static/", http.FileServer(http.FS(staticfs.FS)))
	}
	mux.Handle("/static/", registry.Wrap("static", staticHandler))

	mux.Handle("/features", registry.Wrap("features", featuremap.Handler()))
	mux.Handle("/suggest", registry.Wrap("suggest", suggest.Handler(youtubeClient)))
	mux.Handle("/theme", registry.Wrap("theme", theme.Handler()))
	mux.Handle("/proxy", registry.Wrap("proxy", proxy.Handler(youtubeClient)))

	mux.Handle("/", registry.Wrap("home", index.Handler(youtubeClient)))
	mux.Handle("/explore", registry.Wrap("explore", explore.Handler(youtubeClient)))
	mux.Handle("/search", registry.Wrap("search", search.Handler(youtubeClient)))
	mux.Handle("/watch", registry.Wrap("watch", watch.Handler(youtubeClient)))
	mux.Handle("/channel", registry.Wrap("channel", channel.Handler(youtubeClient)))
	mux.Handle("/playlist", registry.Wrap("playlist", playlist.Handler()))
	mux.Handle("/subscriptions", registry.Wrap("subscriptions", subscriptions.Handler(youtubeClient, watchlater.ReadSet)))

	mux.Handle("/stream/ffmpeg/", registry.Wrap("stream_ffmpeg", transcoder.Handler(youtubeClient, legacy)))
	mux.Handle("/stream/audio/", registry.Wrap("stream_audio", audio.Handler(youtubeClient)))
	mux.Handle("/stream/", registry.Wrap("stream_direct", stream.Handler(youtubeClient)))

	mux.Handle("/queue/add", registry.Wrap("queue_add", queue.AddHandler()))
	mux.Handle("/queue/remove", registry.Wrap("queue_remove", queue.RemoveHandler()))
	mux.Handle("/queue/clear", registry.Wrap("queue_clear", queue.ClearHandler()))

	mux.Handle("/watchlater", registry.Wrap("watchlater", watchlater.Handler(youtubeClient)))
	mux.Handle("/watchlater/add", registry.Wrap("watchlater_add", watchlater.AddHandler()))
	mux.Handle("/watchlater/remove", registry.Wrap("watchlater_remove", watchlater.RemoveHandler()))
	mux.Handle("/watchlater/clear", registry.Wrap("watchlater_clear", watchlater.ClearHandler()))

	mux.Handle("/history", registry.Wrap("history", history.Handler(youtubeClient)))
	mux.Handle("/history/clear", registry.Wrap("history_clear", history.ClearHandler()))

	mux.Handle("/subscriptions/add", registry.Wrap("subscriptions_add", subscriptions.AddHandler()))
	mux.Handle("/subscriptions/remove", registry.Wrap("subscriptions_remove", subscriptions.RemoveHandler()))

	mux.Handle("/settings/autoplay", registry.Wrap("settings_autoplay", settings.AutoplayHandler()))

	return &App{mux: mux, metrics: registry}
}

func staticPath() string {
	if root := os.Getenv("YTM_STATIC_DIR"); root != "" {
		return root
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(base, "static"),
			filepath.Join(base, "..", "static"),
			filepath.Join(base, "..", "..", "static"),
			filepath.Join(".", "static"),
		}
		for _, p := range candidates {
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				return p
			}
		}
	}
	return filepath.Join(".", "static")
}

// Handler returns the root http.Handler.
func (a *App) Handler() http.Handler {
	return a.mux
}
