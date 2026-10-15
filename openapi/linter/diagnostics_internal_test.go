package linter

import (
	"errors"
	"testing"

	baseLinter "github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
)

func TestAttributeExternalDiagnostics_MissingInputs(t *testing.T) {
	t.Parallel()

	plain := errors.New("plain")
	tests := []struct {
		name    string
		docInfo *baseLinter.DocumentInfo[*openapi.OpenAPI]
	}{
		{name: "nil document info"},
		{name: "nil index", docInfo: &baseLinter.DocumentInfo[*openapi.OpenAPI]{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := &baseLinter.Output{Results: []error{plain}}
			attributeExternalDiagnostics(output, tt.docInfo)
			assert.Equal(t, []error{plain}, output.Results, "results should be left unchanged")
		})
	}
}

func TestAttributeExternalDiagnostics_NilOutput(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		attributeExternalDiagnostics(nil, &baseLinter.DocumentInfo[*openapi.OpenAPI]{})
	}, "nil output should be skipped safely")
}

func TestAttributeExternalDiagnostics_NonValidationErrorsPassThrough(t *testing.T) {
	t.Parallel()

	plain := errors.New("plain")
	output := &baseLinter.Output{Results: []error{plain}}
	docInfo := &baseLinter.DocumentInfo[*openapi.OpenAPI]{
		Index:    &openapi.Index{Doc: &openapi.OpenAPI{}},
		Location: "/spec/main.yaml",
	}

	attributeExternalDiagnostics(output, docInfo)

	assert.Equal(t, []error{plain}, output.Results, "errors without validation metadata should be kept as is")
}
