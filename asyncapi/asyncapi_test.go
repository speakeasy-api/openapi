package asyncapi_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const streetlightsDoc = `# Streetlights API
asyncapi: 3.1.0
id: 'urn:example:com:smartylighting:streetlights:server'
info:
  title: Streetlights API
  version: 1.0.0
  description: |
    The Smartylighting Streetlights API allows you to remotely manage the city lights.
  termsOfService: https://asyncapi.org/terms/
  contact:
    name: API Support
    url: https://www.asyncapi.org/support
    email: support@asyncapi.org
  license:
    name: Apache 2.0
    url: https://www.apache.org/licenses/LICENSE-2.0.html
  tags:
    - name: e-commerce
    - $ref: '#/components/tags/lighting'
  externalDocs:
    description: Find more info here
    url: https://www.asyncapi.org
  x-audience: public
defaultContentType: application/json
servers:
  production:
    host: test.mosquitto.org:{port}
    protocol: mqtt
    variables:
      port:
        default: '1883'
channels:
  lightingMeasured:
    address: smartylighting/streetlights/1/0/event/{streetlightId}/lighting/measured
operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
components:
  tags:
    lighting:
      name: lighting
x-owner: platform
`

func unmarshal(t *testing.T, doc string) (*asyncapi.AsyncAPI, []error) {
	t.Helper()

	a, validationErrs, err := asyncapi.Unmarshal(t.Context(), strings.NewReader(doc))
	require.NoError(t, err, "unmarshal should succeed")
	require.NotNil(t, a, "document should not be nil")
	return a, validationErrs
}

func marshal(t *testing.T, a *asyncapi.AsyncAPI) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, asyncapi.Marshal(t.Context(), a, &buf), "marshal should succeed")
	return buf.String()
}

func validationRules(errs []error) []string {
	rules := make([]string, 0, len(errs))
	for _, err := range errs {
		var vErr *validation.Error
		if errors.As(err, &vErr) {
			rules = append(rules, vErr.Rule)
		} else {
			rules = append(rules, "unknown:"+err.Error())
		}
	}
	return rules
}

func TestUnmarshal_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs := unmarshal(t, streetlightsDoc)
	assert.Empty(t, validationErrs, "should have no validation errors")
	assert.True(t, a.Valid, "document should be valid")

	assert.Equal(t, "3.1.0", a.GetAsyncAPI(), "version should be parsed")
	assert.Equal(t, "urn:example:com:smartylighting:streetlights:server", a.GetID(), "id should be parsed")
	assert.Equal(t, "application/json", a.GetDefaultContentType(), "default content type should be parsed")

	info := a.GetInfo()
	assert.Equal(t, "Streetlights API", info.GetTitle(), "title should be parsed")
	assert.Equal(t, "1.0.0", info.GetVersion(), "info version should be parsed")
	assert.Equal(t, "support@asyncapi.org", info.GetContact().GetEmail(), "contact should be parsed")
	assert.Equal(t, "Apache 2.0", info.GetLicense().GetName(), "license should be parsed")
	require.Len(t, info.GetTags(), 2, "tags should be parsed")
	assert.Equal(t, "e-commerce", info.GetTags()[0].GetName(), "tag name should be parsed")
	assert.True(t, info.GetTags()[1].IsReference(), "tag reference should be parsed")
	assert.Equal(t, "https://www.asyncapi.org", info.GetExternalDocs().GetURL(), "external docs should be parsed")
	assert.True(t, info.GetExtensions().Has("x-audience"), "info extensions should be parsed")
	assert.True(t, a.GetExtensions().Has("x-owner"), "root extensions should be parsed")

	assert.NotNil(t, a.Servers, "servers should be preserved")
	assert.NotNil(t, a.Channels, "channels should be preserved")
	assert.NotNil(t, a.Operations, "operations should be preserved")
	assert.NotNil(t, a.Components, "components should be preserved")
}

