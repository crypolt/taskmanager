package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Dist returns the built Vue application, or nil when it has not been built.
func Dist() fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}
