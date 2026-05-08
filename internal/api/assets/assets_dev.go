//go:build dev

package assets

import (
	"io/fs"
	"os"
)

func WebControlsFS() fs.FS {
	return os.DirFS("web-controls/dist")
}
