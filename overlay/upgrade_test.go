package overlay_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestUpgrade_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		inputVersion     string
		inputJSONPath    string
		expectedVersion  string
		expectedJSONPath string
		options          []overlay.Option[overlay.UpgradeOptions]
		expectUpgraded   bool
	}{
		{
			name:             "upgrade 1.0.0 to 1.1.0",
			inputVersion:     "1.0.0",
			expectedVersion:  "1.1.0",
			expectedJSONPath: "",
			expectUpgraded:   true,
		},
		{
			name:             "upgrade 1.0.0 with rfc9535 to 1.1.0 removes redundant extension",
			inputVersion:     "1.0.0",
			inputJSONPath:    "rfc9535",
			expectedVersion:  "1.1.0",
			expectedJSONPath: "", // RFC 9535 is now default, so extension cleared
			expectUpgraded:   true,
		},
		{
			name:             "upgrade 1.0.0 with legacy keeps legacy",
			inputVersion:     "1.0.0",
			inputJSONPath:    "legacy",
			expectedVersion:  "1.1.0",
			expectedJSONPath: "legacy", // User explicitly wants legacy, keep it
			expectUpgraded:   true,
		},
		{
			name:             "no upgrade when already at 1.1.0",
			inputVersion:     "1.1.0",
			expectedVersion:  "1.1.0",
			expectedJSONPath: "",
			expectUpgraded:   false,
		},
		{
			name:            "upgrade with explicit target version",
			inputVersion:    "1.0.0",
			expectedVersion: "1.1.0",
			options:         []overlay.Option[overlay.UpgradeOptions]{overlay.WithUpgradeTargetVersion("1.1.0")},
			expectUpgraded:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			o := &overlay.Overlay{
				Version:         tt.inputVersion,
				JSONPathVersion: tt.inputJSONPath,
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
			}

			upgraded, err := overlay.Upgrade(ctx, o, tt.options...)
			require.NoError(t, err, "upgrade should succeed")
			assert.Equal(t, tt.expectUpgraded, upgraded, "upgrade status mismatch")
			assert.Equal(t, tt.expectedVersion, o.Version, "version mismatch")
			assert.Equal(t, tt.expectedJSONPath, o.JSONPathVersion, "JSONPath version mismatch")
		})
	}
}

func TestUpgrade_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		options  []overlay.Option[overlay.UpgradeOptions]
		wantErrs string
	}{
		{
			name:     "cannot downgrade",
			version:  "1.1.0",
			options:  []overlay.Option[overlay.UpgradeOptions]{overlay.WithUpgradeTargetVersion("1.0.0")},
			wantErrs: "cannot downgrade",
		},
		{
			name:     "invalid version format",
			version:  "invalid",
			wantErrs: "invalid current overlay version",
		},
		{
			name:     "invalid target version format",
			version:  "1.0.0",
			options:  []overlay.Option[overlay.UpgradeOptions]{overlay.WithUpgradeTargetVersion("invalid")},
			wantErrs: "invalid target overlay version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			o := &overlay.Overlay{
				Version: tt.version,
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
			}

			_, err := overlay.Upgrade(ctx, o, tt.options...)
			require.Error(t, err, "upgrade should fail")
			assert.Contains(t, err.Error(), tt.wantErrs, "error message mismatch")
		})
	}
}

func TestUpgrade_NilOverlay(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	upgraded, err := overlay.Upgrade(ctx, nil)
	require.NoError(t, err, "upgrade of nil should not error")
	assert.False(t, upgraded, "nil overlay should not be upgraded")
}

func TestUpgrade_PreservesOverlayContent(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Create a fully populated overlay
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title:       "Test Overlay",
			Version:     "2.0.0",
			Description: "This is a test overlay with description",
		},
		Extends: "https://example.com/openapi.yaml",
		Actions: []overlay.Action{
			{
				Target:      "$.info.title",
				Description: "Update the title",
				Update:      yaml.Node{Kind: yaml.ScalarNode, Value: "New Title"},
			},
			{
				Target: "$.info.description",
				Remove: true,
			},
			{
				Target: "$.components.schemas.NewSchema",
				Copy:   "$.components.schemas.ExistingSchema",
			},
		},
	}

	upgraded, err := overlay.Upgrade(ctx, o)
	require.NoError(t, err, "upgrade should succeed")
	assert.True(t, upgraded, "overlay should be upgraded")

	// Verify all content is preserved
	assert.Equal(t, "1.1.0", o.Version, "version should be upgraded")
	assert.Equal(t, "Test Overlay", o.Info.Title, "title should be preserved")
	assert.Equal(t, "2.0.0", o.Info.Version, "info version should be preserved")
	assert.Equal(t, "This is a test overlay with description", o.Info.Description, "description should be preserved")
	assert.Equal(t, "https://example.com/openapi.yaml", o.Extends, "extends should be preserved")
	assert.Len(t, o.Actions, 3, "all actions should be preserved")
	assert.Equal(t, "$.info.title", o.Actions[0].Target, "action target should be preserved")
	assert.Equal(t, "Update the title", o.Actions[0].Description, "action description should be preserved")
	assert.True(t, o.Actions[1].Remove, "remove action should be preserved")
	assert.Equal(t, "$.components.schemas.ExistingSchema", o.Actions[2].Copy, "copy action should be preserved")
}

func TestUpgrade_WithCopyAction(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Copy action already works in 1.0.0, verify it still works after upgrade
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title:   "Copy Action Overlay",
			Version: "1.0.0",
		},
		Actions: []overlay.Action{
			{
				Target:      "$.components.schemas.Bar",
				Copy:        "$.components.schemas.Foo",
				Description: "Copy Foo schema to Bar",
			},
		},
	}

	upgraded, err := overlay.Upgrade(ctx, o)
	require.NoError(t, err, "upgrade with copy action should succeed")
	assert.True(t, upgraded, "overlay should be upgraded")
	assert.Equal(t, "1.1.0", o.Version, "version should be 1.1.0")
	assert.Equal(t, "$.components.schemas.Foo", o.Actions[0].Copy, "copy source should be preserved")
}
