package watchlater

import (
	"net/http"
	"strings"
	"time"
)

const cookieName = "ytm_watchlater"

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
		if len(out) >= 50 {
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
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
	})
}

// Read returns the list of watch later IDs in insertion order.
func Read(r *http.Request) []string {
	return read(r)
}

// Set writes the provided identifiers to the cookie.
func Set(w http.ResponseWriter, ids []string) {
	write(w, ids)
}

// Contains reports whether id exists in the current request context.
func Contains(r *http.Request, id string) bool {
	for _, existing := range read(r) {
		if existing == id {
			return true
		}
	}
	return false
}

func appendUnique(ids []string, id string) []string {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	ids = append([]string{id}, ids...)
	if len(ids) > 50 {
		ids = ids[:50]
	}
	return ids
}

func removeID(ids []string, id string) []string {
	out := make([]string, 0, len(ids))
	for _, existing := range ids {
		if existing == id {
			continue
		}
		out = append(out, existing)
	}
	return out
}

func toSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// ReadSet returns the watch later collection as a map for fast lookups.
func ReadSet(r *http.Request) map[string]bool {
	return toSet(read(r))
}

// Add stores the given id in the cookie.
func Add(w http.ResponseWriter, r *http.Request, id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	list := appendUnique(read(r), id)
	write(w, list)
}

// Remove deletes id from the cookie.
func Remove(w http.ResponseWriter, r *http.Request, id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	list := removeID(read(r), id)
	write(w, list)
}

// Clear wipes the watch later cookie.
func Clear(w http.ResponseWriter) {
	write(w, nil)
}
