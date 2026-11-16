package arazzo

import (
	"fmt"
	"strings"
)

// workflowInputs describes the inputs declared by a workflow's `inputs` JSON Schema.
type workflowInputs struct {
	names    map[string]bool
	required []string
	// known is false when the inputs can't be determined statically, for example when the schema is a `$ref`, is
	// composed with allOf/anyOf/oneOf, or explicitly allows additional properties. References to inputs aren't
	// checked in that case.
	known bool
}

// getWorkflowInputs returns the inputs declared by the workflow. Inputs are the top level properties of the
// workflow's `inputs` schema. A workflow without `inputs` declares no inputs.
func getWorkflowInputs(w *Workflow) workflowInputs {
	inputs := workflowInputs{names: map[string]bool{}}

	if w.Inputs == nil {
		inputs.known = true
		return inputs
	}

	s := w.Inputs.GetSchema()
	if s == nil || s.IsReference() || len(s.GetAllOf()) > 0 || len(s.GetAnyOf()) > 0 || len(s.GetOneOf()) > 0 || s.GetPatternProperties().Len() > 0 {
		return inputs
	}

	// Inputs are meant to be declared, so an absent additionalProperties is treated as declaring only the listed
	// properties. Explicitly allowing additional properties makes any input name possible.
	if ap := s.GetAdditionalProperties(); ap != nil && (!ap.IsBool() || *ap.GetBool()) {
		return inputs
	}

	for name := range s.GetProperties().All() {
		inputs.names[name] = true
	}
	inputs.required = s.Required
	inputs.known = true

	return inputs
}

// has returns true if the input is declared. As a dotted name may refer to a nested property of an input (for
// example `$inputs.user.name`), a name is declared if it or any of its dotted prefixes is an input.
func (i workflowInputs) has(name string) bool {
	return hasDottedName(i.names, name)
}

func hasDottedName(names map[string]bool, name string) bool {
	for {
		if names[name] {
			return true
		}
		i := strings.LastIndex(name, ".")
		if i < 0 {
			return false
		}
		name = name[:i]
	}
}

// indexedError is an error for the entry at index within a list.
type indexedError struct {
	index int
	err   error
}

// inputParameterErrors are the errors found when checking parameters passed to a workflow against its inputs.
type inputParameterErrors struct {
	// unknown are the errors for parameters that aren't inputs of the workflow.
	unknown []indexedError
	// missing are the errors for required inputs that aren't supplied by a parameter.
	missing []error
}

// checkWorkflowInputParameters checks that parameters passed to the target workflow (by a workflow step or an
// action) are declared inputs of the workflow and that all its required inputs are supplied.
func checkWorkflowInputParameters(parentName string, a *Arazzo, parameters []*ReusableParameter, target *Workflow) inputParameterErrors {
	result := inputParameterErrors{}

	inputs := getWorkflowInputs(target)
	if !inputs.known {
		return result
	}

	var components *Components
	if a != nil {
		components = a.Components
	}

	supplied := map[string]bool{}
	for i, parameter := range parameters {
		if parameter == nil {
			continue
		}
		p := parameter.Get(components)
		if p == nil || p.Name == "" {
			continue
		}
		supplied[p.Name] = true

		if !inputs.has(p.Name) {
			result.unknown = append(result.unknown, indexedError{index: i, err: fmt.Errorf("%s.parameters `%s` is not an input of workflow `%s`", parentName, p.Name, target.WorkflowID)})
		}
	}

	for _, required := range inputs.required {
		if !supplied[required] {
			result.missing = append(result.missing, fmt.Errorf("%s is missing required input `%s` of workflow `%s`", parentName, required, target.WorkflowID))
		}
	}

	return result
}
