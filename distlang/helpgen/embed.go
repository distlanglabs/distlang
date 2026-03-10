package helpgen

import _ "embed"

//go:embed core/objectdb.js
var coreObjectDB string

func CoreObjectDB() string {
	return coreObjectDB
}
