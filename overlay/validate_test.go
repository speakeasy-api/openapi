package overlay_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOverlay_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		overlay        *overlay.Overlay
		expectedErrors []string
	}{
		{
			name: "valid overlay",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Extends: "https://example.com/openapi.yaml",
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: nil,
		},
		{
			name: "invalid overlay version format",
			overlay: &overlay.Overlay{
				Version: "invalid",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay version is invalid"},
		},
		{
			name: "unsupported overlay version",
			overlay: &overlay.Overlay{
				Version: "2.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay version must be one of: `1.0.0, 1.1.0`"},
		},
		{
			name: "empty overlay version",
			overlay: &overlay.Overlay{
				Version: "",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay version is invalid"},
		},
		{
			name: "missing info version",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay info version must be defined"},
		},
		{
			name: "missing info version and invalid overlay version",
			overlay: &overlay.Overlay{
				Version: "invalid",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{
				"overlay info version must be defined",
				"overlay version is invalid",
			},
		},
		{
			name: "valid overlay without extends",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: nil,
		},
		{
			name: "valid overlay with remove action",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.description",
						Remove: true,
					},
				},
			},
			expectedErrors: nil,
		},
		{
			name: "missing title",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay info title must be defined"},
		},
		{
			name: "invalid extends URL",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Extends: "://invalid-url",
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay extends must be a valid URL"},
		},
		{
			name: "no actions",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{},
			},
			expectedErrors: []string{"overlay must define at least one action"},
		},
		{
			name: "action without target",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay action at index 0 target must be defined"},
		},
		{
			name: "action with both remove and update",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Remove: true,
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
				},
			},
			expectedErrors: []string{"overlay action at index 0 should not both set remove and define update"},
		},
		{
			name: "multiple actions with errors at different indices",
			overlay: &overlay.Overlay{
				Version: "1.0.0",
				Info: overlay.Info{
					Title:   "Test Overlay",
					Version: "1.0.0",
				},
				Actions: []overlay.Action{
					{
						Target: "$.info.title",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
					{
						Target: "",
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
					},
					{
						Target: "$.info.description",
						Remove: true,
						Update: yaml.Node{Kind: yaml.ScalarNode, Value: "Description"},
					},
				},
			},
			expectedErrors: []string{
				"overlay action at index 1 target must be defined",
				"overlay action at index 2 should not both set remove and define update",
			},
		},
		{
			name: "all validation errors combined",
			overlay: &overlay.Overlay{
				Version: "invalid",
				Info: overlay.Info{
					Title:   "",
					Version: "",
				},
				Extends: "://invalid-url",
				Actions: []overlay.Action{},
			},
			expectedErrors: []string{
				"overlay info version must be defined",
				"overlay version is invalid",
				"overlay info title must be defined",
				"overlay extends must be a valid URL",
				"overlay must define at least one action",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.overlay.Validate()
			if tt.expectedErrors == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, expectedErr := range tt.expectedErrors {
					assert.Contains(t, err.Error(), expectedErr)
				}
			}
		})
	}
}
