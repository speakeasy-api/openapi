package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitParamName_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "single word", input: "token", expected: []string{"token"}},
		{name: "upper case single word", input: "TOKEN", expected: []string{"token"}},
		{name: "camelCase", input: "clientSecret", expected: []string{"client", "secret"}},
		{name: "PascalCase", input: "ClientSecret", expected: []string{"client", "secret"}},
		{name: "snake_case", input: "access_token", expected: []string{"access", "token"}},
		{name: "kebab-case", input: "api-key", expected: []string{"api", "key"}},
		{name: "dot separated", input: "client.secret", expected: []string{"client", "secret"}},
		{name: "mixed separators", input: "X-API-Key", expected: []string{"x", "api", "key"}},
		{name: "leading acronym", input: "APIKey", expected: []string{"api", "key"}},
		{name: "trailing acronym", input: "userID", expected: []string{"user", "id"}},
		{name: "acronym in the middle", input: "myAPIKey", expected: []string{"my", "api", "key"}},
		{name: "digits are separators", input: "oauth2Token", expected: []string{"oauth", "token"}},
		{name: "trailing digits dropped", input: "token2", expected: []string{"token"}},
		{name: "three camelCase words", input: "mySecretPath", expected: []string{"my", "secret", "path"}},
		{name: "empty", input: "", expected: nil},
		{name: "only separators", input: "_-.", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, splitParamName(tt.input), "should split into expected words")
		})
	}
}
