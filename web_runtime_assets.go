package main

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var webDistFiles embed.FS

func embeddedWebRuntimeFS() fs.FS {
	runtimeFS, err := fs.Sub(webDistFiles, "web/dist")
	if err != nil {
		return nil
	}
	return runtimeFS
}
