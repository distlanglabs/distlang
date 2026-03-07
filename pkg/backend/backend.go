package backend

import (
	"fmt"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	v8backend "github.com/distlanglabs/distlang/pkg/backend/v8"
	wasmbackend "github.com/distlanglabs/distlang/pkg/backend/wasm"
)

// Name identifies an execution backend.
type Name string

const (
	V8   Name = "v8"
	Wasm Name = "wasm"
)

// Result captures artifacts for a backend build.
type Result struct {
	Backend   Name
	Artifacts []artifacts.Artifact
}

// BuildAll builds all supported backends, returning per-backend results and errors.
func BuildAll(filePath string) ([]Result, map[Name]error) {
	backends := []Name{V8, Wasm}
	var results []Result
	errs := make(map[Name]error)

	for _, b := range backends {
		res, err := Build(filePath, b)
		if err != nil {
			errs[b] = err
			continue
		}
		results = append(results, res)
	}

	return results, errs
}

// Build builds artifacts for a single backend.
func Build(filePath string, name Name) (Result, error) {
	switch name {
	case V8:
		out, err := v8backend.Build(filePath)
		if err != nil {
			return Result{}, err
		}
		return Result{Backend: V8, Artifacts: out.Artifacts}, nil
	case Wasm:
		out, err := wasmbackend.Build(filePath)
		if err != nil {
			return Result{}, err
		}
		return Result{Backend: Wasm, Artifacts: out.Artifacts}, nil
	default:
		return Result{}, fmt.Errorf("unsupported backend: %s", name)
	}
}

// BuildV8 builds the V8 backend output.
func BuildV8(filePath string) (v8backend.Output, error) {
	return v8backend.Build(filePath)
}

// BuildWasm builds the Wasm backend output.
func BuildWasm(filePath string) (wasmbackend.Output, error) {
	return wasmbackend.Build(filePath)
}
