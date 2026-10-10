//go:build embedui

package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Assets returns the embedded frontend build.
func Assets() (fs.FS, error) { return fs.Sub(dist, "dist") }
