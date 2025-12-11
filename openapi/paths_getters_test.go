package openapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
)

func TestHTTPMethod_Is_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		method   openapi.HTTPMethod
		input    string
		expected bool
	}{
		{
			name:     "GET matches get lowercase",
			method:   openapi.HTTPMethodGet,
			input:    "get",
			expected: true,
		},
		{
			name:     "GET matches GET uppercase",
			method:   openapi.HTTPMethodGet,
			input:    "GET",
			expected: true,
		},
		{
			name:     "GET matches Get mixed case",
			method:   openapi.HTTPMethodGet,
			input:    "Get",
			expected: true,
		},
		{
			name:     "POST does not match GET",
			method:   openapi.HTTPMethodPost,
			input:    "GET",
			expected: false,
		},
		{
			name:     "POST matches POST",
			method:   openapi.HTTPMethodPost,
			input:    "POST",
			expected: true,
		},
		{
			name:     "QUERY matches QUERY",
			method:   openapi.HTTPMethodQuery,
			input:    "query",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.method.Is(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaths_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		paths      *openapi.Paths
		hasContent bool
	}{
		{
			name:       "nil paths returns empty extensions",
			paths:      nil,
			hasContent: false,
		},
		{
			name: "paths without extensions returns empty extensions",
			paths: &openapi.Paths{
				Map:        sequencedmap.New[string, *openapi.ReferencedPathItem](),
				Extensions: nil,
			},
			hasContent: false,
		},
		{
			name: "paths with extensions returns extensions",
			paths: &openapi.Paths{
				Map:        sequencedmap.New[string, *openapi.ReferencedPathItem](),
				Extensions: extensions.New(),
			},
			hasContent: false, // Empty extensions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.paths.GetExtensions()
			assert.NotNil(t, result, "GetExtensions should never return nil")
		})
	}
}

func TestPathItem_Query_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pathItem *openapi.PathItem
		expected *openapi.Operation
	}{
		{
			name:     "nil path item returns nil",
			pathItem: nil,
			expected: nil,
		},
		{
			name: "path item without query returns nil",
			pathItem: &openapi.PathItem{
				Map: sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.pathItem.Query()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathItem_GetAdditionalOperations_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pathItem *openapi.PathItem
		expected *sequencedmap.Map[string, *openapi.Operation]
	}{
		{
			name:     "nil path item returns nil",
			pathItem: nil,
			expected: nil,
		},
		{
			name: "path item without additional operations returns nil",
			pathItem: &openapi.PathItem{
				Map: sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
			},
			expected: nil,
		},
		{
			name: "path item with additional operations returns them",
			pathItem: &openapi.PathItem{
				Map:                  sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
				AdditionalOperations: sequencedmap.New[string, *openapi.Operation](),
			},
			expected: sequencedmap.New[string, *openapi.Operation](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.pathItem.GetAdditionalOperations()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestPathItem_GetServers_Success(t *testing.T) {
	t.Parallel()

	server := &openapi.Server{URL: "https://api.example.com"}

	tests := []struct {
		name     string
		pathItem *openapi.PathItem
		expected []*openapi.Server
	}{
		{
			name:     "nil path item returns nil",
			pathItem: nil,
			expected: nil,
		},
		{
			name: "path item without servers returns nil",
			pathItem: &openapi.PathItem{
				Map: sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
			},
			expected: nil,
		},
		{
			name: "path item with servers returns them",
			pathItem: &openapi.PathItem{
				Map:     sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
				Servers: []*openapi.Server{server},
			},
			expected: []*openapi.Server{server},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.pathItem.GetServers()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathItem_GetParameters_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pathItem *openapi.PathItem
		expected []*openapi.ReferencedParameter
	}{
		{
			name:     "nil path item returns nil",
			pathItem: nil,
			expected: nil,
		},
		{
			name: "path item without parameters returns nil",
			pathItem: &openapi.PathItem{
				Map: sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.pathItem.GetParameters()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathItem_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pathItem   *openapi.PathItem
		hasContent bool
	}{
		{
			name:       "nil path item returns empty extensions",
			pathItem:   nil,
			hasContent: false,
		},
		{
			name: "path item without extensions returns empty extensions",
			pathItem: &openapi.PathItem{
				Map:        sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
				Extensions: nil,
			},
			hasContent: false,
		},
		{
			name: "path item with extensions returns extensions",
			pathItem: &openapi.PathItem{
				Map:        sequencedmap.New[openapi.HTTPMethod, *openapi.Operation](),
				Extensions: extensions.New(),
			},
			hasContent: false, // Empty extensions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.pathItem.GetExtensions()
			assert.NotNil(t, result, "GetExtensions should never return nil")
		})
	}
}

func TestParameterIn_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		param    openapi.ParameterIn
		expected string
	}{
		{
			name:     "query parameter",
			param:    openapi.ParameterInQuery,
			expected: "query",
		},
		{
			name:     "header parameter",
			param:    openapi.ParameterInHeader,
			expected: "header",
		},
		{
			name:     "path parameter",
			param:    openapi.ParameterInPath,
			expected: "path",
		},
		{
			name:     "cookie parameter",
			param:    openapi.ParameterInCookie,
			expected: "cookie",
		},
		{
			name:     "querystring parameter",
			param:    openapi.ParameterInQueryString,
			expected: "querystring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.param.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParameter_GetContent_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		parameter *openapi.Parameter
		expected  *sequencedmap.Map[string, *openapi.MediaType]
	}{
		{
			name:      "nil parameter returns nil",
			parameter: nil,
			expected:  nil,
		},
		{
			name:      "parameter without content returns nil",
			parameter: &openapi.Parameter{},
			expected:  nil,
		},
		{
			name: "parameter with content returns it",
			parameter: &openapi.Parameter{
				Content: sequencedmap.New[string, *openapi.MediaType](),
			},
			expected: sequencedmap.New[string, *openapi.MediaType](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.parameter.GetContent()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}
