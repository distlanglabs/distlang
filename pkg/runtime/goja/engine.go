package gojaengine

import (
	"fmt"
	"io"
	"os"
	"strings"

	runtimetypes "github.com/distlanglabs/distlang/pkg/runtime/types"
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

// RunWorker loads the script, resolves the worker fetch, invokes it, and returns a Response.
func (e *Engine) RunWorker(filename, source string, req runtimetypes.Request) (runtimetypes.Response, error) {
	vm := goja.New()
	installConsole(vm, e.stdout)
	installResponse(vm)

	if _, err := vm.RunString(source); err != nil {
		return runtimetypes.Response{}, fmt.Errorf("run %s: %w", filename, err)
	}

	worker := vm.Get("distlangWorker")
	if goja.IsUndefined(worker) || goja.IsNull(worker) {
		return runtimetypes.Response{}, fmt.Errorf("worker export not found (distlangWorker)")
	}

	workerObj := worker.ToObject(vm)
	def := workerObj.Get("default")
	if goja.IsUndefined(def) || goja.IsNull(def) {
		return runtimetypes.Response{}, fmt.Errorf("worker default export not found")
	}

	defObj := def.ToObject(vm)
	fetchVal := defObj.Get("fetch")
	fn, ok := goja.AssertFunction(fetchVal)
	if !ok {
		return runtimetypes.Response{}, fmt.Errorf("worker fetch is not a function")
	}

	jsReq := vm.ToValue(map[string]interface{}{
		"url":     req.URL,
		"method":  req.Method,
		"headers": req.Headers,
		"body":    req.Body,
	})
	jsEnv := vm.NewObject()
	jsCtx := vm.NewObject()
	jsCtx.Set("waitUntil", func(goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})

	result, err := fn(def, jsReq, jsEnv, jsCtx)
	if err != nil {
		return runtimetypes.Response{}, fmt.Errorf("fetch call: %w", err)
	}

	if _, ok := result.Export().(*goja.Promise); ok {
		resolved, err := awaitPromise(vm, result)
		if err != nil {
			return runtimetypes.Response{}, err
		}
		result = resolved
	}

	resp, err := exportResponse(vm, result)
	if err != nil {
		return runtimetypes.Response{}, err
	}

	return resp, nil
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

func installResponse(vm *goja.Runtime) {
	constructor := func(call goja.ConstructorCall) *goja.Object {
		body := ""
		if len(call.Arguments) > 0 {
			body = call.Arguments[0].String()
		}

		status := 200
		if len(call.Arguments) > 1 {
			initObj := call.Arguments[1].ToObject(vm)
			if initObj != nil {
				if v := initObj.Get("status"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
					if n, ok := v.Export().(int64); ok {
						status = int(n)
					}
				}
			}
		}

		obj := vm.NewObject()
		obj.Set("status", status)
		obj.Set("body", body)
		obj.Set("headers", map[string]string{})
		obj.Set("text", func(goja.FunctionCall) goja.Value { return vm.ToValue(body) })
		obj.Set("json", func(goja.FunctionCall) goja.Value {
			return vm.ToValue(body)
		})
		return obj
	}

	vm.Set("Response", constructor)
}

func awaitPromise(vm *goja.Runtime, v goja.Value) (goja.Value, error) {
	if p, ok := v.Export().(*goja.Promise); ok {
		if p.State() == goja.PromiseStatePending {
			// Allow any queued jobs to run to settle the promise.
			if _, err := vm.RunString("void 0"); err != nil {
				return nil, err
			}
		}
		switch p.State() {
		case goja.PromiseStateFulfilled:
			return p.Result(), nil
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejection: %v", p.Result())
		}
	}
	return v, nil
}

func exportResponse(vm *goja.Runtime, val goja.Value) (runtimetypes.Response, error) {
	obj := val.ToObject(vm)
	if obj == nil {
		return runtimetypes.Response{}, fmt.Errorf("worker fetch returned non-object")
	}

	status := 200
	if v := obj.Get("status"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		switch n := v.Export().(type) {
		case int64:
			status = int(n)
		case int32:
			status = int(n)
		case int:
			status = n
		}
	}

	body := ""
	if v := obj.Get("body"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		body = v.String()
	} else if v := obj.Get("text"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if fn, ok := goja.AssertFunction(v); ok {
			if out, err := fn(obj); err == nil {
				body = out.String()
			}
		}
	}

	headers := map[string]string{}
	if hv := obj.Get("headers"); hv != nil && !goja.IsUndefined(hv) && !goja.IsNull(hv) {
		if m, ok := hv.Export().(map[string]interface{}); ok {
			for k, v := range m {
				headers[k] = fmt.Sprint(v)
			}
		}
	}

	return runtimetypes.Response{Status: status, Headers: headers, Body: body}, nil
}
