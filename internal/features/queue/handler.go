package queue

import (
	"net/http"
	"strings"
	"time"
)

const queueCookie = "ytm_queue"

// AddHandler appends a video id to the lightweight queue.
func AddHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		queue := readQueue(r)
		queue = appendUnique(queue, id)
		writeQueue(w, queue)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

// RemoveHandler deletes a single id from the queue.
func RemoveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		Remove(w, r, id)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

// ClearHandler wipes the queue.
func ClearHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeQueue(w, nil)
		http.Redirect(w, r, redirectTarget(r), http.StatusFound)
	}
}

func readQueue(r *http.Request) []string {
	cookie, err := r.Cookie(queueCookie)
	if err != nil || cookie.Value == "" {
		return []string{}
	}
	parts := strings.Split(cookie.Value, ",")
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
		if len(out) >= 25 {
			break
		}
	}
	return out
}

func writeQueue(w http.ResponseWriter, queue []string) {
	value := strings.Join(queue, ",")
	http.SetCookie(w, &http.Cookie{
		Name:     queueCookie,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
	})
}

func appendUnique(queue []string, id string) []string {
	for _, existing := range queue {
		if existing == id {
			return queue
		}
	}
	queue = append(queue, id)
	if len(queue) > 25 {
		queue = queue[len(queue)-25:]
	}
	return queue
}

func redirectTarget(r *http.Request) string {
	ret := r.URL.Query().Get("return")
	if strings.TrimSpace(ret) == "" {
		return "/"
	}
	return ret
}

// Read exposes the queue for other handlers.
func Read(r *http.Request) []string {
	return readQueue(r)
}

// Remove deletes the first occurrence of id from the queue.
func Remove(w http.ResponseWriter, r *http.Request, id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	queue := readQueue(r)
	updated := make([]string, 0, len(queue))
	removed := false
	for _, item := range queue {
		if !removed && item == id {
			removed = true
			continue
		}
		updated = append(updated, item)
	}
	if removed {
		writeQueue(w, updated)
	}
}

// Set replaces the queue cookie with a new list.
func Set(w http.ResponseWriter, queue []string) {
	writeQueue(w, queue)
}
