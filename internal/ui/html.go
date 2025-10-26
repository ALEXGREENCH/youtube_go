package ui

import "html"

// Escape ensures user content cannot break inline markup.
func Escape(s string) string {
	return html.EscapeString(s)
}
