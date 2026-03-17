package helpgen

import _ "embed"

//go:embed core/inmemdb.js
var coreInMemDB string

func CoreInMemDB() string {
	return coreInMemDB
}

//go:embed distlang/index.js
var distlangHelpers string

func DistlangHelpers() string {
	return distlangHelpers
}

//go:embed layers/simpleapp.js
var layersSimpleApp string

func LayersSimpleApp() string {
	return layersSimpleApp
}
