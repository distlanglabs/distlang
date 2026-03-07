package v8

import (
	"path/filepath"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/passes"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
)

// Output captures V8-ready build output.
type Output struct {
	EntryPath string
	Emitted   string
	Artifacts []artifacts.Artifact
}

// Build compiles a distlang program into V8-ready JS.
func Build(filePath string) (Output, error) {
	out, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatV8})
	if err != nil {
		return Output{}, err
	}

	entry := filepath.Join("dist", "v8", "worker.js")
	artifact := artifacts.Artifact{Path: entry, Content: []byte(out.Emitted)}
	return Output{EntryPath: entry, Emitted: out.Emitted, Artifacts: []artifacts.Artifact{artifact}}, nil
}
