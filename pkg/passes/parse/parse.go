package parse

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/distlanglabs/distlang/distlang/helpgen"
	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/evanw/esbuild/pkg/api"
)

// Format controls how source is transformed for a backend build.
type Format string

const (
	FormatV8 Format = "v8"
)

// Options controls parse/build-time JS generation.
type Options struct {
	Format Format
}

// Result captures transformed code and any generated helper artifacts.
type Result struct {
	Code      string
	Artifacts []artifacts.Artifact
}

// ToScript transforms module-oriented JS into the requested format.
func ToScript(filename, source string, format Format) (string, error) {
	res, err := ToScriptWithOptions(filename, source, Options{Format: format})
	if err != nil {
		return "", err
	}
	return res.Code, nil
}

// ToScriptWithOptions transforms source and may emit generated helper modules.
func ToScriptWithOptions(filename, source string, opts Options) (Result, error) {
	buildOpts := api.BuildOptions{
		Bundle:      true,
		Write:       false,
		Format:      api.FormatESModule,
		Target:      api.ESNext,
		TreeShaking: api.TreeShakingFalse,
	}

	switch opts.Format {
	case FormatV8:
	default:
		return Result{}, fmt.Errorf("unknown format: %s", opts.Format)
	}

	var generated []artifacts.Artifact
	if strings.Contains(source, "distlang/core") {
		helperSource := helpgen.CoreObjectDB()
		generated = append(generated, artifacts.Artifact{
			Path:    filepath.Join("generated", "distlang", "core", "index.js"),
			Content: []byte(helperSource),
		})

		buildOpts.Plugins = []api.Plugin{{
			Name: "distlang-core",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `^distlang/core$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: "distlang/core", Namespace: "distlang-core"}, nil
				})
				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "distlang-core"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					loader := api.LoaderJS
					return api.OnLoadResult{Contents: &helperSource, Loader: loader}, nil
				})
			},
		}}
	}

	absFile, err := filepath.Abs(filename)
	if err != nil {
		return Result{}, err
	}

	buildOpts.Stdin = &api.StdinOptions{
		Contents:   source,
		ResolveDir: filepath.Dir(absFile),
		Sourcefile: filename,
		Loader:     api.LoaderJS,
	}

	result := api.Build(buildOpts)

	if len(result.Errors) > 0 {
		return Result{}, fmt.Errorf("transform: %s", result.Errors[0].Text)
	}
	if len(result.OutputFiles) == 0 {
		return Result{}, fmt.Errorf("transform: no output files generated")
	}

	emitted := string(result.OutputFiles[0].Contents)
	if len(generated) > 0 {
		wrapped, err := wrapDefaultExport(emitted)
		if err != nil {
			return Result{}, err
		}
		emitted = wrapped
	}

	return Result{Code: emitted, Artifacts: generated}, nil
}

var defaultExportPattern = regexp.MustCompile(`export\s*\{\s*([A-Za-z_$][\w$]*)\s+as\s+default\s*\};?\s*$`)

func wrapDefaultExport(source string) (string, error) {
	matches := defaultExportPattern.FindStringSubmatch(source)
	if len(matches) != 2 {
		return "", fmt.Errorf("distlang/core requires a default export worker object")
	}

	wrapped := defaultExportPattern.ReplaceAllString(source, "const __distlang_wrapped_default__ = wrapWorkerWithObjectDB($1);\nexport { __distlang_wrapped_default__ as default };")
	return wrapped, nil
}
