package helpgen

import (
	"embed"
	"io/fs"
	"strings"
)

import _ "embed"

//go:embed core/inmemdb.js
var coreInMemDB string

func CoreInMemDB() string {
	return coreInMemDB
}

//go:embed distlang/*.js
var distlangHelpersFS embed.FS

func DistlangHelpers() string {
	contents, err := DistlangHelperModule("index.js")
	if err != nil {
		panic(err)
	}
	return contents
}

func DistlangHelperModule(name string) (string, error) {
	cleanName := strings.TrimPrefix(name, "./")
	bytes, err := distlangHelpersFS.ReadFile("distlang/" + cleanName)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func DistlangHelperModules() ([]string, error) {
	entries, err := fs.ReadDir(distlangHelpersFS, "distlang")
	if err != nil {
		return nil, err
	}
	modules := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		modules = append(modules, entry.Name())
	}
	return modules, nil
}

//go:embed layers/simpleapp.js
var layersSimpleApp string

func LayersSimpleApp() string {
	return layersSimpleApp
}
