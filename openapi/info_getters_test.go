package openapi

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
)

// Info getter tests

func TestInfo_GetTitle_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *Info
		expected string
	}{
		{
			name:     "nil info returns empty",
			info:     nil,
			expected: "",
		},
		{
			name:     "returns title",
			info:     &Info{Title: "Test API"},
			expected: "Test API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetTitle())
		})
	}
}

func TestInfo_GetVersion_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *Info
		expected string
	}{
		{
			name:     "nil info returns empty",
			info:     nil,
			expected: "",
		},
		{
			name:     "returns version",
			info:     &Info{Version: "1.0.0"},
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetVersion())
		})
	}
}

func TestInfo_GetSummary_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *Info
		expected string
	}{
		{
			name:     "nil info returns empty",
			info:     nil,
			expected: "",
		},
		{
			name:     "nil summary returns empty",
			info:     &Info{},
			expected: "",
		},
		{
			name:     "returns summary",
			info:     &Info{Summary: pointer.From("A test API")},
			expected: "A test API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetSummary())
		})
	}
}

func TestInfo_GetDescription_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *Info
		expected string
	}{
		{
			name:     "nil info returns empty",
			info:     nil,
			expected: "",
		},
		{
			name:     "nil description returns empty",
			info:     &Info{},
			expected: "",
		},
		{
			name:     "returns description",
			info:     &Info{Description: pointer.From("API description")},
			expected: "API description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetDescription())
		})
	}
}

func TestInfo_GetTermsOfService_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *Info
		expected string
	}{
		{
			name:     "nil info returns empty",
			info:     nil,
			expected: "",
		},
		{
			name:     "nil tos returns empty",
			info:     &Info{},
			expected: "",
		},
		{
			name:     "returns tos",
			info:     &Info{TermsOfService: pointer.From("https://example.com/tos")},
			expected: "https://example.com/tos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetTermsOfService())
		})
	}
}

func TestInfo_GetContact_Success(t *testing.T) {
	t.Parallel()

	contact := &Contact{Name: pointer.From("Test")}
	tests := []struct {
		name     string
		info     *Info
		expected *Contact
	}{
		{
			name:     "nil info returns nil",
			info:     nil,
			expected: nil,
		},
		{
			name:     "nil contact returns nil",
			info:     &Info{},
			expected: nil,
		},
		{
			name:     "returns contact",
			info:     &Info{Contact: contact},
			expected: contact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetContact())
		})
	}
}

func TestInfo_GetLicense_Success(t *testing.T) {
	t.Parallel()

	license := &License{Name: "MIT"}
	tests := []struct {
		name     string
		info     *Info
		expected *License
	}{
		{
			name:     "nil info returns nil",
			info:     nil,
			expected: nil,
		},
		{
			name:     "nil license returns nil",
			info:     &Info{},
			expected: nil,
		},
		{
			name:     "returns license",
			info:     &Info{License: license},
			expected: license,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.info.GetLicense())
		})
	}
}

func TestInfo_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	ext := extensions.New()
	tests := []struct {
		name        string
		info        *Info
		expectEmpty bool
	}{
		{
			name:        "nil info returns empty",
			info:        nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty",
			info:        &Info{},
			expectEmpty: true,
		},
		{
			name:        "returns extensions",
			info:        &Info{Extensions: ext},
			expectEmpty: true, // ext is empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.info.GetExtensions()
			assert.NotNil(t, result)
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			}
		})
	}
}

// Contact getter tests

func TestContact_GetName_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contact  *Contact
		expected string
	}{
		{
			name:     "nil contact returns empty",
			contact:  nil,
			expected: "",
		},
		{
			name:     "nil name returns empty",
			contact:  &Contact{},
			expected: "",
		},
		{
			name:     "returns name",
			contact:  &Contact{Name: pointer.From("John Doe")},
			expected: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.contact.GetName())
		})
	}
}

func TestContact_GetURL_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contact  *Contact
		expected string
	}{
		{
			name:     "nil contact returns empty",
			contact:  nil,
			expected: "",
		},
		{
			name:     "nil url returns empty",
			contact:  &Contact{},
			expected: "",
		},
		{
			name:     "returns url",
			contact:  &Contact{URL: pointer.From("https://example.com")},
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.contact.GetURL())
		})
	}
}

