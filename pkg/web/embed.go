package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFS embed.FS

func FileSystem() http.FileSystem {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}
