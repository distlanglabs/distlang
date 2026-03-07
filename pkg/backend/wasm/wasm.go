package wasm

import (
	"encoding/json"
	"path/filepath"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/passes"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
)

// Output captures the current Wasm backend workspace.
type Output struct {
	EntryPath string
	Emitted   string
	Artifacts []artifacts.Artifact
}

type manifest struct {
	Runtime string `json:"runtime"`
	Entry   string `json:"entry"`
	Note    string `json:"note"`
}

// Build assembles the current Wasm backend workspace.
func Build(filePath string) (Output, error) {
	out, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatWasm})
	if err != nil {
		return Output{}, err
	}

	entry := filepath.Join("dist", "wasm", "module.js")
	meta, err := json.MarshalIndent(manifest{
		Runtime: "wasmtime",
		Entry:   filepath.Base(entry),
		Note:    "Wasm backend workspace placeholder until distlang lowers IR to a runnable Wasm artifact.",
	}, "", "  ")
	if err != nil {
		return Output{}, err
	}

	items := []artifacts.Artifact{
		{Path: entry, Content: []byte(out.Emitted)},
		{Path: filepath.Join("dist", "wasm", "build.json"), Content: append(meta, '\n')},
	}

	return Output{EntryPath: entry, Emitted: out.Emitted, Artifacts: items}, nil
}