func TestContact_GetEmail_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contact  *Contact
		expected string
	}{
		{
			name:     "nil contact returns empty",
			contact:  nil,
			expected: "",
		},
		{
			name:     "nil email returns empty",
			contact:  &Contact{},
			expected: "",
		},
		{
			name:     "returns email",
			contact:  &Contact{Email: pointer.From("test@example.com")},
			expected: "test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.contact.GetEmail())
		})
	}
}

func TestContact_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contact     *Contact
		expectEmpty bool
	}{
		{
			name:        "nil contact returns empty",
			contact:     nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty",
			contact:     &Contact{},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.contact.GetExtensions()
			assert.NotNil(t, result)
			assert.Equal(t, 0, result.Len())
		})
	}
}

// License getter tests

func TestLicense_GetName_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		license  *License
		expected string
	}{
		{
			name:     "nil license returns empty",
			license:  nil,
			expected: "",
		},
		{
			name:     "returns name",
			license:  &License{Name: "MIT"},
			expected: "MIT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.license.GetName())
		})
	}
}

func TestLicense_GetIdentifier_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		license  *License
		expected string
	}{
		{
			name:     "nil license returns empty",
			license:  nil,
			expected: "",
		},
		{
			name:     "nil identifier returns empty",
			license:  &License{Name: "MIT"},
			expected: "",
		},
		{
			name:     "returns identifier",
			license:  &License{Name: "MIT", Identifier: pointer.From("MIT")},
			expected: "MIT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.license.GetIdentifier())
		})
	}
}

func TestLicense_GetURL_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		license  *License
		expected string
	}{
		{
			name:     "nil license returns empty",
			license:  nil,
			expected: "",
		},
		{
			name:     "nil url returns empty",
			license:  &License{},
			expected: "",
		},
		{
			name:     "returns url",
			license:  &License{URL: pointer.From("https://opensource.org/licenses/MIT")},
			expected: "https://opensource.org/licenses/MIT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.license.GetURL())
		})
	}
}

func TestLicense_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		license     *License
		expectEmpty bool
	}{
		{
			name:        "nil license returns empty",
			license:     nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty",
			license:     &License{},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.license.GetExtensions()
			assert.NotNil(t, result)
			assert.Equal(t, 0, result.Len())
		})
	}
}

// Operation getter tests

func TestOperation_GetOperationID_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *Operation
		expected string
	}{
		{
			name:     "nil operation returns empty",
			op:       nil,
			expected: "",
		},
		{
			name:     "nil operationID returns empty",
			op:       &Operation{},
			expected: "",
		},
		{
			name:     "returns operationID",
			op:       &Operation{OperationID: pointer.From("getUser")},
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

func TestOperation_GetSummary_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *Operation
		expected string
	}{
		{
			name:     "nil operation returns empty",
			op:       nil,
			expected: "",
		},
		{
			name:     "nil summary returns empty",
			op:       &Operation{},
			expected: "",
		},
		{
			name:     "returns summary",
			op:       &Operation{Summary: pointer.From("Get a user")},
			expected: "Get a user",
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
		op       *Operation
		expected string
	}{
		{
			name:     "nil operation returns empty",
			op:       nil,
			expected: "",
		},
		{
			name:     "nil description returns empty",
			op:       &Operation{},
			expected: "",
		},
		{
			name:     "returns description",
			op:       &Operation{Description: pointer.From("Get user by ID")},
			expected: "Get user by ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetDescription())
		})
	}
}

func TestOperation_GetDeprecated_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *Operation
		expected bool
	}{
		{
			name:     "nil operation returns false",
			op:       nil,
			expected: false,
		},
		{
			name:     "nil deprecated returns false",
			op:       &Operation{},
			expected: false,
		},
		{
			name:     "returns deprecated true",
			op:       &Operation{Deprecated: pointer.From(true)},
			expected: true,
		},
		{
			name:     "returns deprecated false",
			op:       &Operation{Deprecated: pointer.From(false)},
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

func TestOperation_GetTags_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *Operation
		expected []string
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty tags returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns tags",
			op:       &Operation{Tags: []string{"users", "admin"}},
			expected: []string{"users", "admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetTags())
		})
	}
}

func TestOperation_GetServers_Success(t *testing.T) {
	t.Parallel()

	servers := []*Server{{URL: "https://api.example.com"}}
	tests := []struct {
		name     string
		op       *Operation
		expected []*Server
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty servers returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns servers",
			op:       &Operation{Servers: servers},
			expected: servers,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetServers())
		})
	}
}

