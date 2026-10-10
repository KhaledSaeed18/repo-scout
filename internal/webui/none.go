//go:build !embedui

package webui

import "io/fs"

// Assets returns nil: this build carries no frontend.
func Assets() (fs.FS, error) { return nil, nil }
