package debug

import (
	"encoding/json"
	"fmt"

	"github.com/distlanglabs/distlang/pkg/passes"
	"github.com/distlanglabs/distlang/pkg/runtime"
)

// Run executes debug passes and optionally runs the script.
// Pass names: parse, ir, emit (emit is currently a placeholder printing source).
func Run(filePath string, passOrder []string, execute bool) error {
	needIR := contains(passOrder, "ir")

	result, err := passes.Execute(filePath, needIR)
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
		engine := runtime.NewDefaultEngine()
		if err := engine.RunScript(filePath, result.Emitted); err != nil {
			return fmt.Errorf("run: %w", err)
		}
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
