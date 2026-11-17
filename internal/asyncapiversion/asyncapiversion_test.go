package asyncapiversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Version
	}{
		{name: "3.0.0", input: "3.0.0", expected: Version{Major: 3, Minor: 0, Patch: 0}},
		{name: "3.1.0", input: "3.1.0", expected: Version{Major: 3, Minor: 1, Patch: 0}},
		{name: "future patch", input: "3.1.7", expected: Version{Major: 3, Minor: 1, Patch: 7}},
		{name: "alphanumeric suffix", input: "3.0.0-rc1", expected: Version{Major: 3, Minor: 0, Patch: 0, Suffix: "rc1"}},
		{name: "2.x parses", input: "2.6.0", expected: Version{Major: 2, Minor: 6, Patch: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := Parse(tt.input)
			require.NoError(t, err, "parse should succeed")
			assert.Equal(t, tt.expected, *v, "should parse expected version")
			assert.Equal(t, tt.input, v.String(), "string should round trip")
		})
	}
}

func TestParse_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "missing patch", input: "3.1"},
		{name: "empty suffix", input: "3.1.0-"},
		{name: "non alphanumeric suffix", input: "3.1.0-rc.1"},
		{name: "leading zero", input: "03.1.0"},
		{name: "letters", input: "three.one.zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(tt.input)
			require.Error(t, err, "parse should fail")
		})
	}
}

func TestIsSupported_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "3.0.0 supported", input: "3.0.0", expected: true},
		{name: "3.1.0 supported", input: "3.1.0", expected: true},
		{name: "future 3.1 patch supported", input: "3.1.9", expected: true},
		{name: "suffix supported", input: "3.0.0-rc2", expected: true},
		{name: "3.2 unsupported", input: "3.2.0", expected: false},
		{name: "2.6 unsupported", input: "2.6.0", expected: false},
		{name: "4.0 unsupported", input: "4.0.0", expected: false},
		{name: "invalid unsupported", input: "3.1", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, IsSupported(tt.input), "supported should match")
		})
	}
}

func TestSupportedVersionPatterns_Success(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"3.0.x", "3.1.x"}, SupportedVersionPatterns(), "should list supported minor lines")
}

func TestParseSchemaFormat_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		expected        SchemaFormat
		jsonSchemaBased bool
		draft07         bool
		standard        bool
	}{
		{
			name:            "native asyncapi",
			input:           "application/vnd.aai.asyncapi;version=3.1.0",
			expected:        SchemaFormat{Raw: "application/vnd.aai.asyncapi;version=3.1.0", Kind: SchemaFormatKindAsyncAPI, MediaType: "application/vnd.aai.asyncapi", Version: "3.1.0"},
			jsonSchemaBased: true,
			standard:        true,
		},
		{
			name:            "native asyncapi yaml 3.0",
			input:           "application/vnd.aai.asyncapi+yaml;version=3.0.0",
			expected:        SchemaFormat{Raw: "application/vnd.aai.asyncapi+yaml;version=3.0.0", Kind: SchemaFormatKindAsyncAPI, MediaType: "application/vnd.aai.asyncapi+yaml", Version: "3.0.0"},
			jsonSchemaBased: true,
			standard:        true,
		},
		{
			name:     "native asyncapi 2.x is not modelled",
			input:    "application/vnd.aai.asyncapi+json;version=2.6.0",
			expected: SchemaFormat{Raw: "application/vnd.aai.asyncapi+json;version=2.6.0", Kind: SchemaFormatKindAsyncAPI, MediaType: "application/vnd.aai.asyncapi+json", Version: "2.6.0"},
			standard: true,
		},
		{
			name:            "draft 07",
			input:           "application/schema+json;version=draft-07",
			expected:        SchemaFormat{Raw: "application/schema+json;version=draft-07", Kind: SchemaFormatKindJSONSchema, MediaType: "application/schema+json", Version: "draft-07"},
			jsonSchemaBased: true,
			draft07:         true,
			standard:        true,
		},
		{
			name:     "other json schema draft is not modelled",
			input:    "application/schema+json;version=draft-04",
			expected: SchemaFormat{Raw: "application/schema+json;version=draft-04", Kind: SchemaFormatKindJSONSchema, MediaType: "application/schema+json", Version: "draft-04"},
			standard: true,
		},
		{
			name:     "avro",
			input:    "application/vnd.apache.avro+yaml;version=1.9.0",
			expected: SchemaFormat{Raw: "application/vnd.apache.avro+yaml;version=1.9.0", Kind: SchemaFormatKindAvro, MediaType: "application/vnd.apache.avro+yaml", Version: "1.9.0"},
			standard: true,
		},
		{
			name:     "protobuf",
			input:    "application/vnd.google.protobuf;version=3",
			expected: SchemaFormat{Raw: "application/vnd.google.protobuf;version=3", Kind: SchemaFormatKindProtobuf, MediaType: "application/vnd.google.protobuf", Version: "3"},
			standard: true,
		},
		{
			name:     "media type is case insensitive",
			input:    "Application/Vnd.Apache.Avro;version=1.9.0",
			expected: SchemaFormat{Raw: "Application/Vnd.Apache.Avro;version=1.9.0", Kind: SchemaFormatKindAvro, MediaType: "application/vnd.apache.avro", Version: "1.9.0"},
			standard: true,
		},
		{
			name:     "custom",
			input:    "application/x-custom",
			expected: SchemaFormat{Raw: "application/x-custom", Kind: SchemaFormatKindCustom, MediaType: "application/x-custom"},
		},
		{
			name:     "invalid",
			input:    "not a media type;;",
			expected: SchemaFormat{Raw: "not a media type;;", Kind: SchemaFormatKindInvalid},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := ParseSchemaFormat(tt.input)
			assert.Equal(t, tt.expected, f, "should parse expected format")
			assert.Equal(t, tt.jsonSchemaBased, f.IsJSONSchemaBased(), "json schema based should match")
			assert.Equal(t, tt.draft07, f.IsDraft07(), "draft 07 should match")
			assert.Equal(t, tt.standard, f.IsStandardKind(), "standard kind should match")
		})
	}
}

func TestDefaultSchemaFormat_Success(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "application/vnd.aai.asyncapi+json;version=3.0.0", DefaultSchemaFormat("3.0.0"), "should use document version")
	assert.Equal(t, "application/vnd.aai.asyncapi+json;version="+Latest, DefaultSchemaFormat(""), "should default to latest")
}
