package assets

import (
	"embed"
	"io/fs"
)

//go:embed static/web-display/**
var webDisplayStaticFiles embed.FS

func WebDisplayFS() fs.FS {
	sub, err := fs.Sub(webDisplayStaticFiles, "static/web-display")
	if err != nil {
		panic(err)
	}

	return sub
}
