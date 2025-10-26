package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"youtube-mini/internal/platform/cache"
	"youtube-mini/internal/youtube"
)

const maxResourceSize = 5 << 20 // 5MB

var allowedHosts = []string{
	"ytimg.com",
	"ggpht.com",
	"googleusercontent.com",
	"googlevideo.com",
	"youtube.com",
}

type cachedResource struct {
	Data        []byte
	ContentType string
	Status      int
}

// Handler returns an HTTP handler that proxies approved media through the backend.
func Handler(client *youtube.Client) http.HandlerFunc {
	httpClient := client.HTTPClient()
	resCache := cache.New[cachedResource]()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		raw := strings.TrimSpace(r.URL.Query().Get("url"))
		if raw == "" {
			http.Error(w, "missing url", http.StatusBadRequest)
			return
		}

		target, err := url.Parse(raw)
		if err != nil || target.Scheme == "" || target.Host == "" {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}

		if !isAllowedHost(target.Host) {
			http.Error(w, "host not permitted", http.StatusForbidden)
			return
		}

		cacheKey := target.String()
		if entry, ok := resCache.Get(cacheKey); ok {
			writeCached(w, entry)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", "YouTubeMiniProxy/1.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "upstream status "+resp.Status, http.StatusBadGateway)
			return
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResourceSize))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		entry := cachedResource{
			Data:        body,
			ContentType: contentType,
			Status:      http.StatusOK,
		}
		resCache.Set(cacheKey, entry, 10*time.Minute)

		writeCached(w, entry)
	}
}

func writeCached(w http.ResponseWriter, entry cachedResource) {
	w.Header().Set("Content-Type", entry.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(entry.Status)
	_, _ = w.Write(entry.Data)
}

func isAllowedHost(host string) bool {
	host = strings.ToLower(host)
	for _, suffix := range allowedHosts {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
