package main

import (
	"embed"
	"net/http"
)

// static/ holds the favicon, home-screen icons and web app manifest. They are
// embedded like the cards and templates, so the server needs no filesystem.
//
//go:embed static/*
var staticFS embed.FS

// staticFiles are served from the site root, where browsers and phones look
// for them by default (/favicon.ico, /apple-touch-icon.png).
var staticFiles = []string{
	"favicon.ico",
	"favicon-32x32.png",
	"apple-touch-icon.png",
	"android-chrome-192x192.png",
	"android-chrome-512x512.png",
	"site.webmanifest",
}

func registerStatic(mux *http.ServeMux) {
	for _, name := range staticFiles {
		mux.HandleFunc("GET /"+name, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=86400")
			if name == "site.webmanifest" {
				w.Header().Set("Content-Type", "application/manifest+json")
			}
			http.ServeFileFS(w, r, staticFS, "static/"+name)
		})
	}
}
