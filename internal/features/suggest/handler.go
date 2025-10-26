package suggest

import (
	"encoding/json"
	"net/http"

	"youtube-mini/internal/youtube"
)

type response struct {
	Suggestions []string `json:"suggestions"`
}

// Handler returns JSON suggestions for autocomplete.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		results, err := client.Suggest(r.Context(), query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Suggestions: results})
	}
}
