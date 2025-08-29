package openapi

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPI_GetOpenAPI_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected string
	}{
		{
			name:     "nil openapi returns empty string",
			openapi:  nil,
			expected: "",
		},
		{
			name: "openapi with version returns version",
			openapi: &OpenAPI{
				OpenAPI: "3.1.0",
			},
			expected: "3.1.0",
		},
		{
			name: "openapi with empty version returns empty string",
			openapi: &OpenAPI{
				OpenAPI: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetOpenAPI()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetInfo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected *Info
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with info returns info pointer",
			openapi: &OpenAPI{
				Info: Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
			expected: &Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetInfo()
			if tt.expected == nil {
				assert.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				assert.Equal(t, tt.expected.Title, actual.Title)
				assert.Equal(t, tt.expected.Version, actual.Version)
			}
		})
	}
}

func TestOpenAPI_GetExternalDocs_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected *oas3.ExternalDocumentation
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with nil external docs returns nil",
			openapi: &OpenAPI{
				ExternalDocs: nil,
			},
			expected: nil,
		},
		{
			name: "openapi with external docs returns external docs",
			openapi: &OpenAPI{
				ExternalDocs: &oas3.ExternalDocumentation{},
			},
			expected: &oas3.ExternalDocumentation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetExternalDocs()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetTags_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected []*Tag
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with nil tags returns nil",
			openapi: &OpenAPI{
				Tags: nil,
			},
			expected: nil,
		},
		{
			name: "openapi with empty tags returns empty slice",
			openapi: &OpenAPI{
				Tags: []*Tag{},
			},
			expected: []*Tag{},
		},
		{
			name: "openapi with tags returns tags",
			openapi: &OpenAPI{
				Tags: []*Tag{
					{Name: "test"},
				},
			},
			expected: []*Tag{
				{Name: "test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetTags()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetServers_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected []*Server
	}{
		{
			name:     "nil openapi returns default server",
			openapi:  nil,
			expected: []*Server{{URL: "/"}},
		},
		{
			name: "openapi with nil servers returns default server",
			openapi: &OpenAPI{
				Servers: nil,
			},
			expected: []*Server{{URL: "/"}},
		},
		{
			name: "openapi with empty servers returns default server",
			openapi: &OpenAPI{
				Servers: []*Server{},
			},
			expected: []*Server{{URL: "/"}},
		},
		{
			name: "openapi with servers returns servers",
			openapi: &OpenAPI{
				Servers: []*Server{
					{URL: "https://api.example.com"},
				},
			},
			expected: []*Server{
				{URL: "https://api.example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetServers()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetSecurity_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected []*SecurityRequirement
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with nil security returns nil",
			openapi: &OpenAPI{
				Security: nil,
			},
			expected: nil,
		},
		{
			name: "openapi with empty security returns empty slice",
			openapi: &OpenAPI{
				Security: []*SecurityRequirement{},
			},
			expected: []*SecurityRequirement{},
		},
		{
			name: "openapi with security returns security",
			openapi: &OpenAPI{
				Security: []*SecurityRequirement{
					NewSecurityRequirement(),
				},
			},
			expected: []*SecurityRequirement{
				NewSecurityRequirement(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetSecurity()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetPaths_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected *Paths
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with nil paths returns nil",
			openapi: &OpenAPI{
				Paths: nil,
			},
			expected: nil,
		},
		{
			name: "openapi with paths returns paths",
			openapi: &OpenAPI{
				Paths: NewPaths(),
			},
			expected: NewPaths(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetPaths()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected *extensions.Extensions
	}{
		{
			name:     "nil openapi returns empty extensions",
			openapi:  nil,
			expected: extensions.New(),
		},
		{
			name: "openapi with nil extensions returns empty extensions",
			openapi: &OpenAPI{
				Extensions: nil,
			},
			expected: extensions.New(),
		},
		{
			name: "openapi with extensions returns extensions",
			openapi: &OpenAPI{
				Extensions: extensions.New(),
			},
			expected: extensions.New(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetExtensions()
			require.NotNil(t, actual)
			// Both should be empty extensions
			assert.Equal(t, 0, actual.Len())
		})
	}
}

func TestOpenAPI_GetWebhooks_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected *sequencedmap.Map[string, *ReferencedPathItem]
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with nil webhooks returns nil",
			openapi: &OpenAPI{
				Webhooks: nil,
			},
			expected: nil,
		},
		{
			name: "openapi with webhooks returns webhooks",
			openapi: &OpenAPI{
				Webhooks: sequencedmap.New[string, *ReferencedPathItem](),
			},
			expected: sequencedmap.New[string, *ReferencedPathItem](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetWebhooks()
			if tt.expected == nil {
				assert.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				assert.Equal(t, 0, actual.Len())
			}
		})
	}
}

func TestOpenAPI_GetComponents_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected *Components
	}{
		{
			name:     "nil openapi returns nil",
			openapi:  nil,
			expected: nil,
		},
		{
			name: "openapi with nil components returns nil",
			openapi: &OpenAPI{
				Components: nil,
			},
			expected: nil,
		},
		{
			name: "openapi with components returns components",
			openapi: &OpenAPI{
				Components: &Components{},
			},
			expected: &Components{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetComponents()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOpenAPI_GetJSONSchemaDialect_Success(t *testing.T) {
	t.Parallel()

	dialect := "https://json-schema.org/draft/2020-12/schema"

	tests := []struct {
		name     string
		openapi  *OpenAPI
		expected string
	}{
		{
			name:     "nil openapi returns empty string",
			openapi:  nil,
			expected: "",
		},
		{
			name: "openapi with nil json schema dialect returns empty string",
			openapi: &OpenAPI{
				JSONSchemaDialect: nil,
			},
			expected: "",
		},
		{
			name: "openapi with json schema dialect returns dialect",
			openapi: &OpenAPI{
				JSONSchemaDialect: &dialect,
			},
			expected: dialect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.openapi.GetJSONSchemaDialect()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
