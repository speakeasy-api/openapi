package customrules

import (
	"context"
	"errors"
	"fmt"

	"github.com/dop251/goja"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// Runtime wraps a goja JavaScript runtime with custom rule support.
// Each Runtime instance is NOT thread-safe and should only be used
// from a single goroutine.
type Runtime struct {
	vm     *goja.Runtime
	logger Logger
	config *Config

	// registeredRules holds rules registered via registerRule()
	registeredRules []goja.Value
}

// NewRuntime creates a new JavaScript runtime configured for custom rules.
func NewRuntime(logger Logger, config *Config) (*Runtime, error) {
	vm := goja.New()

	// Use uncapitalized field/method mapper for JS-style naming
	// This exposes Go fields and methods with their first letter lowercased
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	rt := &Runtime{
		vm:     vm,
		logger: logger,
		config: config,
	}

	// Set up console object
	if err := rt.setupConsole(); err != nil {
		return nil, fmt.Errorf("setting up console: %w", err)
	}

	// Set up global functions
	if err := rt.setupGlobals(); err != nil {
		return nil, fmt.Errorf("setting up globals: %w", err)
	}

	return rt, nil
}

// setupConsole creates the console object with log, warn, error methods.
func (rt *Runtime) setupConsole() error {
	console := rt.vm.NewObject()

	if err := console.Set("log", func(call goja.FunctionCall) goja.Value {
		rt.logger.Log(rt.formatArgs(call.Arguments)...)
		return goja.Undefined()
	}); err != nil {
		return err
	}

	if err := console.Set("warn", func(call goja.FunctionCall) goja.Value {
		rt.logger.Warn(rt.formatArgs(call.Arguments)...)
		return goja.Undefined()
	}); err != nil {
		return err
	}

	if err := console.Set("error", func(call goja.FunctionCall) goja.Value {
		rt.logger.Error(rt.formatArgs(call.Arguments)...)
		return goja.Undefined()
	}); err != nil {
		return err
	}

	return rt.vm.Set("console", console)
}

// formatArgs converts goja values to strings for logging.
func (rt *Runtime) formatArgs(args []goja.Value) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		result[i] = arg.Export()
	}
	return result
}

// setupGlobals sets up global functions available to custom rules.
func (rt *Runtime) setupGlobals() error {
	// registerRule(rule) - registers a rule class instance
	if err := rt.vm.Set("registerRule", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(rt.vm.ToValue("registerRule requires a rule argument"))
		}
		rt.registeredRules = append(rt.registeredRules, call.Arguments[0])
		return goja.Undefined()
	}); err != nil {
		return err
	}

	// createValidationError(severity, ruleId, message, node) - creates a validation error
	if err := rt.vm.Set("createValidationError", rt.createValidationError); err != nil {
		return err
	}

	// createFix(options) - creates a fix object for attaching to validation errors
	if err := rt.vm.Set("createFix", rt.createFix); err != nil {
		return err
	}

	// createValidationErrorWithFix(severity, ruleId, message, node, fix) - creates a validation error with a fix
	if err := rt.vm.Set("createValidationErrorWithFix", rt.createValidationErrorWithFix); err != nil {
		return err
	}

	return nil
}

// createValidationError is the JS-callable function for creating validation errors.
func (rt *Runtime) createValidationError(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 4 {
		panic(rt.vm.ToValue("createValidationError requires 4 arguments: severity, ruleId, message, node"))
	}

	severityStr := call.Arguments[0].String()
	ruleID := call.Arguments[1].String()
	message := call.Arguments[2].String()
	nodeVal := call.Arguments[3].Export()

	// Parse severity
	severity := parseSeverity(severityStr)

	// Get the yaml.Node if provided
	var node *yaml.Node
	if nodeVal != nil {
		if n, ok := nodeVal.(*yaml.Node); ok {
			node = n
		}
	}

	// Create the validation error
	err := validation.NewValidationError(
		severity,
		ruleID,
		errors.New(message),
		node,
	)

	return rt.vm.ToValue(err)
}

