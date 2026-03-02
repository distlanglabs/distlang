package passes

import (
	"fmt"

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
}

// Execute runs the compile pipeline and returns outputs. If needIR is true, the
// IR stage is also executed.
func Execute(filePath string, needIR bool) (Result, error) {
	var res Result

	src, err := source.ReadFile(filePath)
	if err != nil {
		return res, err
	}
	res.Source = src

	transformed, err := parsepass.ToScript(filePath, src)
	if err != nil {
		return res, fmt.Errorf("parse: %w", err)
	}
	res.Transformed = transformed

	if needIR {
		built, err := ir.Build(filePath, transformed)
		if err != nil {
			return res, fmt.Errorf("ir: %w", err)
		}
		res.IR = built
	}

	res.Emitted = emit.Source(transformed)
	return res, nil
}
