package overlay_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewTargetSelector_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		method   string
		expected string
	}{
		{
			name:     "simple path and method",
			path:     "/users",
			method:   "get",
			expected: `$["paths"]["/users"]["get"]`,
		},
		{
			name:     "path with parameter",
			path:     "/users/{id}",
			method:   "patch",
			expected: `$["paths"]["/users/{id}"]["patch"]`,
		},
		{
			name:     "root path",
			path:     "/",
			method:   "post",
			expected: `$["paths"]["/"]["post"]`,
		},
		{
			name:     "nested path",
			path:     "/users/{userId}/orders/{orderId}",
			method:   "delete",
			expected: `$["paths"]["/users/{userId}/orders/{orderId}"]["delete"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := overlay.NewTargetSelector(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewUpdateAction_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		path           string
		method         string
		update         yaml.Node
		expectedTarget string
	}{
		{
			name:           "simple update action",
			path:           "/users",
			method:         "get",
			update:         yaml.Node{Kind: yaml.ScalarNode, Value: "test"},
			expectedTarget: `$["paths"]["/users"]["get"]`,
		},
		{
			name:           "update action with path parameter",
			path:           "/users/{id}",
			method:         "put",
			update:         yaml.Node{Kind: yaml.MappingNode},
			expectedTarget: `$["paths"]["/users/{id}"]["put"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := overlay.NewUpdateAction(tt.path, tt.method, tt.update)
			assert.Equal(t, tt.expectedTarget, result.Target)
			assert.Equal(t, tt.update.Kind, result.Update.Kind)
		})
	}
}
