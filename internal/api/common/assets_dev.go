//go:build dev

package common

import (
	"io/fs"
	"os"
)

func AssetsFS() fs.FS {
	return os.DirFS("web-controls/dist")
}
