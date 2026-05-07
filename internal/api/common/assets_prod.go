//go:build !dev

package common

import (
	"embed"
	"io/fs"
)

//go:embed static/**
var embeddedFiles embed.FS

func AssetsFS() fs.FS {
	sub, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic(err)
	}

	return sub
}
