package customrules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-sourcemap/sourcemap"
	"github.com/speakeasy-api/openapi/internal/utils"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

// CustomRule wraps a JavaScript rule object and implements RuleRunner.
type CustomRule struct {
	runtime    *Runtime
	jsRule     goja.Value
	sourceFile string
	sourceMap  *sourcemap.Consumer
	config     *Config

	// Cached metadata from JS
	id          string
	category    string
	description string
	summary     string
	link        string
	severity    validation.Severity
	versions    []string
}

// NewCustomRule creates a new CustomRule from a JavaScript rule object.
func NewCustomRule(rt *Runtime, jsRule goja.Value, sourceFile string, sm *sourcemap.Consumer, config *Config) (*CustomRule, error) {
	rule := &CustomRule{
		runtime:    rt,
		jsRule:     jsRule,
		sourceFile: sourceFile,
		sourceMap:  sm,
		config:     config,
	}

	// Extract and cache metadata
	if err := rule.extractMetadata(); err != nil {
		return nil, err
	}

	return rule, nil
}

// extractMetadata calls JS methods to extract rule metadata.
func (r *CustomRule) extractMetadata() error {
	// ID (required)
	id, err := r.callStringMethod("id")
	if err != nil {
		return fmt.Errorf("getting rule id: %w", err)
	}
	if id == "" {
		return fmt.Errorf("rule id() returned empty string")
	}
	r.id = id

	// Category (required)
	category, err := r.callStringMethod("category")
	if err != nil {
		return fmt.Errorf("getting rule category: %w", err)
	}
	r.category = category

	// Description (required)
	description, err := r.callStringMethod("description")
	if err != nil {
		return fmt.Errorf("getting rule description: %w", err)
	}
	r.description = description

	// Summary (required)
	summary, err := r.callStringMethod("summary")
	if err != nil {
		return fmt.Errorf("getting rule summary: %w", err)
	}
	r.summary = summary

	// Link (optional, defaults to empty)
	link, _ := r.callStringMethod("link")
	r.link = link

	// DefaultSeverity (optional, defaults to warning)
	severityStr, _ := r.callStringMethod("defaultSeverity")
	r.severity = parseSeverity(severityStr)
	if severityStr == "" {
		r.severity = validation.SeverityWarning
	}

	// Versions (optional, nil means all versions)
	versions, _ := r.callStringArrayMethod("versions")
	r.versions = versions

	return nil
}

// callStringMethod calls a method on the JS rule that returns a string.
func (r *CustomRule) callStringMethod(method string) (string, error) {
	result, err := r.runtime.CallMethod(r.jsRule, method)
	if err != nil {
		return "", err
	}
	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return "", nil
	}
	return result.String(), nil
}

// callStringArrayMethod calls a method that returns a string array.
func (r *CustomRule) callStringArrayMethod(method string) ([]string, error) {
	result, err := r.runtime.CallMethod(r.jsRule, method)
	if err != nil {
		return nil, err
	}
	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return nil, nil
	}

	exported := result.Export()
	if arr, ok := exported.([]interface{}); ok {
		strings := make([]string, len(arr))
		for i, v := range arr {
			strings[i] = utils.AnyToString(v)
		}
		return strings, nil
	}

	return nil, nil
}

// RuleRunner interface implementation

func (r *CustomRule) ID() string                         { return r.id }
func (r *CustomRule) Category() string                   { return r.category }
func (r *CustomRule) Description() string                { return r.description }
func (r *CustomRule) Summary() string                    { return r.summary }
func (r *CustomRule) Link() string                       { return r.link }
func (r *CustomRule) DefaultSeverity() validation.Severity { return r.severity }
func (r *CustomRule) Versions() []string                 { return r.versions }

// Run executes the JavaScript rule against the document.
func (r *CustomRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	// Set up timeout
	timeout := r.config.GetTimeout()
	timer := time.AfterFunc(timeout, func() {
		r.runtime.Interrupt(fmt.Sprintf("rule %s: execution timeout exceeded (%v)", r.id, timeout))
	})
	defer timer.Stop()
	defer r.runtime.ClearInterrupt()

	// Create bridged context for JS
	bridgedCtx := NewBridgedContext(ctx)

	// Create bridged config helper
	configHelper := &ruleConfigHelper{config: config}

	// Call the run method
	runMethod := r.jsRule.ToObject(r.runtime.vm).Get("run")
	if runMethod == nil || goja.IsUndefined(runMethod) {
		return []error{fmt.Errorf("rule %s has no run method", r.id)}
	}

	callable, ok := goja.AssertFunction(runMethod)
	if !ok {
		return []error{fmt.Errorf("rule %s: run is not a function", r.id)}
	}

	// Call: rule.run(ctx, docInfo, config)
	result, err := callable(
		r.jsRule,
		r.runtime.ToValue(bridgedCtx),
		r.runtime.ToValue(docInfo),
		r.runtime.ToValue(configHelper),
	)

	if err != nil {
		return r.handleError(err)
	}

	// Convert result array to []error
	return r.convertErrors(result)
}

// handleError handles errors from JS execution.
func (r *CustomRule) handleError(err error) []error {
	// Check for goja exception
	if exc, ok := err.(*goja.Exception); ok {
		mappedErr := MapException(exc, r.sourceFile, r.sourceMap)
		return []error{fmt.Errorf("rule %s: %w", r.id, mappedErr)}
	}

	// Check for timeout/interrupt
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "interrupt") {
		return []error{fmt.Errorf("rule %s: execution timeout exceeded (%v)", r.id, r.config.GetTimeout())}
	}

	return []error{fmt.Errorf("rule %s: %w", r.id, err)}
}

// convertErrors converts the JS result array to Go errors.
func (r *CustomRule) convertErrors(result goja.Value) []error {
	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return nil
	}

	exported := result.Export()
	if exported == nil {
		return nil
	}

	arr, ok := exported.([]interface{})
	if !ok {
		return nil
	}

	var errors []error
	for _, item := range arr {
		if err, ok := item.(error); ok {
			errors = append(errors, err)
		} else if verr, ok := item.(*validation.Error); ok {
			errors = append(errors, verr)
		}
	}

	return errors
}

// ruleConfigHelper provides JS-friendly access to RuleConfig.
type ruleConfigHelper struct {
	config *linter.RuleConfig
}

// GetSeverity returns the effective severity as a string.
func (h *ruleConfigHelper) GetSeverity(defaultSeverity string) string {
	defSev := parseSeverity(defaultSeverity)
	if h.config == nil {
		return string(defSev)
	}
	return string(h.config.GetSeverity(defSev))
}

// Enabled returns whether the rule is enabled.
func (h *ruleConfigHelper) Enabled() bool {
	if h.config == nil || h.config.Enabled == nil {
		return true
	}
	return *h.config.Enabled
}
