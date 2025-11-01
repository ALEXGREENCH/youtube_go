package transcoder

import (
	"net/http"
	"strings"

	"youtube-mini/internal/transcode"
	"youtube-mini/internal/youtube"
)

// Handler wraps the transcode service into an HTTP endpoint.
func Handler(client *youtube.Client, svc *transcode.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/stream/ffmpeg/")
		id := strings.TrimSuffix(path, ".mp4")
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

		profile := profileFromQuery(r)
		start := startFromQuery(r)
		if err := svc.Stream(r.Context(), w, video.Stream, id, profile, start); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func profileFromQuery(r *http.Request) transcode.Profile {
	q := r.URL.Query()
	switch {
	case len(q["aac"]) > 0:
		return transcode.ProfileAAC
	case len(q["mp3"]) > 0:
		return transcode.ProfileMP3
	case len(q["edge"]) > 0:
		return transcode.ProfileEdge
	default:
		return transcode.ProfileRetro
	}
}

func startFromQuery(r *http.Request) float64 {
	q := r.URL.Query()
	raw := strings.TrimSpace(q.Get("start"))
	if raw == "" {
		raw = strings.TrimSpace(q.Get("t"))
	}
	if raw == "" {
		return 0
	}
	if secs, ok := transcode.ParseTimeSpec(raw); ok {
		return secs
	}
	return 0
}
