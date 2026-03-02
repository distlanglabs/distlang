package parse

import (
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
)

// ToScript transforms module-oriented JS into Goja-friendly script code.
func ToScript(filename, source string) (string, error) {
	result := api.Transform(source, api.TransformOptions{
		Loader:      api.LoaderJS,
		Sourcefile:  filename,
		Format:      api.FormatIIFE,
		GlobalName:  "distlangWorker",
		Target:      api.ESNext,
		TreeShaking: api.TreeShakingFalse,
	})

	if len(result.Errors) > 0 {
		return "", fmt.Errorf("transform: %s", result.Errors[0].Text)
	}

	return string(result.Code), nil
}
