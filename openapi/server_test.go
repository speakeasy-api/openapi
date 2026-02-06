package openapi

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resolveServerVariables_Success(t *testing.T) {
	t.Parallel()

	type args struct {
		serverURL string
		variables *sequencedmap.Map[string, *ServerVariable]
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "single variable substitution",
			args: args{
				serverURL: "https://{host}/api",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					return vars
				}(),
			},
			expected: "https://api.example.com/api",
		},
		{
			name: "multiple variable substitution",
			args: args{
				serverURL: "https://{host}:{port}/{basePath}",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					vars.Set("port", &ServerVariable{Default: "8080"})
					vars.Set("basePath", &ServerVariable{Default: "v1"})
					return vars
				}(),
			},
			expected: "https://api.example.com:8080/v1",
		},
		{
			name: "duplicate variable substitution",
			args: args{
				serverURL: "https://{host}/api/{host}",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					return vars
				}(),
			},
			expected: "https://api.example.com/api/api.example.com",
		},
		{
			name: "no variables in URL",
			args: args{
				serverURL: "https://api.example.com/v1",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "unused.com"})
					return vars
				}(),
			},
			expected: "https://api.example.com/v1",
		},
		{
			name: "URL with encoded curly brackets should not be substituted",
			args: args{
				serverURL: "https://api.example.com/path%7Bnotvar%7D/{host}",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					vars.Set("notvar", &ServerVariable{Default: "shouldnotbeused"})
					return vars
				}(),
			},
			expected: "https://api.example.com/path%7Bnotvar%7D/api.example.com",
		},
		{
			name: "URL with mixed encoded and unencoded brackets",
			args: args{
				serverURL: "https://{host}/path%7Bstatic%7D/api/{version}",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					vars.Set("version", &ServerVariable{Default: "v1"})
					vars.Set("static", &ServerVariable{Default: "shouldnotbeused"})
					return vars
				}(),
			},
			expected: "https://api.example.com/path%7Bstatic%7D/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, errs := resolveServerVariables(tt.args.serverURL, tt.args.variables)
			require.Empty(t, errs, "expected no errors")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_resolveServerVariables_Error(t *testing.T) {
	t.Parallel()

	type args struct {
		serverURL string
		variables *sequencedmap.Map[string, *ServerVariable]
	}
	tests := []struct {
		name         string
		args         args
		expectedErrs []string
	}{
		{
			name: "no variables defined",
			args: args{
				serverURL: "https://{host}/api",
				variables: sequencedmap.New[string, *ServerVariable](),
			},
			expectedErrs: []string{"serverURL contains variables but no variables are defined"},
		},
		{
			name: "undefined variable",
			args: args{
				serverURL: "https://{host}/api",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("port", &ServerVariable{Default: "8080"})
					return vars
				}(),
			},
			expectedErrs: []string{"server variable `host` is not defined"},
		},
		{
			name: "variable with empty default",
			args: args{
				serverURL: "https://{host}/api",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: ""})
					return vars
				}(),
			},
			expectedErrs: []string{"server variable `host` has no default value"},
		},
		{
			name: "multiple variables with one undefined",
			args: args{
				serverURL: "https://{host}:{port}/api",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					return vars
				}(),
			},
			expectedErrs: []string{"server variable `port` is not defined"},
		},
		{
			name: "multiple variables with one having empty default",
			args: args{
				serverURL: "https://{host}:{port}/api",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					vars.Set("port", &ServerVariable{Default: ""})
					return vars
				}(),
			},
			expectedErrs: []string{"server variable `port` has no default value"},
		},
		{
			name: "malformed nested brackets creates invalid variable name",
			args: args{
				serverURL: "https://api.example.com/{incomplete/path/{host}/end}",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					return vars
				}(),
			},
			expectedErrs: []string{"server variable `incomplete/path/{host` is not defined"},
		},
		{
			name: "double curly braces produces multiple errors",
			args: args{
				serverURL: "https://{{host}}{{port}}/api",
				variables: func() *sequencedmap.Map[string, *ServerVariable] {
					vars := sequencedmap.New[string, *ServerVariable]()
					vars.Set("host", &ServerVariable{Default: "api.example.com"})
					vars.Set("port", &ServerVariable{Default: "8080"})
					return vars
				}(),
			},
			expectedErrs: []string{
				"server variable `{host}` is not defined. Use single curly braces for variable substitution",
				"server variable `{port}` is not defined. Use single curly braces for variable substitution",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, errs := resolveServerVariables(tt.args.serverURL, tt.args.variables)
			require.NotEmpty(t, errs, "expected errors")
			assert.Empty(t, result)
			for _, expectedErr := range tt.expectedErrs {
				assertErrorContains(t, errs, expectedErr)
			}
		})
	}
}

func assertErrorContains(t *testing.T, errs []error, expected string) {
	t.Helper()

	for _, err := range errs {
		if err == nil {
			continue
		}
		if strings.Contains(err.Error(), expected) {
			return
		}
	}

	assert.Fail(t, "expected error not found", "expected %q in errors: %v", expected, errs)
}