func TestOperation_GetSecurity_Success(t *testing.T) {
	t.Parallel()

	security := []*SecurityRequirement{NewSecurityRequirement()}
	tests := []struct {
		name     string
		op       *Operation
		expected []*SecurityRequirement
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty security returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns security",
			op:       &Operation{Security: security},
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

func TestOperation_GetParameters_Success(t *testing.T) {
	t.Parallel()

	params := []*ReferencedParameter{{}}
	tests := []struct {
		name     string
		op       *Operation
		expected []*ReferencedParameter
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "empty params returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns parameters",
			op:       &Operation{Parameters: params},
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

func TestOperation_GetRequestBody_Success(t *testing.T) {
	t.Parallel()

	body := &ReferencedRequestBody{}
	tests := []struct {
		name     string
		op       *Operation
		expected *ReferencedRequestBody
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "nil request body returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns request body",
			op:       &Operation{RequestBody: body},
			expected: body,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetRequestBody())
		})
	}
}

func TestOperation_GetCallbacks_Success(t *testing.T) {
	t.Parallel()

	callbacks := sequencedmap.New[string, *ReferencedCallback]()
	tests := []struct {
		name     string
		op       *Operation
		expected *sequencedmap.Map[string, *ReferencedCallback]
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "nil callbacks returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns callbacks",
			op:       &Operation{Callbacks: callbacks},
			expected: callbacks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.GetCallbacks())
		})
	}
}

func TestOperation_GetExternalDocs_Success(t *testing.T) {
	t.Parallel()

	docs := &oas3.ExternalDocumentation{}
	tests := []struct {
		name     string
		op       *Operation
		expected *oas3.ExternalDocumentation
	}{
		{
			name:     "nil operation returns nil",
			op:       nil,
			expected: nil,
		},
		{
			name:     "nil docs returns nil",
			op:       &Operation{},
			expected: nil,
		},
		{
			name:     "returns external docs",
			op:       &Operation{ExternalDocs: docs},
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

func TestOperation_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		op          *Operation
		expectEmpty bool
	}{
		{
			name:        "nil operation returns empty",
			op:          nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty",
			op:          &Operation{},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.op.GetExtensions()
			assert.NotNil(t, result)
			assert.Equal(t, 0, result.Len())
		})
	}
}

func TestOperation_IsDeprecated_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       *Operation
		expected bool
	}{
		{
			name:     "returns false when nil deprecated",
			op:       &Operation{},
			expected: false,
		},
		{
			name:     "returns true when deprecated",
			op:       &Operation{Deprecated: pointer.From(true)},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.op.IsDeprecated())
		})
	}
}

// Callback tests

func TestNewCallback_Success(t *testing.T) {
	t.Parallel()

	callback := NewCallback()
	assert.NotNil(t, callback)
	assert.NotNil(t, callback.Map)
	assert.Equal(t, 0, callback.Len())
}

func TestCallback_Len_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		callback *Callback
		expected int
	}{
		{
			name:     "nil callback returns 0",
			callback: nil,
			expected: 0,
		},
		{
			name:     "nil map returns 0",
			callback: &Callback{},
			expected: 0,
		},
		{
			name:     "empty map returns 0",
			callback: NewCallback(),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.callback.Len())
		})
	}
}

func TestCallback_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		callback    *Callback
		expectEmpty bool
	}{
		{
			name:        "nil callback returns empty",
			callback:    nil,
			expectEmpty: true,
		},
		{
			name:        "nil extensions returns empty",
			callback:    &Callback{},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.callback.GetExtensions()
			assert.NotNil(t, result)
			assert.Equal(t, 0, result.Len())
		})
	}
}

// SerializationStyle tests

func TestSerializationStyle_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		style    SerializationStyle
		expected string
	}{
		{
			name:     "simple",
			style:    SerializationStyleSimple,
			expected: "simple",
		},
		{
			name:     "form",
			style:    SerializationStyleForm,
			expected: "form",
		},
		{
			name:     "label",
			style:    SerializationStyleLabel,
			expected: "label",
		},
		{
			name:     "matrix",
			style:    SerializationStyleMatrix,
			expected: "matrix",
		},
		{
			name:     "spaceDelimited",
			style:    SerializationStyleSpaceDelimited,
			expected: "spaceDelimited",
		},
		{
			name:     "pipeDelimited",
			style:    SerializationStylePipeDelimited,
			expected: "pipeDelimited",
		},
		{
			name:     "deepObject",
			style:    SerializationStyleDeepObject,
			expected: "deepObject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.style.String())
		})
	}
}
