package openapi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptimize_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/optimize/optimize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Optimize the document
	err = openapi.Optimize(ctx, inputDoc, nil)
	require.NoError(t, err)

	// Marshal the optimized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/optimize/optimize_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Optimized document should match expected output")
}

func TestOptimize_EmptyDocument_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test with nil document
	err := openapi.Optimize(ctx, nil, nil)
	require.NoError(t, err)

	// Test with minimal document (no components)
	doc := &openapi.OpenAPI{
		OpenAPI: openapi.Version,
		Info: openapi.Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	err = openapi.Optimize(ctx, doc, nil)
	require.NoError(t, err)
}

func TestOptimize_WithCallback_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Load the input document
	inputFile, err := os.Open("testdata/optimize/optimize_input.yaml")
	require.NoError(t, err)
	defer inputFile.Close()

	inputDoc, validationErrs, err := openapi.Unmarshal(ctx, inputFile)
	require.NoError(t, err)
	require.Empty(t, validationErrs, "Input document should be valid")

	// Define a mapping of hashes to meaningful names
	hashToName := map[string]string{
		"da0c4bbf": "OrderItem",         // Order item schema with productId, quantity, price
		"22b284ff": "OrderStatus",       // Order status enum
		"8c71cc44": "EmailNotification", // Email notification schema
		"faf3f3c6": "SmsNotification",   // SMS notification schema
		"09e719fb": "PushNotification",  // Push notification schema
		"2e1336d2": "Notification",      // anyOf notification schema
		"4f8aeb0f": "PhysicalProduct",   // Physical product schema
		"7a715a64": "DigitalProduct",    // Digital product schema
		"13c3942f": "Product",           // oneOf product schema
		"a89b7799": "Dimensions",        // Dimensions schema
		"8054b7a2": "BaseEntity",        // Base entity with id, createdAt, updatedAt
		"93d337a4": "OrderDetails",      // Order details with status and items
		"76a1fb01": "Order",             // allOf order schema
		"5eb90aa8": "Profile",           // Profile schema with bio and avatar
	}

	// Create a callback that uses our mapping
	nameCallback := func(suggestedName, hash string, locations []string, schema *oas3.JSONSchema[oas3.Referenceable]) string {
		// Extract the hash from the suggested name (format: "Schema_12345678")
		if len(hash) >= 8 {
			shortHash := hash[:8]
			if customName, exists := hashToName[shortHash]; exists {
				return customName
			}
		}
		// Fallback to suggested name if not in our mapping
		return suggestedName
	}

	// Optimize the document with the callback
	err = openapi.Optimize(ctx, inputDoc, nameCallback)
	require.NoError(t, err)

	// Marshal the optimized document to YAML
	var buf bytes.Buffer
	err = openapi.Marshal(ctx, inputDoc, &buf)
	require.NoError(t, err)
	actualYAML := buf.Bytes()

	// Load the expected output
	expectedBytes, err := os.ReadFile("testdata/optimize/optimize_callback_expected.yaml")
	require.NoError(t, err)

	// Compare the actual output with expected output
	assert.Equal(t, string(expectedBytes), string(actualYAML), "Optimized document should match expected output")
}
