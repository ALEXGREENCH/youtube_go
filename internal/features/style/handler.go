package style

import (
	"net/http"

	"youtube-mini/internal/ui"
)

// Handler returns the base CSS stylesheet.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderStyle()))
	}
}