func TestMarshal_RoundTrip_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
	}{
		{name: "yaml", doc: streetlightsDoc},
		{
			name: "json",
			doc: `{
  "asyncapi": "3.0.0",
  "info": {
    "title": "Account Service",
    "version": "1.0.0"
  },
  "channels": {
    "userSignedup": {
      "address": "user/signedup"
    }
  }
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, validationErrs := unmarshal(t, tt.doc)
			require.Empty(t, validationErrs, "should have no validation errors")
			assert.Equal(t, tt.doc, marshal(t, a), "unmodified document should round trip unchanged")
		})
	}
}

func TestMarshal_Mutate_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs := unmarshal(t, streetlightsDoc)
	require.Empty(t, validationErrs, "should have no validation errors")

	a.Info.Title = "Streetlights Events"
	a.Info.License = nil
	a.Info.Tags = a.Info.Tags[:1]
	a.DefaultContentType = pointer.From("application/cloudevents+json")

	out := marshal(t, a)
	expected := strings.NewReplacer(
		"  title: Streetlights API\n", "  title: Streetlights Events\n",
		"  license:\n    name: Apache 2.0\n    url: https://www.apache.org/licenses/LICENSE-2.0.html\n", "",
		"    - $ref: '#/components/tags/lighting'\n", "",
		"defaultContentType: application/json\n", "defaultContentType: application/cloudevents+json\n",
	).Replace(streetlightsDoc)
	assert.Equal(t, expected, out, "mutations should sync while preserving the rest of the document")

	reparsed, reparsedErrs := unmarshal(t, out)
	assert.Empty(t, reparsedErrs, "mutated document should be valid")
	assert.Equal(t, "Streetlights Events", reparsed.Info.Title, "mutation should survive a round trip")
}

func TestMarshal_NewDocument_Success(t *testing.T) {
	t.Parallel()

	a := asyncapi.NewAsyncAPI()
	a.Info = asyncapi.Info{Title: "Orders", Version: "0.1.0"}

	out := marshal(t, a)
	assert.Equal(t, "asyncapi: 3.1.0\ninfo:\n  title: Orders\n  version: 0.1.0\n", out, "new document should marshal")

	_, validationErrs := unmarshal(t, out)
	assert.Empty(t, validationErrs, "new document should be valid")
}

func TestValidate_NewDocument_Success(t *testing.T) {
	t.Parallel()

	a := asyncapi.NewAsyncAPI()
	a.Info = asyncapi.Info{
		Title:   "Orders",
		Version: "0.1.0",
		Tags:    []*asyncapi.Tag{{Name: pointer.From("orders")}},
	}

	assert.Empty(t, a.Validate(t.Context()), "new document should have no validation errors")
	assert.True(t, a.Valid, "new document built in code should be valid")
	assert.True(t, a.Info.Valid, "info built in code should be valid")
}

func TestValidate_Nil_Success(t *testing.T) {
	t.Parallel()

	var a *asyncapi.AsyncAPI
	assert.Nil(t, a.Validate(t.Context()), "validating a nil document should not panic")
}

func TestSupportedMinorVersions_ReturnsCopy(t *testing.T) {
	t.Parallel()

	versions := asyncapi.SupportedMinorVersions()
	assert.Equal(t, []string{"3.0", "3.1"}, versions, "should return the supported minor versions")

	versions[0] = "9.9"
	assert.Equal(t, []string{"3.0", "3.1"}, asyncapi.SupportedMinorVersions(), "mutating the result should not change the policy")
	assert.True(t, asyncapi.IsSupportedVersion("3.0.0"), "version policy should be unchanged")
}

func TestUnmarshal_Versions_Success(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"3.0.0", "3.1.0", "3.1.4", "3.0.0-rc1"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			_, validationErrs := unmarshal(t, "asyncapi: "+version+"\ninfo:\n  title: T\n  version: '1'\n")
			assert.Empty(t, validationErrs, "supported version should be valid")
		})
	}
}

func TestUnmarshal_ValidationErrors_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		doc           string
		expectedRules []string
	}{
		{
			name:          "unsupported 2.x version",
			doc:           "asyncapi: 2.6.0\ninfo:\n  title: T\n  version: '1'\n",
			expectedRules: []string{validation.RuleValidationSupportedVersion},
		},
		{
			name:          "unsupported future minor",
			doc:           "asyncapi: 3.2.0\ninfo:\n  title: T\n  version: '1'\n",
			expectedRules: []string{validation.RuleValidationSupportedVersion},
		},
		{
			name:          "invalid version string",
			doc:           "asyncapi: '3.1'\ninfo:\n  title: T\n  version: '1'\n",
			expectedRules: []string{validation.RuleValidationInvalidFormat},
		},
		{
			name:          "missing info",
			doc:           "asyncapi: 3.1.0\n",
			expectedRules: []string{validation.RuleValidationRequiredField},
		},
		{
			name:          "missing info title and version",
			doc:           "asyncapi: 3.1.0\ninfo:\n  description: d\n",
			expectedRules: []string{validation.RuleValidationRequiredField, validation.RuleValidationRequiredField},
		},
		{
			name:          "id is not a uri",
			doc:           "asyncapi: 3.1.0\nid: not a uri\ninfo:\n  title: T\n  version: '1'\n",
			expectedRules: []string{validation.RuleValidationInvalidFormat},
		},
		{
			name:          "invalid default content type",
			doc:           "asyncapi: 3.1.0\ndefaultContentType: 'json;;'\ninfo:\n  title: T\n  version: '1'\n",
			expectedRules: []string{validation.RuleValidationInvalidFormat},
		},
		{
			name:          "invalid contact email",
			doc:           "asyncapi: 3.1.0\ninfo:\n  title: T\n  version: '1'\n  contact:\n    email: nope\n",
			expectedRules: []string{validation.RuleValidationInvalidFormat},
		},
		{
			name:          "duplicate tag names",
			doc:           "asyncapi: 3.1.0\ninfo:\n  title: T\n  version: '1'\n  tags:\n    - name: a\n    - name: a\n",
			expectedRules: []string{validation.RuleValidationDuplicateKey},
		},
		{
			name:          "tag without name",
			doc:           "asyncapi: 3.1.0\ninfo:\n  title: T\n  version: '1'\n  tags:\n    - description: d\n",
			expectedRules: []string{validation.RuleValidationRequiredField},
		},
		{
			name:          "tag without name still validates external docs",
			doc:           "asyncapi: 3.1.0\ninfo:\n  title: T\n  version: '1'\n  tags:\n    - externalDocs:\n        description: d\n",
			expectedRules: []string{validation.RuleValidationRequiredField, validation.RuleValidationRequiredField},
		},
		{
			name:          "external docs without url",
			doc:           "asyncapi: 3.1.0\ninfo:\n  title: T\n  version: '1'\n  externalDocs:\n    description: d\n",
			expectedRules: []string{validation.RuleValidationRequiredField},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, validationErrs := unmarshal(t, tt.doc)
			assert.Equal(t, tt.expectedRules, validationRules(validationErrs), "validation rules should match")
			assert.False(t, a.Valid, "document should not be valid")
		})
	}
}

func TestUnmarshal_SkipValidation_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs, err := asyncapi.Unmarshal(t.Context(), strings.NewReader("asyncapi: 2.6.0\ninfo:\n  title: T\n  version: '1'\n"), asyncapi.WithSkipValidation())
	require.NoError(t, err, "unmarshal should succeed")
	assert.Empty(t, validationErrs, "validation should be skipped")
	assert.Equal(t, "2.6.0", a.GetAsyncAPI(), "unsupported documents can still be loaded for editing")
}

func TestUnmarshal_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
	}{
		{name: "empty document", doc: ""},
		{name: "invalid yaml", doc: "asyncapi: [3.1.0\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := asyncapi.Unmarshal(t.Context(), strings.NewReader(tt.doc))
			require.Error(t, err, "unmarshal should fail")
		})
	}
}

func TestAsyncAPI_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-owner", nil))
	populated := &asyncapi.AsyncAPI{
		AsyncAPI:           "3.0.0",
		ID:                 pointer.From("urn:example:orders"),
		Info:               asyncapi.Info{Title: "Orders"},
		DefaultContentType: pointer.From("application/json"),
		Extensions:         exts,
	}
	unset := &asyncapi.AsyncAPI{}

	type getters struct {
		asyncAPI           string
		id                 string
		info               *asyncapi.Info
		defaultContentType string
		extensions         *extensions.Extensions
	}

	tests := []struct {
		name     string
		doc      *asyncapi.AsyncAPI
		expected getters
	}{
		{
			name:     "nil document returns defaults",
			doc:      nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			doc:      unset,
			expected: getters{info: &unset.Info, extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			doc:  populated,
			expected: getters{
				asyncAPI:           "3.0.0",
				id:                 "urn:example:orders",
				info:               &populated.Info,
				defaultContentType: "application/json",
				extensions:         exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				asyncAPI:           tt.doc.GetAsyncAPI(),
				id:                 tt.doc.GetID(),
				info:               tt.doc.GetInfo(),
				defaultContentType: tt.doc.GetDefaultContentType(),
				extensions:         tt.doc.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestAsyncAPI_GetInfo_ReturnsField(t *testing.T) {
	t.Parallel()

	a := asyncapi.NewAsyncAPI()
	a.GetInfo().Title = "Orders"
	assert.Equal(t, "Orders", a.Info.Title, "GetInfo should return a pointer to the document's Info field")
}

func TestAsyncAPI_Sync_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs := unmarshal(t, streetlightsDoc)
	require.Empty(t, validationErrs, "should have no validation errors")

	a.Info.Title = "Streetlights Events"
	a.DefaultContentType = pointer.From("application/cloudevents+json")
	require.NoError(t, a.Sync(t.Context()), "sync should succeed")

	c := a.GetCore()
	assert.Equal(t, "Streetlights Events", c.Info.Value.Title.Value, "core info title should be synced")
	assert.Equal(t, "application/cloudevents+json", *c.DefaultContentType.Value, "core default content type should be synced")
	assert.Equal(t, "application/cloudevents+json", c.DefaultContentType.ValueNode.Value, "yaml node should be synced")
}

func TestIsSupportedVersion_Success(t *testing.T) {
	t.Parallel()

	assert.True(t, asyncapi.IsSupportedVersion(asyncapi.Version), "the package version should be supported")
	assert.True(t, asyncapi.IsSupportedVersion("3.0.0"), "3.0 should be supported")
	assert.False(t, asyncapi.IsSupportedVersion("2.6.0"), "2.x should not be supported")
}
