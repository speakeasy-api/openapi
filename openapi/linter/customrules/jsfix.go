package customrules

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/speakeasy-api/openapi/validation"
)

// JSFix bridges a JavaScript fix object to the Go Fix interface.
type JSFix struct {
	rt          *Runtime
	config      *Config
	description string
	interactive bool
	prompts     []validation.Prompt
	applyFn     goja.Callable
	jsFixObj    goja.Value
	inputs      []string
}

func (f *JSFix) Description() string             { return f.description }
func (f *JSFix) Interactive() bool                { return f.interactive }
func (f *JSFix) Prompts() []validation.Prompt     { return f.prompts }

func (f *JSFix) SetInput(responses []string) error {
	if len(responses) != len(f.prompts) {
		return fmt.Errorf("expected %d responses, got %d", len(f.prompts), len(responses))
	}
	f.inputs = responses
	return nil
}

func (f *JSFix) Apply(doc any) error {
	// Set up timeout using the same pattern as rule.Run()
	timeout := f.config.GetTimeout()
	timer := time.AfterFunc(timeout, func() {
		f.rt.Interrupt("fix apply timeout exceeded")
	})
	defer timer.Stop()
	defer f.rt.ClearInterrupt()

	args := []goja.Value{f.rt.ToValue(doc)}

	if f.interactive && len(f.inputs) > 0 {
		jsInputs := make([]interface{}, len(f.inputs))
		for i, input := range f.inputs {
			jsInputs[i] = input
		}
		args = append(args, f.rt.ToValue(jsInputs))
	}

	_, err := f.applyFn(f.jsFixObj, args...)
	if err != nil {
		return fmt.Errorf("fix apply error: %w", err)
	}
	return nil
}

// newJSFix creates a JSFix from a JavaScript options object.
// Expected JS shape: { description: string, interactive?: bool, prompts?: [...], apply: (doc, inputs?) => void }
func newJSFix(rt *Runtime, config *Config, optionsVal goja.Value) (*JSFix, error) {
	obj := optionsVal.ToObject(rt.vm)
	if obj == nil {
		return nil, fmt.Errorf("createFix: argument must be an object")
	}

	// description (required)
	descVal := obj.Get("description")
	if descVal == nil || goja.IsUndefined(descVal) {
		return nil, fmt.Errorf("createFix: description is required")
	}

	// apply (required)
	applyVal := obj.Get("apply")
	if applyVal == nil || goja.IsUndefined(applyVal) {
		return nil, fmt.Errorf("createFix: apply function is required")
	}
	applyFn, ok := goja.AssertFunction(applyVal)
	if !ok {
		return nil, fmt.Errorf("createFix: apply must be a function")
	}

	fix := &JSFix{
		rt:          rt,
		config:      config,
		description: descVal.String(),
		applyFn:     applyFn,
		jsFixObj:    optionsVal,
	}

	// interactive (optional)
	interVal := obj.Get("interactive")
	if interVal != nil && !goja.IsUndefined(interVal) {
		fix.interactive = interVal.ToBoolean()
	}

	// prompts (optional)
	promptsVal := obj.Get("prompts")
	if promptsVal != nil && !goja.IsUndefined(promptsVal) && !goja.IsNull(promptsVal) {
		prompts, err := parseJSPrompts(promptsVal)
		if err != nil {
			return nil, fmt.Errorf("createFix: %w", err)
		}
		fix.prompts = prompts
	}

	return fix, nil
}

// parseJSPrompts converts a JS array of prompt objects to Go Prompt slice.
func parseJSPrompts(val goja.Value) ([]validation.Prompt, error) {
	exported := val.Export()
	arr, ok := exported.([]interface{})
	if !ok {
		return nil, fmt.Errorf("prompts must be an array")
	}

	prompts := make([]validation.Prompt, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("prompts[%d] must be an object", i)
		}

		prompt := validation.Prompt{}

		// type
		if t, ok := m["type"].(string); ok {
			switch t {
			case "choice":
				prompt.Type = validation.PromptChoice
			case "text":
				prompt.Type = validation.PromptFreeText
			default:
				return nil, fmt.Errorf("prompts[%d]: unknown type %q (use \"choice\" or \"text\")", i, t)
			}
		}

		// message
		if msg, ok := m["message"].(string); ok {
			prompt.Message = msg
		}

		// default
		if def, ok := m["default"].(string); ok {
			prompt.Default = def
		}

		// choices (for choice type)
		if choices, ok := m["choices"].([]interface{}); ok {
			for _, c := range choices {
				if s, ok := c.(string); ok {
					prompt.Choices = append(prompt.Choices, s)
				}
			}
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}
