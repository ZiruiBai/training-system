// Package web embeds the built frontend SPA and serves it alongside the API.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns a filesystem rooted at the built frontend `dist` directory.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
