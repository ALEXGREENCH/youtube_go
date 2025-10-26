package stream

import (
	"io"
	"net/http"
	"strings"

	"youtube-mini/internal/youtube"
)

var passthroughHeaders = []string{
	"Content-Type",
	"Content-Length",
	"Content-Range",
	"Accept-Ranges",
	"Cache-Control",
	"ETag",
	"Last-Modified",
	"Expires",
}

// Handler proxies the first available MP4 stream directly, including range support for seeking.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
			return
		}

		idPath := strings.TrimPrefix(r.URL.Path, "/stream/")
		id := strings.TrimSuffix(idPath, ".mp4")
		if id == "" {
			http.Error(w, "missing video id", http.StatusBadRequest)
			return
		}

		video, err := client.GetVideo(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if video.Stream == "" {
			http.Error(w, "stream not available", http.StatusNotFound)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), r.Method, video.Stream, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Referer", "https://www.youtube.com/")

		if rng := r.Header.Get("Range"); rng != "" {
			req.Header.Set("Range", rng)
		}
		if ifRange := r.Header.Get("If-Range"); ifRange != "" {
			req.Header.Set("If-Range", ifRange)
		}

		resp, err := client.HTTPClient().Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			w.WriteHeader(resp.StatusCode)
			_, _ = io.Copy(w, resp.Body)
			return
		}

		// Copy selected headers from upstream response.
		for _, key := range passthroughHeaders {
			if values, ok := resp.Header[key]; ok {
				w.Header()[key] = values
			}
		}
		// Ensure downstream clients know seeking is available.
		if w.Header().Get("Accept-Ranges") == "" {
			w.Header().Set("Accept-Ranges", "bytes")
		}

		w.WriteHeader(resp.StatusCode)

		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.Copy(w, resp.Body)
	}
}
