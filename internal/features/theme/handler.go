package theme

import (
	"net/http"
	"strings"
	"time"
)

const cookieName = "ytm_theme"

// FromRequest extracts the theme from request cookies.
func FromRequest(r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil {
		value := strings.ToLower(strings.TrimSpace(c.Value))
		if value == "dark" {
			return "dark"
		}
	}
	return "light"
}

// Handler sets the theme cookie and redirects back.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("name")))
		switch name {
		case "dark", "light":
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    name,
				Path:     "/",
				Expires:  time.Now().Add(365 * 24 * time.Hour),
				HttpOnly: true,
			})
		}
		returnURL := r.URL.Query().Get("return")
		if strings.TrimSpace(returnURL) == "" {
			returnURL = "/"
		}
		http.Redirect(w, r, returnURL, http.StatusFound)
	}
}
