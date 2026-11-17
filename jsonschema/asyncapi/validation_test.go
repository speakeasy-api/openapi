package asyncapi_test

import (
	"errors"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func errorRules(errs []error) []string {
	rules := make([]string, 0, len(errs))
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			rules = append(rules, vErr.Severity.String()+":"+vErr.Rule)
		} else {
			rules = append(rules, "unknown:"+err.Error())
		}
	}
	return rules
}

func TestSchemaUnion_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
		opts []validation.Option
	}{
		{
			name: "native schema with discriminator",
			doc: `type: object
discriminator: petType
required: [petType]
properties:
  petType:
    type: string
  name:
    type: string
    default: rex
externalDocs:
  url: https://example.com/pets
deprecated: false
`,
		},
		{
			name: "boolean schema",
			doc:  "false\n",
		},
		{
			name: "draft 07 keywords",
			doc: `type: array
items:
  - type: string
  - type: integer
additionalItems: false
definitions:
  positive:
    type: integer
    exclusiveMinimum: 0
dependencies:
  a: [b]
  c:
    required: [d]
if:
  minItems: 1
then:
  maxItems: 5
`,
		},
		{
			name: "reference object siblings are ignored",
			doc:  "$ref: '#/components/schemas/User'\ntype: not-a-type\n",
		},
		{
			name: "nested reference siblings are ignored",
			doc:  "type: object\nproperties:\n  user:\n    $ref: '#/components/schemas/User'\n    type: 7\n",
		},
		{
			name: "property named $ref is a property not a reference",
			doc:  "type: object\nproperties:\n  $ref:\n    type: string\n",
		},
		{
			name: "integer default for number type",
			doc:  "type: number\ndefault: 1\n",
		},
		{
			name: "draft 07 format does not apply asyncapi discriminator rules",
			doc:  "schemaFormat: application/schema+json;version=draft-07\nschema:\n  type: object\n  discriminator: missing\n",
		},
		{
			name: "native multi format schema is validated",
			doc:  "schemaFormat: application/vnd.aai.asyncapi+yaml;version=3.0.0\nschema:\n  type: string\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, unmarshalErrs := unmarshalSchemaUnion(t, tt.doc)
			require.Empty(t, unmarshalErrs, "should have no unmarshal errors")

			errs := u.Validate(t.Context(), tt.opts...)
			assert.Empty(t, errs, "should have no validation errors")
		})
	}
}

func TestSchemaUnion_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		doc           string
		expectedRules []string
		expectedLines []int
	}{
		{
			name:          "invalid type value",
			doc:           "type: object\nproperties:\n  id:\n    type: text\n",
			expectedRules: []string{"error:" + validation.RuleValidationInvalidSchema},
			expectedLines: []int{4},
		},
		{
			name:          "minLength must be non negative",
			doc:           "type: string\nminLength: -1\n",
			expectedRules: []string{"error:" + validation.RuleValidationInvalidSchema},
			expectedLines: []int{2},
		},

		{
			name: "discriminator not defined or required",
			doc:  "type: object\ndiscriminator: petType\nproperties:\n  name:\n    type: string\n",
			expectedRules: []string{
				"error:" + validation.RuleValidationInvalidTarget,
				"error:" + validation.RuleValidationRequiredField,
			},
			expectedLines: []int{2, 2},
		},
		{
			name:          "nested discriminator is checked",
			doc:           "type: object\nproperties:\n  pet:\n    type: object\n    discriminator: kind\n    required: [kind]\n",
			expectedRules: []string{"error:" + validation.RuleValidationInvalidTarget},
			expectedLines: []int{5},
		},
		{
			name:          "default must conform to type",
			doc:           "type: string\ndefault: 1\n",
			expectedRules: []string{"error:" + validation.RuleValidationTypeMismatch},
			expectedLines: []int{2},
		},
		{
			name:          "external docs url is required",
			doc:           "type: string\nexternalDocs:\n  description: more\n",
			expectedRules: []string{"error:" + validation.RuleValidationRequiredField},
			expectedLines: []int{3},
		},
		{
			name:          "foreign format is reported as not validated",
			doc:           "schemaFormat: application/vnd.apache.avro;version=1.9.0\nschema:\n  type: record\n",
			expectedRules: []string{"warning:" + validation.RuleValidationUnsupportedFormat},
			expectedLines: []int{1},
		},
		{
			name:          "unsupported json schema draft is reported as not validated",
			doc:           "schemaFormat: application/schema+json;version=draft-04\nschema:\n  type: string\n",
			expectedRules: []string{"warning:" + validation.RuleValidationUnsupportedFormat},
			expectedLines: []int{1},
		},
		{
			name: "protobuf must be a string",
			doc:  "schemaFormat: application/vnd.google.protobuf;version=3\nschema:\n  message: Point\n",
			expectedRules: []string{
				"error:" + validation.RuleValidationTypeMismatch,
				"warning:" + validation.RuleValidationUnsupportedFormat,
			},
			expectedLines: []int{3, 1},
		},
		{
			name:          "invalid schema format media type",
			doc:           "schemaFormat: 'not a media type;;'\nschema:\n  type: string\n",
			expectedRules: []string{"error:" + validation.RuleValidationInvalidFormat},
			expectedLines: []int{1},
		},
		{
			name:          "native multi format schema content is validated",
			doc:           "schemaFormat: application/vnd.aai.asyncapi;version=3.1.0\nschema:\n  type: string\n  default: 5\n",
			expectedRules: []string{"error:" + validation.RuleValidationTypeMismatch},
			expectedLines: []int{4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, unmarshalErrs := unmarshalSchemaUnion(t, tt.doc)
			require.Empty(t, unmarshalErrs, "should have no unmarshal errors")

			errs := u.Validate(t.Context())
			assert.Equal(t, tt.expectedRules, errorRules(errs), "rules should match")

			lines := make([]int, 0, len(errs))
			for _, err := range errs {
				var vErr *validation.Error
				require.ErrorAs(t, err, &vErr, "should be a validation error")
				lines = append(lines, vErr.GetLineNumber())
			}
			assert.Equal(t, tt.expectedLines, lines, "error lines should match")
		})
	}
}

func TestSchemaUnion_Unmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		doc           string
		expectedRules []string
	}{
		{
			name:          "draft 04 style boolean exclusiveMinimum",
			doc:           "type: number\nexclusiveMinimum: true\n",
			expectedRules: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:          "non string discriminator",
			doc:           "type: object\ndiscriminator:\n  propertyName: petType\n",
			expectedRules: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, unmarshalErrs := unmarshalSchemaUnion(t, tt.doc)
			assert.Equal(t, tt.expectedRules, errorRules(unmarshalErrs), "unmarshal errors should match")
		})
	}
}

func TestSchema_Validate_WithSchemaFormat_Draft07_Success(t *testing.T) {
	t.Parallel()

	u, unmarshalErrs := unmarshalSchemaUnion(t, "type: string\ndefault: 1\n")
	require.Empty(t, unmarshalErrs, "should have no unmarshal errors")

	native := u.Validate(t.Context())
	assert.Len(t, native, 1, "native schemas require defaults to conform to the type")

	draft07 := u.Validate(t.Context(), asyncapi.WithSchemaFormat("application/schema+json;version=draft-07"))
	assert.Empty(t, draft07, "draft 07 does not require defaults to conform to the type")
}
