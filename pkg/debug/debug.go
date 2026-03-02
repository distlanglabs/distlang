package debug

import (
	"encoding/json"
	"fmt"

	"github.com/distlanglabs/distlang/pkg/parser"
	"github.com/distlanglabs/distlang/pkg/runtime"
	gojaengine "github.com/distlanglabs/distlang/pkg/runtime/goja"
	"github.com/dop251/goja/ast"
	goparser "github.com/dop251/goja/parser"
)

// Run executes debug passes and optionally runs the script.
// Pass names: parse, ir, emit (emit is currently a placeholder printing source).
func Run(filePath string, passes []string, execute bool) error {
	source, err := parser.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var parsed *ast.Program
	for _, pass := range passes {
		switch pass {
		case "parse":
			if parsed == nil {
				parsed, err = goparser.ParseFile(nil, filePath, source, 0)
				if err != nil {
					return fmt.Errorf("parse: %w", err)
				}
			}
			printParsePass(parsed)
		case "ir":
			ir, err := gojaengine.BuildIR(filePath, source)
			if err != nil {
				return fmt.Errorf("ir: %w", err)
			}
			data, err := json.MarshalIndent(ir, "", "  ")
			if err != nil {
				return fmt.Errorf("ir marshal: %w", err)
			}
			fmt.Println("== ir ==")
			fmt.Println(string(data))
		case "emit":
			fmt.Println("== emit (placeholder) ==")
			fmt.Print(source)
		default:
			return fmt.Errorf("unknown pass: %s", pass)
		}
	}

	if execute {
		engine := runtime.NewDefaultEngine()
		if err := engine.RunScript(filePath, source); err != nil {
			return fmt.Errorf("run: %w", err)
		}
	}

	return nil
}

func printParsePass(prog *ast.Program) {
	fmt.Println("== parse ==")
	fmt.Printf("statements: %d\n", len(prog.Body))
	for i, stmt := range prog.Body {
		fmt.Printf("[%d] %T\n", i, stmt)
	}
}
