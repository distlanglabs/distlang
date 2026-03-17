package passes

import (
	"fmt"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/passes/emit"
	"github.com/distlanglabs/distlang/pkg/passes/ir"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
	"github.com/distlanglabs/distlang/pkg/passes/source"
)

// Result captures outputs from the compiler pipeline.
type Result struct {
	Source      string
	Transformed string
	IR          *ir.IR
	Emitted     string
	Artifacts   []artifacts.Artifact
	UsesLayers  bool
}

// Options controls pipeline execution.
type Options struct {
	Format parsepass.Format
	NeedIR bool
}

// Execute runs the compile pipeline and returns outputs.
func Execute(filePath string, opts Options) (Result, error) {
	var res Result

	src, err := source.ReadFile(filePath)
	if err != nil {
		return res, err
	}
	res.Source = src

	parsed, err := parsepass.ToScriptWithOptions(filePath, src, parsepass.Options{Format: opts.Format})
	if err != nil {
		return res, fmt.Errorf("parse: %w", err)
	}
	res.Transformed = parsed.Code
	res.Artifacts = parsed.Artifacts
	res.UsesLayers = parsed.UsesLayers

	if opts.NeedIR {
		built, err := ir.Build(filePath, parsed.Code)
		if err != nil {
			return res, fmt.Errorf("ir: %w", err)
		}
		res.IR = built
	}

	res.Emitted = emit.Source(parsed.Code)
	return res, nil
}
