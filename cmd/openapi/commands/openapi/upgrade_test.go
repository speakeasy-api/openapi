package openapi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeOpenAPI_ValidVersionTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		inputDoc        string
		opts            []openapi.Option[openapi.UpgradeOptions]
		expectedVersion string
		expectUpgraded  bool
	}{
		{
			name: "upgrades 3.0.3 to latest by default",
			inputDoc: `openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths: {}
`,
			opts:            []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSameMinorVersion()},
			expectedVersion: openapi.Version,
			expectUpgraded:  true,
		},
		{
			name: "upgrades 3.1.0 to latest by default",
			inputDoc: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
`,
			opts:            []openapi.Option[openapi.UpgradeOptions]{openapi.WithUpgradeSameMinorVersion()},
			expectedVersion: openapi.Version,
			expectUpgraded:  true,
		},
		{
			name: "upgrades 3.0.3 to target version 3.1.0",
			inputDoc: `openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths: {}
`,
			opts: []openapi.Option[openapi.UpgradeOptions]{
				openapi.WithUpgradeTargetVersion("3.1.0"),
				openapi.WithUpgradeSameMinorVersion(),
			},
			expectedVersion: "3.1.0",
			expectUpgraded:  true,
		},
		{
			name: "no upgrade needed when already at target version",
			inputDoc: `openapi: "3.1.0"
info:
  title: Test
  version: "1.0"
paths: {}
`,
			opts: []openapi.Option[openapi.UpgradeOptions]{
				openapi.WithUpgradeTargetVersion("3.1.0"),
				openapi.WithUpgradeSameMinorVersion(),
			},
			expectedVersion: "3.1.0",
			expectUpgraded:  false,
		},
		{
			name: "minor-only skips same-minor upgrade",
			inputDoc: `openapi: "3.2.0"
info:
  title: Test
  version: "1.0"
paths: {}
`,
			opts:            nil, // no WithUpgradeSameMinorVersion
			expectedVersion: "3.2.0",
			expectUpgraded:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			processor := &OpenAPIProcessor{
				InputFile:     "-",
				ReadFromStdin: true,
				WriteToStdout: true,
				Stdin:         strings.NewReader(tt.inputDoc),
				Stdout:        &stdout,
				Stderr:        &stderr,
			}

			err := upgradeOpenAPI(t.Context(), processor, tt.opts...)
			require.NoError(t, err, "upgradeOpenAPI should succeed")

			assert.Contains(t, stdout.String(), "openapi: \""+tt.expectedVersion+"\"",
				"output should contain the expected version")

			if tt.expectUpgraded {
				assert.Contains(t, stderr.String(), "Successfully upgraded",
					"stderr should report successful upgrade")
			} else {
				assert.Contains(t, stderr.String(), "No upgrade needed",
					"stderr should report no upgrade needed")
			}
		})
	}
}

func TestUpgradeOpenAPI_InvalidVersionTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		inputDoc    string
		opts        []openapi.Option[openapi.UpgradeOptions]
		expectedErr string
	}{
		{
			name: "cannot downgrade version",
			inputDoc: `openapi: "3.2.0"
info:
  title: Test
  version: "1.0"
paths: {}
`,
			opts: []openapi.Option[openapi.UpgradeOptions]{
				openapi.WithUpgradeTargetVersion("3.1.0"),
				openapi.WithUpgradeSameMinorVersion(),
			},
			expectedErr: "cannot downgrade",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			processor := &OpenAPIProcessor{
				InputFile:     "-",
				ReadFromStdin: true,
				WriteToStdout: true,
				Stdin:         strings.NewReader(tt.inputDoc),
				Stdout:        &stdout,
				Stderr:        &stderr,
			}

			err := upgradeOpenAPI(t.Context(), processor, tt.opts...)
			require.Error(t, err, "upgradeOpenAPI should return an error")
			assert.Contains(t, err.Error(), tt.expectedErr, "error should contain expected message")
		})
	}
}
