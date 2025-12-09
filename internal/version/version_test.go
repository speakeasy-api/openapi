package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseVersion_Success(t *testing.T) {
	t.Parallel()
	type args struct {
		version string
	}
	tests := []struct {
		name          string
		args          args
		expectedMajor int
		expectedMinor int
		expectedPatch int
	}{
		{
			name:          "standard version",
			args:          args{version: "1.2.3"},
			expectedMajor: 1,
			expectedMinor: 2,
			expectedPatch: 3,
		},
		{
			name:          "zero version",
			args:          args{version: "0.0.0"},
			expectedMajor: 0,
			expectedMinor: 0,
			expectedPatch: 0,
		},
		{
			name:          "high version numbers",
			args:          args{version: "10.20.30"},
			expectedMajor: 10,
			expectedMinor: 20,
			expectedPatch: 30,
		},
		{
			name:          "mixed single and multi digit",
			args:          args{version: "2.0.15"},
			expectedMajor: 2,
			expectedMinor: 0,
			expectedPatch: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			version, err := Parse(tt.args.version)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMajor, version.Major)
			assert.Equal(t, tt.expectedMinor, version.Minor)
			assert.Equal(t, tt.expectedPatch, version.Patch)
		})
	}
}

func Test_ParseVersion_Error(t *testing.T) {
	t.Parallel()
	type args struct {
		version string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "empty string",
			args: args{version: ""},
		},
		{
			name: "single number",
			args: args{version: "1"},
		},
		{
			name: "two numbers",
			args: args{version: "1.2"},
		},
		{
			name: "four numbers",
			args: args{version: "1.2.3.4"},
		},
		{
			name: "invalid major version",
			args: args{version: "a.2.3"},
		},
		{
			name: "invalid minor version",
			args: args{version: "1.b.3"},
		},
		{
			name: "invalid patch version",
			args: args{version: "1.2.c"},
		},
		{
			name: "negative major version",
			args: args{version: "-1.2.3"},
		},
		{
			name: "negative minor version",
			args: args{version: "1.-2.3"},
		},
		{
			name: "negative patch version",
			args: args{version: "1.2.-3"},
		},
		{
			name: "extra dots",
			args: args{version: "1..2.3"},
		},
		{
			name: "trailing dot",
			args: args{version: "1.2.3."},
		},
		{
			name: "leading dot",
			args: args{version: ".1.2.3"},
		},
		{
			name: "spaces in version",
			args: args{version: "1 . 2 . 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			version, err := Parse(tt.args.version)
			require.Error(t, err)
			assert.Nil(t, version)
		})
	}
}
