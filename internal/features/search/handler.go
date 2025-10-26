package search

import (
	"net/http"
	"net/url"
	"strings"

	"youtube-mini/internal/youtube"
)

// Handler redirects to unified home search.
func Handler(_ *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if strings.TrimSpace(query) == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/?q="+url.QueryEscape(query), http.StatusFound)
	}
}