// createFix is the JS-callable function for creating a fix object.
func (rt *Runtime) createFix(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(rt.vm.ToValue("createFix requires an options argument"))
	}

	fix, err := newJSFix(rt, rt.config, call.Arguments[0])
	if err != nil {
		panic(rt.vm.ToValue(err.Error()))
	}

	return rt.vm.ToValue(fix)
}

// createValidationErrorWithFix is the JS-callable function for creating validation errors with fixes.
func (rt *Runtime) createValidationErrorWithFix(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 5 {
		panic(rt.vm.ToValue("createValidationErrorWithFix requires 5 arguments: severity, ruleId, message, node, fix"))
	}

	severityStr := call.Arguments[0].String()
	ruleID := call.Arguments[1].String()
	message := call.Arguments[2].String()
	nodeVal := call.Arguments[3].Export()
	fixVal := call.Arguments[4].Export()

	severity := parseSeverity(severityStr)

	var node *yaml.Node
	if nodeVal != nil {
		if n, ok := nodeVal.(*yaml.Node); ok {
			node = n
		}
	}

	vErr := &validation.Error{
		UnderlyingError: errors.New(message),
		Node:            node,
		Severity:        severity,
		Rule:            ruleID,
	}

	if fix, ok := fixVal.(validation.Fix); ok {
		vErr.Fix = fix
	}

	return rt.vm.ToValue(vErr)
}

// parseSeverity converts a string to validation.Severity.
func parseSeverity(s string) validation.Severity {
	switch s {
	case "error":
		return validation.SeverityError
	case "warning":
		return validation.SeverityWarning
	case "hint":
		return validation.SeverityHint
	default:
		return validation.SeverityError
	}
}

// RunScript executes JavaScript code in the runtime.
func (rt *Runtime) RunScript(name, code string) (goja.Value, error) {
	return rt.vm.RunScript(name, code)
}

// GetRegisteredRules returns all rules registered via registerRule().
func (rt *Runtime) GetRegisteredRules() []goja.Value {
	return rt.registeredRules
}

// ClearRegisteredRules clears the registered rules list.
func (rt *Runtime) ClearRegisteredRules() {
	rt.registeredRules = nil
}

// ToValue converts a Go value to a goja value.
func (rt *Runtime) ToValue(v any) goja.Value {
	return rt.vm.ToValue(v)
}

// Interrupt interrupts the currently running JavaScript.
// This is used for timeout handling.
func (rt *Runtime) Interrupt(reason string) {
	rt.vm.Interrupt(reason)
}

// ClearInterrupt clears any pending interrupt.
func (rt *Runtime) ClearInterrupt() {
	rt.vm.ClearInterrupt()
}

// CallMethod calls a method on a JavaScript object.
func (rt *Runtime) CallMethod(obj goja.Value, method string, args ...goja.Value) (goja.Value, error) {
	objVal := obj.ToObject(rt.vm)
	if objVal == nil {
		return nil, errors.New("object is nil")
	}

	methodVal := objVal.Get(method)
	if methodVal == nil || goja.IsUndefined(methodVal) {
		return nil, fmt.Errorf("method %s not found", method)
	}

	callable, ok := goja.AssertFunction(methodVal)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", method)
	}

	return callable(obj, args...)
}

// BridgedContext wraps a Go context for JavaScript access.
type BridgedContext struct {
	ctx context.Context
}

// NewBridgedContext creates a JavaScript-accessible context wrapper.
func NewBridgedContext(ctx context.Context) *BridgedContext {
	return &BridgedContext{ctx: ctx}
}

// IsCancelled returns true if the context has been cancelled.
func (bc *BridgedContext) IsCancelled() bool {
	select {
	case <-bc.ctx.Done():
		return true
	default:
		return false
	}
}

// Deadline returns the deadline in milliseconds since epoch, or undefined if no deadline.
func (bc *BridgedContext) Deadline() any {
	deadline, ok := bc.ctx.Deadline()
	if !ok {
		return nil
	}
	return deadline.UnixMilli()
}
