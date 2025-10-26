package ui

import "strings"

// ThemeBodyClass converts theme id to body class.
func ThemeBodyClass(theme string) string {
	switch strings.ToLower(theme) {
	case "dark":
		return "theme-dark"
	default:
		return "theme-light"
	}
}

// NextTheme returns the opposite theme name.
func NextTheme(theme string) string {
	if strings.ToLower(theme) == "dark" {
		return "light"
	}
	return "dark"
}
