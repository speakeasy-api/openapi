package swagger_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/assert"
)

func TestOperation_GetTags_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected []string
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty tags returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns tags",
			op:       &swagger.Operation{Tags: []string{"tag1", "tag2"}},
			expected: []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetTags())
		})
	}
}

func TestOperation_GetSummary_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected string
	}{
		{
			name:     "nil operation returns empty string",
			op:       nil,
			expected: "",
		},
		{
			name:     "nil Summary returns empty string",
			op:       &swagger.Operation{},
			expected: "",
		},
		{
			name:     "returns Summary value",
			op:       &swagger.Operation{Summary: pointer.From("Test summary")},
			expected: "Test summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetSummary())
		})
	}
}

func TestOperation_GetDescription_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected string
	}{
		{
			name:     "nil operation returns empty string",
			op:       nil,
			expected: "",
		},
		{
			name:     "nil Description returns empty string",
			op:       &swagger.Operation{},
			expected: "",
		},
		{
			name:     "returns Description value",
			op:       &swagger.Operation{Description: pointer.From("Test description")},
			expected: "Test description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetDescription())
		})
	}
}

func TestOperation_GetExternalDocs_Success(t *testing.T) {
	t.Parallel()

	docs := &swagger.ExternalDocumentation{URL: "https://docs.example.com"}
	tests := []struct {
		name     string
		op       *swagger.Operation
		expected *swagger.ExternalDocumentation
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "nil ExternalDocs returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns ExternalDocs value",
			op:       &swagger.Operation{ExternalDocs: docs},
			expected: docs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetExternalDocs())
		})
	}
}

func TestOperation_GetOperationID_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected string
	}{
		{
			name:     "nil operation returns empty string",
			op:       nil,
			expected: "",
		},
		{
			name:     "nil OperationID returns empty string",
			op:       &swagger.Operation{},
			expected: "",
		},
		{
			name:     "returns OperationID value",
			op:       &swagger.Operation{OperationID: pointer.From("getUser")},
			expected: "getUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetOperationID())
		})
	}
}

func TestOperation_GetConsumes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected []string
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty Consumes returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns Consumes value",
			op:       &swagger.Operation{Consumes: []string{"application/json", "application/xml"}},
			expected: []string{"application/json", "application/xml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetConsumes())
		})
	}
}

func TestOperation_GetProduces_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected []string
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty Produces returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns Produces value",
			op:       &swagger.Operation{Produces: []string{"application/json"}},
			expected: []string{"application/json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetProduces())
		})
	}
}

func TestOperation_GetParameters_Success(t *testing.T) {
	t.Parallel()

	params := []*swagger.ReferencedParameter{{}}
	tests := []struct {
		name     string
		op       *swagger.Operation
		expected []*swagger.ReferencedParameter
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty Parameters returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns Parameters value",
			op:       &swagger.Operation{Parameters: params},
			expected: params,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetParameters())
		})
	}
}

func TestOperation_GetResponses_Success(t *testing.T) {
	t.Parallel()

	responses := &swagger.Responses{}
	tests := []struct {
		name     string
		op       *swagger.Operation
		expected *swagger.Responses
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "nil Responses returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns Responses value",
			op:       &swagger.Operation{Responses: responses},
			expected: responses,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetResponses())
		})
	}
}

func TestOperation_GetSchemes_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected []string
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty Schemes returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns Schemes value",
			op:       &swagger.Operation{Schemes: []string{"https", "wss"}},
			expected: []string{"https", "wss"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetSchemes())
		})
	}
}

func TestOperation_GetDeprecated_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *swagger.Operation
		expected bool
	}{
		{
			name:     "nil operation returns false",
			op:       nil,
			expected: false,
		},
		{
			name:     "nil Deprecated returns false",
			op:       &swagger.Operation{},
			expected: false,
		},
		{
			name:     "returns Deprecated true",
			op:       &swagger.Operation{Deprecated: pointer.From(true)},
			expected: true,
		},
		{
			name:     "returns Deprecated false",
			op:       &swagger.Operation{Deprecated: pointer.From(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetDeprecated())
		})
	}
}

func TestOperation_GetSecurity_Success(t *testing.T) {
	t.Parallel()

	security := []*swagger.SecurityRequirement{{}}
	tests := []struct {
		name     string
		op       *swagger.Operation
		expected []*swagger.SecurityRequirement
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty Security returns nil",
			op:       &swagger.Operation{},
			expected: nil,
		},
		{
			name:     "returns Security value",
			op:       &swagger.Operation{Security: security},
			expected: security,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetSecurity())
		})
	}
}

func TestOperation_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	ext := extensions.New()
	tests := []struct {
		name         string
		op           *swagger.Operation
		expectEmpty  bool
		expectedExts *extensions.Extensions
	}{
		{
			name:        "nil operation returns empty extensions",
			op:          nil,
			expectEmpty: true,
		},
		{
			name:        "nil Extensions returns empty extensions",
			op:          &swagger.Operation{},
			expectEmpty: true,
		},
		{
			name:         "returns Extensions value",
			op:           &swagger.Operation{Extensions: ext},
			expectedExts: ext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.op.GetExtensions()
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			} else {
				assert.Equal(t, tt.expectedExts, result)
			}
		})
	}
}
