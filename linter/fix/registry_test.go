package fix_test

import (
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixRegistry_GetFix_Success(t *testing.T) {
	t.Parallel()

	registry := fix.NewFixRegistry()
	expectedFix := &mockFix{description: "test fix"}

	registry.Register("validation-empty-value", func(_ *validation.Error) validation.Fix {
		return expectedFix
	})

	vErr := &validation.Error{
		UnderlyingError: errors.New("empty value"),
		Rule:            "validation-empty-value",
	}

	result := registry.GetFix(vErr)
	require.NotNil(t, result, "should return a fix")
	assert.Equal(t, "test fix", result.Description(), "should return the registered fix")
}

func TestFixRegistry_GetFix_NoProvider(t *testing.T) {
	t.Parallel()

	registry := fix.NewFixRegistry()

	vErr := &validation.Error{
		UnderlyingError: errors.New("some error"),
		Rule:            "unknown-rule",
	}

	result := registry.GetFix(vErr)
	assert.Nil(t, result, "should return nil for unregistered rules")
}

func TestFixRegistry_GetFix_ProviderReturnsNil(t *testing.T) {
	t.Parallel()

	registry := fix.NewFixRegistry()
	registry.Register("test-rule", func(_ *validation.Error) validation.Fix {
		return nil
	})

	vErr := &validation.Error{
		UnderlyingError: errors.New("test"),
		Rule:            "test-rule",
	}

	result := registry.GetFix(vErr)
	assert.Nil(t, result, "should return nil when provider returns nil")
}

func TestFixRegistry_GetFix_MultipleProviders(t *testing.T) {
	t.Parallel()

	registry := fix.NewFixRegistry()

	// First provider returns nil
	registry.Register("test-rule", func(_ *validation.Error) validation.Fix {
		return nil
	})

	// Second provider returns a fix
	expectedFix := &mockFix{description: "second provider fix"}
	registry.Register("test-rule", func(_ *validation.Error) validation.Fix {
		return expectedFix
	})

	vErr := &validation.Error{
		UnderlyingError: errors.New("test"),
		Rule:            "test-rule",
	}

	result := registry.GetFix(vErr)
	require.NotNil(t, result, "should return fix from second provider")
	assert.Equal(t, "second provider fix", result.Description(), "first non-nil provider wins")
}

func TestFixRegistry_GetFix_FirstProviderWins(t *testing.T) {
	t.Parallel()

	registry := fix.NewFixRegistry()

	firstFix := &mockFix{description: "first"}
	registry.Register("test-rule", func(_ *validation.Error) validation.Fix {
		return firstFix
	})

	registry.Register("test-rule", func(_ *validation.Error) validation.Fix {
		return &mockFix{description: "second"}
	})

	vErr := &validation.Error{
		UnderlyingError: errors.New("test"),
		Rule:            "test-rule",
	}

	result := registry.GetFix(vErr)
	require.NotNil(t, result, "should return a fix")
	assert.Equal(t, "first", result.Description(), "first non-nil provider should win")
}
