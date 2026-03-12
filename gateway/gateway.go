// Package gateway serves the Admin and Portal frontend SPAs as embedded static files.
package gateway

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed admin/*
var adminFS embed.FS

//go:embed portal/*
var portalFS embed.FS

var adminHandler http.Handler
var portalHandler http.Handler

func init() {
	adminSub, _ := fs.Sub(adminFS, "admin")
	portalSub, _ := fs.Sub(portalFS, "portal")
	adminHandler = spaHandler(http.FileServerFS(adminSub), adminSub)
	portalHandler = spaHandler(http.FileServerFS(portalSub), portalSub)
}

// spaHandler wraps a file server to fall back to index.html for SPA routing.
func spaHandler(fileServer http.Handler, fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file; if it doesn't exist, serve index.html
		if _, err := fs.Stat(fsys, path); err != nil {
			r.URL.Path = "/"
		}

		// Set cache headers for hashed assets
		if strings.HasPrefix(path, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		fileServer.ServeHTTP(w, r)
	})
}

// Version is set at build time via ldflags, or read from .version file.
var Version = "dev"

func init() {
	if Version == "dev" {
		if data, err := os.ReadFile(".version"); err == nil {
			if v := strings.TrimSpace(string(data)); v != "" {
				Version = v
			}
		}
	}
}

// VersionResponse returns the running version of BOS3000.
type VersionResponse struct {
	Version string `json:"version"`
}

//encore:api public method=GET path=/version
func GetVersion(ctx context.Context) (*VersionResponse, error) {
	return &VersionResponse{Version: Version}, nil
}

//encore:api public raw path=/admin/!rest
func Admin(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/admin")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	adminHandler.ServeHTTP(w, r)
}

//encore:api public raw path=/portal/!rest
func Portal(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/portal")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	portalHandler.ServeHTTP(w, r)
}
