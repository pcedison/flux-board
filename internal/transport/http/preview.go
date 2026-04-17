package transporthttp

import (
	"errors"
	"fmt"
	"io/fs"
	stdhttp "net/http"
	"os"
	"path"
	"strings"
)

type WebRuntimeOptions struct {
	MissingMessage string
	MountPath      string
}

func NewRootWebRuntimeHandler(runtimeFS fs.FS) (stdhttp.Handler, error) {
	return NewWebRuntimeHandler(runtimeFS, WebRuntimeOptions{
		MissingMessage: "React runtime is unavailable until web/dist is built.",
		MountPath:      "/",
	})
}

func NewNextPreviewHandler(previewFS fs.FS) (stdhttp.Handler, error) {
	return NewWebRuntimeHandler(previewFS, WebRuntimeOptions{
		MissingMessage: "React preview is unavailable until web/dist is built.",
		MountPath:      "/next/",
	})
}

func NewWebRuntimeHandler(runtimeFS fs.FS, options WebRuntimeOptions) (stdhttp.Handler, error) {
	if runtimeFS == nil {
		resolvedFS, err := resolveWebRuntimeFS()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return unavailableWebRuntimeHandler(options.MissingMessage), nil
			}
			return nil, err
		}
		runtimeFS = resolvedFS
	}

	mountPath := normalizeMountPath(options.MountPath)

	indexHTML, err := fs.ReadFile(runtimeFS, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read runtime index: %w", err)
	}

	fileServer := newRuntimeFileServer(runtimeFS, mountPath)
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		relativePath := runtimeRelativePath(r.URL.Path, mountPath)

		if relativePath == "" {
			serveNextPreviewIndex(w, indexHTML)
			return
		}

		if previewFileExists(runtimeFS, relativePath) {
			fileServer.ServeHTTP(w, r)
			return
		}

		serveNextPreviewIndex(w, indexHTML)
	}), nil
}

func resolveWebRuntimeFS() (fs.FS, error) {
	if _, err := os.Stat("web/dist/index.html"); err != nil {
		return nil, err
	}
	return os.DirFS("web/dist"), nil
}

func unavailableWebRuntimeHandler(message string) stdhttp.Handler {
	if strings.TrimSpace(message) == "" {
		message = "React runtime is unavailable until web/dist is built."
	}
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Cache-Control", "no-store")
		stdhttp.Error(w, message, stdhttp.StatusServiceUnavailable)
	})
}

func serveNextPreviewIndex(w stdhttp.ResponseWriter, indexHTML []byte) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(stdhttp.StatusOK)
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

func newRuntimeFileServer(runtimeFS fs.FS, mountPath string) stdhttp.Handler {
	if mountPath == "/" {
		return stdhttp.StripPrefix("/", stdhttp.FileServer(stdhttp.FS(runtimeFS)))
	}
	return stdhttp.StripPrefix(mountPath, stdhttp.FileServer(stdhttp.FS(runtimeFS)))
}

func runtimeRelativePath(requestPath string, mountPath string) string {
	if mountPath == "/" {
		return strings.TrimPrefix(requestPath, "/")
	}

	relativePath := strings.TrimPrefix(requestPath, strings.TrimSuffix(mountPath, "/"))
	return strings.TrimPrefix(relativePath, "/")
}

func normalizeMountPath(mountPath string) string {
	if mountPath == "" || mountPath == "/" {
		return "/"
	}
	return strings.TrimRight(mountPath, "/") + "/"
}
