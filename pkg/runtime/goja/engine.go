package gojaengine

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dop251/goja"
)

// Engine executes JavaScript using goja.
// It is intentionally minimal for Phase 0.
type Engine struct {
	stdout io.Writer
}

func NewEngine() *Engine {
	return &Engine{stdout: os.Stdout}
}

// RunScript executes the provided JavaScript source.
func (e *Engine) RunScript(filename, source string) error {
	vm := goja.New()
	installConsole(vm, e.stdout)

	_, err := vm.RunString(source)
	if err != nil {
		return fmt.Errorf("run %s: %w", filename, err)
	}
	return nil
}

func installConsole(vm *goja.Runtime, out io.Writer) {
	console := map[string]func(goja.FunctionCall) goja.Value{
		"log": func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = arg.String()
			}
			fmt.Fprintln(out, strings.Join(parts, " "))
			return goja.Undefined()
		},
	}

	vm.Set("console", console)
}
