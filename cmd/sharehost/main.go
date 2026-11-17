package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/sharehost"
)

func main() {
	var (
		store sharehost.Store
		err   error
	)
	if endpoint := os.Getenv("SHARE_S3_ENDPOINT"); endpoint == "" {
		store = sharehost.NewMemStore()
	} else {
		store, err = sharehost.NewS3Store(endpoint, os.Getenv("SHARE_S3_BUCKET"), os.Getenv("SHARE_S3_KEY"), os.Getenv("SHARE_S3_SECRET"))
		if err != nil {
			log.Fatal(err)
		}
	}
	go sharehost.RunSweep(context.Background(), store, 24*time.Hour, time.Now)

	h := newMux(os.Getenv("SITE_DIR"), os.Getenv("PACKAGES_DIR"), sharehost.New(store))
	addr := ":" + getenv("PORT", "8080")
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newMux(siteDir, packagesDir string, api http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/shares", api)
	mux.Handle("/api/shares/", api)
	if packagesDir != "" {
		mux.Handle("/packages/", http.StripPrefix("/packages/", http.FileServer(http.Dir(packagesDir))))
	}
	if siteDir != "" {
		mux.Handle("/", siteWith404(siteDir))
	}
	return mux
}

func siteWith404(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	root := os.DirFS(dir)
	notFound := filepath.Join(dir, "404.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if rel == "." {
			rel = ""
		}
		if rel == "" || fileExists(root, rel) || fileExists(root, path.Join(rel, "index.html")) {
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		b, err := os.ReadFile(notFound) //nolint:gosec // SITE_DIR is the operator-configured document root
		if err == nil {
			_, _ = w.Write(b)
		}
	})
}

func fileExists(root fs.FS, name string) bool {
	if name == "" {
		return false
	}
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	return err == nil && !st.IsDir()
}
