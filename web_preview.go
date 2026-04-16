package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

func newNextPreviewHandler(previewFS fs.FS) (http.Handler, error) {
	if previewFS == nil {
		resolvedFS, err := resolveNextPreviewFS()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return unavailableNextPreviewHandler(), nil
			}
			return nil, err
		}
		previewFS = resolvedFS
	}

	indexHTML, err := fs.ReadFile(previewFS, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read preview index: %w", err)
	}

	fileServer := http.StripPrefix("/next/", http.FileServer(http.FS(previewFS)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relativePath := strings.TrimPrefix(r.URL.Path, "/next")
		relativePath = strings.TrimPrefix(relativePath, "/")

		if relativePath == "" {
			serveNextPreviewIndex(w, indexHTML)
			return
		}

		if previewFileExists(previewFS, relativePath) {
			fileServer.ServeHTTP(w, r)
			return
		}

		serveNextPreviewIndex(w, indexHTML)
	}), nil
}

func resolveNextPreviewFS() (fs.FS, error) {
	if _, err := os.Stat("web/dist/index.html"); err != nil {
		return nil, err
	}
	return os.DirFS("web/dist"), nil
}

func unavailableNextPreviewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "React preview is unavailable until web/dist is built.", http.StatusServiceUnavailable)
	})
}

func serveNextPreviewIndex(w http.ResponseWriter, indexHTML []byte) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexHTML)
}

func previewFileExists(previewFS fs.FS, name string) bool {
	cleaned := path.Clean(strings.TrimPrefix(name, "/"))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return false
	}

	info, err := fs.Stat(previewFS, cleaned)
	return err == nil && !info.IsDir()
}
