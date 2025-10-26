package audio

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"youtube-mini/internal/youtube"
)

// Handler streams audio-only formats with range support.
func Handler(client *youtube.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/stream/audio/")
		id = strings.TrimSuffix(id, ".mp3")
		if id == "" {
			http.Error(w, "missing video id", http.StatusBadRequest)
			return
		}

		video, err := client.GetVideo(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		format := selectBestAudio(video.Audio)
		if format.URL == "" {
			http.Error(w, "audio stream unavailable", http.StatusNotFound)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), r.Method, format.URL, nil)
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

		contentType := format.Mime
		if idx := strings.Index(contentType, ";"); idx > 0 {
			contentType = contentType[:idx]
		}
		if contentType == "" {
			contentType = "audio/mpeg"
		}

		for _, key := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Cache-Control", "ETag", "Last-Modified", "Expires"} {
			if values, ok := resp.Header[key]; ok {
				w.Header()[key] = values
			}
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", contentType)
		}
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

func selectBestAudio(formats []youtube.Format) youtube.Format {
	if len(formats) == 0 {
		return youtube.Format{}
	}
	sort.SliceStable(formats, func(i, j int) bool {
		ib, _ := strconv.Atoi(formats[i].Bitrate)
		jb, _ := strconv.Atoi(formats[j].Bitrate)
		return ib > jb
	})
	return formats[0]
}
