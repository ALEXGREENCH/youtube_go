package playlist

import (
	"net/http"
	"strings"

	"youtube-mini/internal/ui"
)

// Handler renders the playlist placeholder page.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("list"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderPlaylist(id)))
	}
}
