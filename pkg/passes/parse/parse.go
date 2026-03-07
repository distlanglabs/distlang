package parse

import (
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
)

// Format controls how source is transformed for a backend build.
type Format string

const (
	FormatV8   Format = "v8"
	FormatWasm Format = "wasm"
)

// ToScript transforms module-oriented JS into the requested format.
func ToScript(filename, source string, format Format) (string, error) {
	opts := api.TransformOptions{
		Loader:      api.LoaderJS,
		Sourcefile:  filename,
		Target:      api.ESNext,
		TreeShaking: api.TreeShakingFalse,
	}

	switch format {
	case FormatV8, FormatWasm:
		opts.Format = api.FormatESModule
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}

	result := api.Transform(source, opts)

	if len(result.Errors) > 0 {
		return "", fmt.Errorf("transform: %s", result.Errors[0].Text)
	}

	return string(result.Code), nil
}
