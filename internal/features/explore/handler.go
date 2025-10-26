package explore

import (
	"net/http"
	"net/url"

	"youtube-mini/internal/youtube"
)

// Handler redirects to the unified home endpoint with the explore tab.
func Handler(_ *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := "/?tab=explore"
		if q := r.URL.Query().Get("q"); q != "" {
			target += "&q=" + url.QueryEscape(q)
		}
		http.Redirect(w, r, target, http.StatusFound)
	}
}
