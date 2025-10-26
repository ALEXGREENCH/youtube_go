package history

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const cookieName = "ytm_history"

// Entry describes a single watch history record.
type Entry struct {
	VideoID string
	SeenAt  time.Time
}

func read(r *http.Request) []Entry {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return []Entry{}
	}
	parts := strings.Split(c.Value, ",")
	out := make([]Entry, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segments := strings.Split(part, "|")
		id := strings.TrimSpace(segments[0])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ts := time.Now()
		if len(segments) > 1 {
			if unix, err := strconv.ParseInt(segments[1], 10, 64); err == nil {
				ts = time.Unix(unix, 0)
			}
		}
		out = append(out, Entry{VideoID: id, SeenAt: ts})
		if len(out) >= 50 {
			break
		}
	}
	return out
}

// Read returns watch history entries ordered from newest to oldest.
func Read(r *http.Request) []Entry {
	return read(r)
}

func write(w http.ResponseWriter, entries []Entry) {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.VideoID+"|"+strconv.FormatInt(entry.SeenAt.Unix(), 10))
	}
	value := strings.Join(parts, ",")
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
	})
}

// Record prepends an item to history, ensuring uniqueness.
func Record(w http.ResponseWriter, r *http.Request, id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	now := time.Now()
	entries := read(r)
	pruned := make([]Entry, 0, len(entries)+1)
	pruned = append(pruned, Entry{VideoID: id, SeenAt: now})
	for _, entry := range entries {
		if entry.VideoID == id {
			continue
		}
		pruned = append(pruned, entry)
		if len(pruned) >= 50 {
			break
		}
	}
	write(w, pruned)
}

// Clear removes history data.
func Clear(w http.ResponseWriter) {
	write(w, nil)
}
