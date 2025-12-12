package game

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dop251/goja"
)

// WASMRunner executes JavaScript code using goja (pure Go JavaScript engine)
// Note: While named "WASMRunner" for interface compatibility, this uses goja
// which is a pure-Go JS engine. To use actual QuickJS WASM, see:
// https://github.com/aspect-build/nickel/go-quickjs-wasi for WASI-based execution.
type WASMRunner struct {
	mu sync.Mutex
}

// NewWASMRunner creates a new JavaScript runner with goja
func NewWASMRunner() (*WASMRunner, error) {
	return &WASMRunner{}, nil
}

// Close releases resources held by the runner
func (r *WASMRunner) Close() error {
	return nil
}

// ExecuteScript executes a JavaScript AI script with the given context
// The script should define a function called 'decide' that takes a context object
// and returns an AI decision object
func (r *WASMRunner) ExecuteScript(code string, aiCtx AIContext) (AIDecision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a new runtime for each execution (isolation)
	vm := goja.New()

	// Serialize context to JSON
	ctxJSON, err := json.Marshal(aiCtx)
	if err != nil {
		return AIDecision{}, fmt.Errorf("failed to serialize context: %w", err)
	}

	// Parse context JSON to JavaScript object
	ctxObj, err := vm.RunString(fmt.Sprintf("(%s)", string(ctxJSON)))
	if err != nil {
		return AIDecision{}, fmt.Errorf("failed to parse context: %w", err)
	}

	// Execute the user's script to define the decide function
	_, err = vm.RunString(code)
	if err != nil {
		return AIDecision{}, fmt.Errorf("script execution failed: %w", err)
	}

	// Get the decide function
	decideFunc, ok := goja.AssertFunction(vm.Get("decide"))
	if !ok {
		return AIDecision{}, fmt.Errorf("script must define a 'decide' function")
	}

	// Call the decide function with the context
	result, err := decideFunc(goja.Undefined(), ctxObj)
	if err != nil {
		return AIDecision{}, fmt.Errorf("decide function failed: %w", err)
	}

	// Convert result to JSON and then to AIDecision
	resultObj := result.Export()
	resultJSON, err := json.Marshal(resultObj)
	if err != nil {
		return AIDecision{}, fmt.Errorf("failed to serialize result: %w", err)
	}

	var decision AIDecision
	if err := json.Unmarshal(resultJSON, &decision); err != nil {
		return AIDecision{}, fmt.Errorf("failed to parse script result: %w (result: %s)", err, string(resultJSON))
	}

	return decision, nil
}

// ExecuteScriptRaw executes raw JavaScript code and returns the result as a string
func (r *WASMRunner) ExecuteScriptRaw(code string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	vm := goja.New()
	result, err := vm.RunString(code)
	if err != nil {
		return "", fmt.Errorf("script execution failed: %w", err)
	}

	return result.String(), nil
}

// ValidateScript checks if a script is valid JavaScript that defines a 'decide' function
func (r *WASMRunner) ValidateScript(code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vm := goja.New()

	// Try to run the script
	_, err := vm.RunString(code)
	if err != nil {
		return fmt.Errorf("script parse error: %w", err)
	}

	// Check for decide function
	decideFunc := vm.Get("decide")
	if decideFunc == nil || decideFunc == goja.Undefined() {
		return fmt.Errorf("script must define a 'decide' function")
	}

	if _, ok := goja.AssertFunction(decideFunc); !ok {
		return fmt.Errorf("'decide' must be a function")
	}

	return nil
}
