package debug

import (
	"encoding/json"
	"fmt"

	"github.com/distlanglabs/distlang/pkg/passes"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
	"github.com/distlanglabs/distlang/pkg/runtime"
	runtimetypes "github.com/distlanglabs/distlang/pkg/runtime/types"
)

// Run executes debug passes and optionally runs the script.
// Pass names: parse, ir, emit (emit is currently a placeholder printing source).
func Run(filePath string, passOrder []string, execute bool) error {
	needIR := contains(passOrder, "ir")

	result, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatGoja, NeedIR: needIR})
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
		resp, err := engine.RunWorker(filePath, result.Emitted, runtimetypes.Request{URL: "http://localhost/", Method: "GET", Headers: map[string]string{"host": "localhost"}})
		if err != nil {
			return fmt.Errorf("run: %w", err)
		}
		fmt.Printf("== run ==\nstatus: %d\n%s\n", resp.Status, resp.Body)
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
