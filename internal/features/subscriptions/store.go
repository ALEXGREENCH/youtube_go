package subscriptions

import (
	"net/http"
	"strings"
	"time"
)

const cookieName = "ytm_subscriptions"

func read(r *http.Request) []string {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return []string{}
	}
	parts := strings.Split(c.Value, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
		if len(out) >= 100 {
			break
		}
	}
	return out
}

func write(w http.ResponseWriter, ids []string) {
	value := strings.Join(ids, ",")
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(90 * 24 * time.Hour),
		HttpOnly: true,
	})
}

// Read returns subscribed channel IDs.
func Read(r *http.Request) []string {
	return read(r)
}

// ReadSet returns a set of subscribed channel IDs for quick lookup.
func ReadSet(r *http.Request) map[string]bool {
	set := make(map[string]bool)
	for _, id := range read(r) {
		set[id] = true
	}
	return set
}

// IsSubscribed reports whether the channel is stored in the cookie.
func IsSubscribed(r *http.Request, channelID string) bool {
	if strings.TrimSpace(channelID) == "" {
		return false
	}
	for _, id := range read(r) {
		if id == channelID {
			return true
		}
	}
	return false
}

// Add inserts a channel into the cookie.
func Add(w http.ResponseWriter, r *http.Request, channelID string) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return
	}
	list := read(r)
	for _, existing := range list {
		if existing == channelID {
			write(w, list)
			return
		}
	}
	list = append([]string{channelID}, list...)
	if len(list) > 100 {
		list = list[:100]
	}
	write(w, list)
}

// Remove deletes a channel from the cookie.
func Remove(w http.ResponseWriter, r *http.Request, channelID string) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return
	}
	list := read(r)
	out := make([]string, 0, len(list))
	for _, existing := range list {
		if existing == channelID {
			continue
		}
		out = append(out, existing)
	}
	write(w, out)
}
