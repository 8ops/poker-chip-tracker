package main

import (
	"embed"
	"flag"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"poker-chip-tracker/internal/api"
	"poker-chip-tracker/internal/store"
	"poker-chip-tracker/internal/ws"
)

//go:embed all:web
var webFS embed.FS

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "data/poker.db", "sqlite database path")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	st, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer st.Close()

	hub := ws.NewHub()
	srv := api.NewServer(st, hub)

	mux := http.NewServeMux()
	srv.Register(mux)

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed web: %v", err)
	}
	mux.Handle("/", spaHandler(webRoot))

	log.Printf("Poker Chip Tracker running on http://localhost%s", *addr)
	log.Printf("Database: %s", *dbPath)
	if err := http.ListenAndServe(*addr, withCORS(mux)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func spaHandler(webRoot fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(webRoot))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if f, err := webRoot.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback → index.html
		f, err := webRoot.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		stat, _ := f.Stat()
		if rs, ok := f.(io.ReadSeeker); ok && stat != nil {
			http.ServeContent(w, r, "index.html", stat.ModTime(), rs)
			return
		}
		_, _ = io.Copy(w, f)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
