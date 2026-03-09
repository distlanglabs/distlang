package debug

import (
	"encoding/json"
	"fmt"

	"github.com/distlanglabs/distlang/pkg/passes"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
)

// Run executes debug passes for a V8 backend build.
// Pass names: parse, ir, emit.
func Run(filePath string, passOrder []string, execute bool) error {
	needIR := contains(passOrder, "ir")

	result, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatV8, NeedIR: needIR})
	if err != nil {
		return err
	}

	for _, pass := range passOrder {
		switch pass {
		case "parse":
			fmt.Println("== parse (transformed) ==")
			fmt.Print(result.Transformed)
		case "ir":
			data, err := json.MarshalIndent(result.IR, "", "  ")
			if err != nil {
				return fmt.Errorf("ir marshal: %w", err)
			}
			fmt.Println("== ir ==")
			fmt.Println(string(data))
		case "emit":
			fmt.Println("== emit ==")
			fmt.Print(result.Emitted)
		default:
			return fmt.Errorf("unknown pass: %s", pass)
		}
	}

	if execute {
		return fmt.Errorf("debug run is no longer available in-process; use `distlang run %s` to launch workerd", filePath)
	}

	return nil
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
