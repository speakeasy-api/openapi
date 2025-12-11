package swagger_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/swagger"
	"github.com/stretchr/testify/assert"
)

func TestInfo_GetTermsOfService_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *swagger.Info
		expected string
	}{
		{
			name:     "nil info returns empty string",
			info:     nil,
			expected: "",
		},
		{
			name:     "nil TermsOfService returns empty string",
			info:     &swagger.Info{},
			expected: "",
		},
		{
			name:     "returns TermsOfService value",
			info:     &swagger.Info{TermsOfService: pointer.From("https://example.com/terms")},
			expected: "https://example.com/terms",
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

	contact := &swagger.Contact{Name: pointer.From("Test Contact")}
	tests := []struct {
		name     string
		info     *swagger.Info
		expected *swagger.Contact
	}{
		{
			name:     "nil info returns nil",
			info:     nil,
			expected: nil,
		},
		{
			name:     "nil Contact returns nil",
			info:     &swagger.Info{},
			expected: nil,
		},
		{
			name:     "returns Contact value",
			info:     &swagger.Info{Contact: contact},
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

	license := &swagger.License{Name: "MIT"}
	tests := []struct {
		name     string
		info     *swagger.Info
		expected *swagger.License
	}{
		{
			name:     "nil info returns nil",
			info:     nil,
			expected: nil,
		},
		{
			name:     "nil License returns nil",
			info:     &swagger.Info{},
			expected: nil,
		},
		{
			name:     "returns License value",
			info:     &swagger.Info{License: license},
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
		name         string
		info         *swagger.Info
		expectEmpty  bool
		expectedExts *extensions.Extensions
	}{
		{
			name:        "nil info returns empty extensions",
			info:        nil,
			expectEmpty: true,
		},
		{
			name:        "nil Extensions returns empty extensions",
			info:        &swagger.Info{},
			expectEmpty: true,
		},
		{
			name:         "returns Extensions value",
			info:         &swagger.Info{Extensions: ext},
			expectedExts: ext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.info.GetExtensions()
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			} else {
				assert.Equal(t, tt.expectedExts, result)
			}
		})
	}
}

func TestContact_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	ext := extensions.New()
	tests := []struct {
		name         string
		contact      *swagger.Contact
		expectEmpty  bool
		expectedExts *extensions.Extensions
	}{
		{
			name:        "nil contact returns empty extensions",
			contact:     nil,
			expectEmpty: true,
		},
		{
			name:        "nil Extensions returns empty extensions",
			contact:     &swagger.Contact{},
			expectEmpty: true,
		},
		{
			name:         "returns Extensions value",
			contact:      &swagger.Contact{Extensions: ext},
			expectedExts: ext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.contact.GetExtensions()
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			} else {
				assert.Equal(t, tt.expectedExts, result)
			}
		})
	}
}

func TestLicense_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	ext := extensions.New()
	tests := []struct {
		name         string
		license      *swagger.License
		expectEmpty  bool
		expectedExts *extensions.Extensions
	}{
		{
			name:        "nil license returns empty extensions",
			license:     nil,
			expectEmpty: true,
		},
		{
			name:        "nil Extensions returns empty extensions",
			license:     &swagger.License{},
			expectEmpty: true,
		},
		{
			name:         "returns Extensions value",
			license:      &swagger.License{Extensions: ext},
			expectedExts: ext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.license.GetExtensions()
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			} else {
				assert.Equal(t, tt.expectedExts, result)
			}
		})
	}
}

func TestExternalDocumentation_GetExtensions_Success(t *testing.T) {
	t.Parallel()

	ext := extensions.New()
	tests := []struct {
		name         string
		extDoc       *swagger.ExternalDocumentation
		expectEmpty  bool
		expectedExts *extensions.Extensions
	}{
		{
			name:        "nil extDoc returns empty extensions",
			extDoc:      nil,
			expectEmpty: true,
		},
		{
			name:        "nil Extensions returns empty extensions",
			extDoc:      &swagger.ExternalDocumentation{},
			expectEmpty: true,
		},
		{
			name:         "returns Extensions value",
			extDoc:       &swagger.ExternalDocumentation{Extensions: ext},
			expectedExts: ext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.extDoc.GetExtensions()
			if tt.expectEmpty {
				assert.Equal(t, 0, result.Len())
			} else {
				assert.Equal(t, tt.expectedExts, result)
			}
		})
	}
}
