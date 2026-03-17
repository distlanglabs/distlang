package v8

import (
	"path/filepath"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/passes"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
)

// Output captures V8-ready build output.
type WorkerOutput struct {
	Name      string
	EntryPath string
	Emitted   string
}

type Output struct {
	EntryPath string
	Emitted   string
	Artifacts []artifacts.Artifact
	Workers   []WorkerOutput
}

// Build compiles a distlang program into V8-ready JS.
func Build(filePath string) (Output, error) {
	out, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatV8})
	if err != nil {
		return Output{}, err
	}

	if out.UsesLayers {
		workers := []WorkerOutput{
			{
				Name:      "handlerSet1",
				EntryPath: filepath.Join("dist", "v8", "handlerSet1", "worker.js"),
				Emitted:   "globalThis.__DISTLANG_SIMPLEAPP_TARGET__ = \"handlerSet1\";\n" + out.Emitted,
			},
			{
				Name:      "handlerSet2",
				EntryPath: filepath.Join("dist", "v8", "handlerSet2", "worker.js"),
				Emitted:   "globalThis.__DISTLANG_SIMPLEAPP_TARGET__ = \"handlerSet2\";\n" + out.Emitted,
			},
		}

		items := append([]artifacts.Artifact{}, out.Artifacts...)
		for _, worker := range workers {
			items = append(items, artifacts.Artifact{Path: worker.EntryPath, Content: []byte(worker.Emitted)})
		}

		return Output{
			EntryPath: workers[0].EntryPath,
			Emitted:   workers[0].Emitted,
			Artifacts: items,
			Workers:   workers,
		}, nil
	}

	entry := filepath.Join("dist", "v8", "worker.js")
	artifact := artifacts.Artifact{Path: entry, Content: []byte(out.Emitted)}
	items := append([]artifacts.Artifact{}, out.Artifacts...)
	items = append(items, artifact)
	return Output{EntryPath: entry, Emitted: out.Emitted, Artifacts: items}, nil
}
