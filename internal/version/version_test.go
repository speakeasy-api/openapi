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

func Test_Version_IsOneOf(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  Version
		versions []*Version
		expected bool
	}{
		{
			name:    "version is in list",
			version: Version{Major: 1, Minor: 2, Patch: 3},
			versions: []*Version{
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 2, Patch: 3},
				{Major: 2, Minor: 0, Patch: 0},
			},
			expected: true,
		},
		{
			name:    "version is not in list",
			version: Version{Major: 1, Minor: 2, Patch: 3},
			versions: []*Version{
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 2, Patch: 4},
				{Major: 2, Minor: 0, Patch: 0},
			},
			expected: false,
		},
		{
			name:     "empty list",
			version:  Version{Major: 1, Minor: 2, Patch: 3},
			versions: []*Version{},
			expected: false,
		},
		{
			name:     "nil list",
			version:  Version{Major: 1, Minor: 2, Patch: 3},
			versions: nil,
			expected: false,
		},
		{
			name:    "list with nil values",
			version: Version{Major: 1, Minor: 2, Patch: 3},
			versions: []*Version{
				nil,
				{Major: 1, Minor: 2, Patch: 3},
				nil,
			},
			expected: true,
		},
		{
			name:    "version is first in list",
			version: Version{Major: 1, Minor: 0, Patch: 0},
			versions: []*Version{
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 2, Patch: 3},
				{Major: 2, Minor: 0, Patch: 0},
			},
			expected: true,
		},
		{
			name:    "version is last in list",
			version: Version{Major: 2, Minor: 0, Patch: 0},
			versions: []*Version{
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 2, Patch: 3},
				{Major: 2, Minor: 0, Patch: 0},
			},
			expected: true,
		},
		{
			name:    "similar but different versions",
			version: Version{Major: 1, Minor: 2, Patch: 3},
			versions: []*Version{
				{Major: 1, Minor: 2, Patch: 2},
				{Major: 1, Minor: 2, Patch: 4},
				{Major: 1, Minor: 3, Patch: 3},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.IsOneOf(tt.versions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_Version_String_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  Version
		expected string
	}{
		{
			name:     "standard version",
			version:  Version{Major: 1, Minor: 2, Patch: 3},
			expected: "1.2.3",
		},
		{
			name:     "zero version",
			version:  Version{Major: 0, Minor: 0, Patch: 0},
			expected: "0.0.0",
		},
		{
			name:     "high version numbers",
			version:  Version{Major: 10, Minor: 20, Patch: 30},
			expected: "10.20.30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.version.String())
		})
	}
}

func Test_Version_GreaterThan_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        Version
		b        Version
		expected bool
	}{
		{
			name:     "same version returns false",
			a:        Version{Major: 1, Minor: 2, Patch: 3},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: false,
		},
		{
			name:     "greater major version returns true",
			a:        Version{Major: 2, Minor: 0, Patch: 0},
			b:        Version{Major: 1, Minor: 9, Patch: 9},
			expected: true,
		},
		{
			name:     "lesser major version returns false",
			a:        Version{Major: 1, Minor: 9, Patch: 9},
			b:        Version{Major: 2, Minor: 0, Patch: 0},
			expected: false,
		},
		{
			name:     "greater minor version returns true",
			a:        Version{Major: 1, Minor: 3, Patch: 0},
			b:        Version{Major: 1, Minor: 2, Patch: 9},
			expected: true,
		},
		{
			name:     "lesser minor version returns false",
			a:        Version{Major: 1, Minor: 2, Patch: 9},
			b:        Version{Major: 1, Minor: 3, Patch: 0},
			expected: false,
		},
		{
			name:     "greater patch version returns true",
			a:        Version{Major: 1, Minor: 2, Patch: 4},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: true,
		},
		{
			name:     "lesser patch version returns false",
			a:        Version{Major: 1, Minor: 2, Patch: 3},
			b:        Version{Major: 1, Minor: 2, Patch: 4},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.a.GreaterThan(tt.b))
		})
	}
}

func Test_Version_LessThan_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        Version
		b        Version
		expected bool
	}{
		{
			name:     "same version returns false",
			a:        Version{Major: 1, Minor: 2, Patch: 3},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: false,
		},
		{
			name:     "lesser version returns true",
			a:        Version{Major: 1, Minor: 2, Patch: 3},
			b:        Version{Major: 2, Minor: 0, Patch: 0},
			expected: true,
		},
		{
			name:     "greater version returns false",
			a:        Version{Major: 2, Minor: 0, Patch: 0},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.a.LessThan(tt.b))
		})
	}
}

func Test_MustParse_Success(t *testing.T) {
	t.Parallel()

	v := MustParse("1.2.3")
	assert.Equal(t, 1, v.Major)
	assert.Equal(t, 2, v.Minor)
	assert.Equal(t, 3, v.Patch)
}

func Test_MustParse_Panic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		MustParse("invalid")
	})
}

func Test_IsGreaterOrEqual_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "equal versions returns true",
			a:        "1.2.3",
			b:        "1.2.3",
			expected: true,
		},
		{
			name:     "greater version returns true",
			a:        "2.0.0",
			b:        "1.2.3",
			expected: true,
		},
		{
			name:     "lesser version returns false",
			a:        "1.2.3",
			b:        "2.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := IsGreaterOrEqual(tt.a, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_IsGreaterOrEqual_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "invalid first version",
			a:    "invalid",
			b:    "1.2.3",
		},
		{
			name: "invalid second version",
			a:    "1.2.3",
			b:    "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := IsGreaterOrEqual(tt.a, tt.b)
			require.Error(t, err)
		})
	}
}

func Test_IsLessThan_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "equal versions returns false",
			a:        "1.2.3",
			b:        "1.2.3",
			expected: false,
		},
		{
			name:     "lesser version returns true",
			a:        "1.2.3",
			b:        "2.0.0",
			expected: true,
		},
		{
			name:     "greater version returns false",
			a:        "2.0.0",
			b:        "1.2.3",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := IsLessThan(tt.a, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_IsLessThan_Error(t *testing.T) {
	t.Parallel()

	_, err := IsLessThan("invalid", "1.2.3")
	require.Error(t, err)
}
