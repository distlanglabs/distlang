package parse

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
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
	Code       string
	Artifacts  []artifacts.Artifact
	UsesLayers bool
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
	usesCore := usesDistlangCore(source)
	usesHelpers := usesDistlangHelpers(source)
	usesApp := usesDistlangApp(source)
	usesLayers := usesDistlangLayers(source)

	coreSource := helpgen.CoreInMemDB()
	helperModules := map[string]string{}
	if usesHelpers {
		moduleNames, err := helpgen.DistlangHelperModules()
		if err != nil {
			return Result{}, err
		}
		sort.Strings(moduleNames)
		for _, moduleName := range moduleNames {
			contents, err := helpgen.DistlangHelperModule(moduleName)
			if err != nil {
				return Result{}, err
			}
			helperModules[moduleName] = contents
		}
	}
	layersSource := helpgen.LayersSimpleApp()
	appSource := helpgen.AppIndex()
	if usesCore || usesHelpers {
		generated = append(generated, artifacts.Artifact{
			Path:    filepath.Join("generated", "distlang", "core", "index.js"),
			Content: []byte(coreSource),
		})
	}
	if usesHelpers {
		moduleNames := make([]string, 0, len(helperModules))
		for moduleName := range helperModules {
			moduleNames = append(moduleNames, moduleName)
		}
		sort.Strings(moduleNames)
		for _, moduleName := range moduleNames {
			generated = append(generated, artifacts.Artifact{
				Path:    filepath.Join("generated", "distlang", moduleName),
				Content: []byte(helperModules[moduleName]),
			})
		}
	}
	if usesLayers {
		generated = append(generated, artifacts.Artifact{
			Path:    filepath.Join("generated", "distlang", "layers", "index.js"),
			Content: []byte(layersSource),
		})
	}
	if usesApp {
		generated = append(generated, artifacts.Artifact{
			Path:    filepath.Join("generated", "distlang", "app", "index.js"),
			Content: []byte(appSource),
		})
	}

	if usesCore || usesHelpers || usesLayers || usesApp {
		buildOpts.Plugins = []api.Plugin{{
			Name: "distlang-modules",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `^distlang/core$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: "distlang/core", Namespace: "distlang-core"}, nil
				})
				build.OnResolve(api.OnResolveOptions{Filter: `^distlang$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: "index.js", Namespace: "distlang-helpers"}, nil
				})
				build.OnResolve(api.OnResolveOptions{Filter: `^\./.*`, Namespace: "distlang-helpers"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: strings.TrimPrefix(args.Path, "./"), Namespace: "distlang-helpers"}, nil
				})
				build.OnResolve(api.OnResolveOptions{Filter: `^distlang/layers$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: "distlang/layers", Namespace: "distlang-layers"}, nil
				})
				build.OnResolve(api.OnResolveOptions{Filter: `^distlang/app$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: "distlang/app", Namespace: "distlang-app"}, nil
				})
				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "distlang-core"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					loader := api.LoaderJS
					return api.OnLoadResult{Contents: &coreSource, Loader: loader}, nil
				})
				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "distlang-helpers"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					loader := api.LoaderJS
					contents, ok := helperModules[args.Path]
					if !ok {
						return api.OnLoadResult{}, fmt.Errorf("unknown distlang helper module: %s", args.Path)
					}
					return api.OnLoadResult{Contents: &contents, Loader: loader}, nil
				})
				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "distlang-layers"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					loader := api.LoaderJS
					return api.OnLoadResult{Contents: &layersSource, Loader: loader}, nil
				})
				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "distlang-app"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					loader := api.LoaderJS
					return api.OnLoadResult{Contents: &appSource, Loader: loader}, nil
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
	wrappers := []string{}
	if usesCore {
		wrappers = append(wrappers, "wrapWorkerWithInMemDB")
	}
	if usesHelpers {
		wrappers = append(wrappers, "wrapWorkerWithHelpers")
	}
	if len(wrappers) > 0 {
		wrapped, err := wrapDefaultExport(emitted, wrappers)
		if err != nil {
			return Result{}, err
		}
		emitted = wrapped
	}

	return Result{Code: emitted, Artifacts: generated, UsesLayers: usesLayers}, nil
}

var defaultExportPattern = regexp.MustCompile(`export\s*\{\s*([A-Za-z_$][\w$]*)\s+as\s+default\s*\};?\s*$`)

func usesDistlangCore(source string) bool {
	return strings.Contains(source, `from "distlang/core"`) || strings.Contains(source, "from 'distlang/core'")
}

func usesDistlangHelpers(source string) bool {
	return strings.Contains(source, `from "distlang"`) || strings.Contains(source, "from 'distlang'")
}

func usesDistlangApp(source string) bool {
	return strings.Contains(source, `from "distlang/app"`) || strings.Contains(source, "from 'distlang/app'")
}

func usesDistlangLayers(source string) bool {
	return strings.Contains(source, `from "distlang/layers"`) || strings.Contains(source, "from 'distlang/layers'")
}

func wrapDefaultExport(source string, wrappers []string) (string, error) {
	matches := defaultExportPattern.FindStringSubmatch(source)
	if len(matches) != 2 {
		return "", fmt.Errorf("distlang helper imports require a default export worker object")
	}
	if len(wrappers) == 0 {
		return source, nil
	}

	expr := matches[1]
	for _, wrapper := range wrappers {
		expr = wrapper + "(" + expr + ")"
	}

	wrapped := defaultExportPattern.ReplaceAllString(source, "const __distlang_wrapped_default__ = "+expr+";\nexport { __distlang_wrapped_default__ as default };")
	return wrapped, nil
}
