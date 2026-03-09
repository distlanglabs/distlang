package backend

import v8backend "github.com/distlanglabs/distlang/pkg/backend/v8"

// Name identifies an execution backend.
type Name string

const (
	V8 Name = "v8"
)

// BuildV8 builds the V8 backend output.
func BuildV8(filePath string) (v8backend.Output, error) {
	return v8backend.Build(filePath)
}
