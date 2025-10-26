package ui

import (
	"net/url"
	"strings"
)

// ProxiedImage returns a proxied URL for remote images to avoid direct CDN access.
func ProxiedImage(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/proxy?") || strings.HasPrefix(raw, "/static/") || strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return "/proxy?url=" + url.QueryEscape(raw)
	}
	return raw
}
