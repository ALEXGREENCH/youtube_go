package index

import (
	"net/http"

	"youtube-mini/internal/features/home"
	"youtube-mini/internal/youtube"
)

// Handler delegates to the home feed for the landing page.
func Handler(client *youtube.Client) http.HandlerFunc {
	return home.Handler(client)
}
