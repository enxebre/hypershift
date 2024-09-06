package assets

import (
	"embed"
)

//go:embed **/*.yaml
var Root embed.FS

// ReadFile reads and returns the content of the named file.
func ReadFile(name string) ([]byte, error) {
	return Root.ReadFile(name)
}
