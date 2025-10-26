package static

import "embed"

// FS exposes embedded static assets.
//go:embed *
var FS embed.FS
