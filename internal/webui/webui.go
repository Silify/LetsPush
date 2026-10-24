// Package webui serves the admin interface that is built into the binary.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var assets embed.FS

// Handler serves the built interface. A path with no file behind it gets the page itself,
// so the interface's own routes work when reloaded.
func Handler() http.Handler {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return handler(dist)
}

func handler(files fs.FS) http.Handler {
	server := http.FileServerFS(files)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if _, err := fs.Stat(files, "index.html"); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("LetsPush is running, but its web interface has not been built into this binary.\n"))
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if info, err := fs.Stat(files, name); err != nil || info.IsDir() {
			// An asset that does not exist is a real 404; anything else is a page of the interface.
			if path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/"
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		server.ServeHTTP(w, r)
	})
}
