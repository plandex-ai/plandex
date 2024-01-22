package format

import (
	"path/filepath"
	"strings"
)

func GetFileNameWithoutExt(path string) string {
	name := path[:len(path)-len(filepath.Ext(path))]

	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "`", "")
	name = strings.ReplaceAll(name, "\"", "")

	return name
}
