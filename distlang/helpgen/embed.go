package helpgen

import _ "embed"

//go:embed core/inmemdb.js
var coreInMemDB string

func CoreInMemDB() string {
	return coreInMemDB
}
