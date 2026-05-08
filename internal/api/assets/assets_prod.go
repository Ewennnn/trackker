//go:build !dev

package assets

import (
	"embed"
	"io/fs"
)

//go:embed static/web-controls/**
var webControlsStaticFiles embed.FS

func WebControlsFS() fs.FS {
	sub, err := fs.Sub(webControlsStaticFiles, "static/web-controls")
	if err != nil {
		panic(err)
	}

	return sub
}
