package values

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple test types to isolate the EitherValue sync issue
type TestSchema struct {
	marshaller.Model[TestCore]
	Type string
	Ref  string
}

type TestCore struct {
	marshaller.CoreModel

	Type marshaller.Node[string] `key:"type"`
	Ref  marshaller.Node[string] `key:"$ref"`
}

type TestEitherValue struct {
	EitherValue[TestSchema, TestCore, bool, bool]
}

func TestEitherValue_SyncAfterInPlaceModification(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// First, load from JSON to establish a valid core
	jsonInput := `{"$ref": "#/$defs/Test"}`
	reader := strings.NewReader(jsonInput)

	original := &TestEitherValue{}
	_, err := marshaller.Unmarshal(ctx, reader, original)
	require.NoError(t, err)

	// Verify it loaded correctly
	require.True(t, original.IsLeft())
	require.NotNil(t, original.GetLeft())
	assert.Equal(t, "#/$defs/Test", original.GetLeft().Ref)

	// Marshal to JSON to establish baseline
	var buf1 bytes.Buffer
	err = marshaller.Marshal(ctx, original, &buf1)
	require.NoError(t, err)

	t.Logf("Original JSON: %s", buf1.String())
	assert.Contains(t, buf1.String(), "$ref")
	assert.Contains(t, buf1.String(), "#/$defs/Test")

	// Now modify the SAME EitherValue in place (simulating our inlining scenario)
	boolVal := true
	original.Left = nil
	original.Right = &boolVal

	// Verify the modification worked at the Go level
	require.True(t, original.IsRight())
	require.NotNil(t, original.GetRight())
	assert.Equal(t, true, *original.GetRight())

	// Marshal the same instance after modification
	var buf2 bytes.Buffer
	err = marshaller.Marshal(ctx, original, &buf2)
	require.NoError(t, err)

	// The modified version should show "true", not the original reference
	assert.Contains(t, buf2.String(), "true")
	assert.NotContains(t, buf2.String(), "$ref")
}

func TestEitherValue_BooleanLoad(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Load a boolean value from JSON
	jsonInput := `true`
	reader := strings.NewReader(jsonInput)

	boolValue := &TestEitherValue{}
	_, err := marshaller.Unmarshal(ctx, reader, boolValue)
	require.NoError(t, err)

	// Verify it loaded correctly
	require.True(t, boolValue.IsRight())
	require.NotNil(t, boolValue.GetRight())
	assert.Equal(t, true, *boolValue.GetRight())

	// Marshal back to JSON
	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, boolValue, &buf)
	require.NoError(t, err)

	t.Logf("Boolean JSON: %s", buf.String())
	assert.Contains(t, buf.String(), "true")
}
