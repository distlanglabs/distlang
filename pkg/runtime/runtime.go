package runtime

import gojaengine "github.com/distlanglabs/distlang/pkg/runtime/goja"

// Engine executes JavaScript code.
type Engine interface {
	RunScript(filename, source string) error
}

// NewDefaultEngine returns the current default runtime engine.
// For Phase 0 this is backed by goja.
func NewDefaultEngine() Engine {
	return gojaengine.NewEngine()
}
