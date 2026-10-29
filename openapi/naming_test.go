package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileStem_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "strips directory and extension", path: "some/path/User.yaml", expected: "User"},
		{name: "keeps inner dots", path: "schemas/user.v1.yaml", expected: "user.v1"},
		{name: "no extension", path: "schemas/User", expected: "User"},
		{name: "empty path", path: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, fileStem(tt.path), "should return the file name without extension")
		})
	}
}

func TestSanitizeFileNameForComponent_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "safe name unchanged", input: "User_1", expected: "User_1"},
		{name: "hyphen and dot replaced", input: "user-schema.v1", expected: "user_schema_v1"},
		{name: "each unsafe character replaced", input: "a  b", expected: "a__b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, sanitizeFileNameForComponent(tt.input), "should replace unsafe characters")
		})
	}
}

func TestSanitizeFileNameForFragment_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "safe name unchanged", input: "User_1", expected: "User_1"},
		{name: "hyphen kept", input: "user-api", expected: "user-api"},
		{name: "dot and space replaced", input: "user api.v1", expected: "user_api_v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, sanitizeFileNameForFragment(tt.input), "should replace unsafe characters but keep hyphens")
		})
	}
}

func TestFilePathToComponentPrefix_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cleanPath string
		expected  string
	}{
		{name: "keeps extension as suffix", cleanPath: "some/path/external.yaml", expected: "some_path_external_yaml"},
		{name: "no extension", cleanPath: "schemas/User", expected: "schemas_User"},
		{name: "hyphens and inner dots replaced", cleanPath: "my-dir/user.v1.json", expected: "my_dir_user_v1_json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, filePathToComponentPrefix(tt.cleanPath), "should build a safe component prefix")
		})
	}
}
