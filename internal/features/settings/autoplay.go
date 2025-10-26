package settings

import (
	"net/http"
	"strings"
	"time"
)

const autoplayCookie = "ytm_autoplay"

// AutoplayHandler toggles autoplay preference.
func AutoplayHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("state")))
		if state != "on" && state != "off" {
			if cookie, err := r.Cookie(autoplayCookie); err == nil && cookie.Value == "on" {
				state = "off"
			} else {
				state = "on"
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:     autoplayCookie,
			Value:    state,
			Path:     "/",
			Expires:  time.Now().Add(30 * 24 * time.Hour),
			HttpOnly: true,
		})

		target := r.URL.Query().Get("return")
		if strings.TrimSpace(target) == "" {
			target = "/watch"
		}
		http.Redirect(w, r, target, http.StatusFound)
	}
}

// ReadAutoplay returns true if autoplay is enabled.
func ReadAutoplay(r *http.Request) bool {
	if cookie, err := r.Cookie(autoplayCookie); err == nil && cookie.Value == "on" {
		return true
	}
	return false
}
