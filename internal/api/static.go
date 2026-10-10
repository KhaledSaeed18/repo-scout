package api

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// contentSecurityPolicy limits the interface to its own origin. Styles allow
// inline attributes because React and React Flow position elements with them.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; " +
	"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

// securityHeaders sets headers every response should carry.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// spa serves the embedded frontend. Paths that are not files fall back to
// index.html so client-side routes load on refresh. Hashed build assets are
// cached for good; everything else is revalidated so upgrades show at once.
func spa(assets fs.FS) http.HandlerFunc {
	files := http.FileServerFS(assets)
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		info, err := fs.Stat(assets, name)
		if name == "" || errors.Is(err, fs.ErrNotExist) || (err == nil && info.IsDir()) {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, assets, "index.html")
			return
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	}
}
