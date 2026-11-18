package asyncapi_test

import (
	"bytes"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func errorStrings(errs []error) []string {
	msgs := make([]string, 0, len(errs))
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return msgs
}

func TestInfo_Getters_Success(t *testing.T) {
	t.Parallel()

	contact := &asyncapi.Contact{Name: pointer.From("Support")}
	license := &asyncapi.License{Name: "MIT"}
	tags := []*asyncapi.Tag{{Name: pointer.From("orders")}}
	externalDocs := &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")}
	exts := extensions.New(extensions.NewElem("x-audience", nil))

	type getters struct {
		title          string
		version        string
		description    string
		termsOfService string
		contact        *asyncapi.Contact
		license        *asyncapi.License
		tags           []*asyncapi.Tag
		externalDocs   *asyncapi.ExternalDocumentation
		extensions     *extensions.Extensions
	}

	tests := []struct {
		name     string
		info     *asyncapi.Info
		expected getters
	}{
		{
			name:     "nil info returns defaults",
			info:     nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			info:     &asyncapi.Info{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			info: &asyncapi.Info{
				Title:          "Orders",
				Version:        "1.0.0",
				Description:    pointer.From("Order events"),
				TermsOfService: pointer.From("https://example.com/terms"),
				Contact:        contact,
				License:        license,
				Tags:           tags,
				ExternalDocs:   externalDocs,
				Extensions:     exts,
			},
			expected: getters{
				title:          "Orders",
				version:        "1.0.0",
				description:    "Order events",
				termsOfService: "https://example.com/terms",
				contact:        contact,
				license:        license,
				tags:           tags,
				externalDocs:   externalDocs,
				extensions:     exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				title:          tt.info.GetTitle(),
				version:        tt.info.GetVersion(),
				description:    tt.info.GetDescription(),
				termsOfService: tt.info.GetTermsOfService(),
				contact:        tt.info.GetContact(),
				license:        tt.info.GetLicense(),
				tags:           tt.info.GetTags(),
				externalDocs:   tt.info.GetExternalDocs(),
				extensions:     tt.info.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestContact_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))

	type getters struct {
		name       string
		url        string
		email      string
		extensions *extensions.Extensions
	}

	tests := []struct {
		name     string
		contact  *asyncapi.Contact
		expected getters
	}{
		{
			name:     "nil contact returns defaults",
			contact:  nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			contact:  &asyncapi.Contact{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			contact: &asyncapi.Contact{
				Name:       pointer.From("Support"),
				URL:        pointer.From("https://example.com/support"),
				Email:      pointer.From("support@example.com"),
				Extensions: exts,
			},
			expected: getters{
				name:       "Support",
				url:        "https://example.com/support",
				email:      "support@example.com",
				extensions: exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				name:       tt.contact.GetName(),
				url:        tt.contact.GetURL(),
				email:      tt.contact.GetEmail(),
				extensions: tt.contact.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestLicense_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-spdx", nil))

	type getters struct {
		name       string
		url        string
		extensions *extensions.Extensions
	}

	tests := []struct {
		name     string
		license  *asyncapi.License
		expected getters
	}{
		{
			name:     "nil license returns defaults",
			license:  nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			license:  &asyncapi.License{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			license: &asyncapi.License{
				Name:       "MIT",
				URL:        pointer.From("https://opensource.org/licenses/MIT"),
				Extensions: exts,
			},
			expected: getters{
				name:       "MIT",
				url:        "https://opensource.org/licenses/MIT",
				extensions: exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				name:       tt.license.GetName(),
				url:        tt.license.GetURL(),
				extensions: tt.license.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestTag_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-group", nil))

	type getters struct {
		isReference bool
		name        string
		description string
		extensions  *extensions.Extensions
	}

	tests := []struct {
		name     string
		tag      *asyncapi.Tag
		expected getters
	}{
		{
			name:     "nil tag returns defaults",
			tag:      nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			tag:      &asyncapi.Tag{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			tag: &asyncapi.Tag{
				Name:        pointer.From("orders"),
				Description: pointer.From("Order events"),
				Extensions:  exts,
			},
			expected: getters{
				name:        "orders",
				description: "Order events",
				extensions:  exts,
			},
		},
		{
			name: "reference tag",
			tag:  &asyncapi.Tag{Ref: pointer.From(references.Reference("#/components/tags/orders"))},
			expected: getters{
				isReference: true,
				extensions:  extensions.New(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				isReference: tt.tag.IsReference(),
				name:        tt.tag.GetName(),
				description: tt.tag.GetDescription(),
				extensions:  tt.tag.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestInfo_Validate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yml  string
	}{
		{
			name: "minimal info",
			yml:  "title: Orders\nversion: 1.0.0\n",
		},
		{
			name: "info with all fields",
			yml: `title: Orders
version: 1.0.0
description: Order events
termsOfService: https://example.com/terms
contact:
  name: Support
  url: https://example.com/support
  email: support@example.com
license:
  name: MIT
  url: https://opensource.org/licenses/MIT
tags:
  - name: orders
    description: Order events
    externalDocs:
      url: https://example.com/docs/orders
  - $ref: '#/components/tags/payments'
  - $ref: '#/components/tags/payments'
externalDocs:
  url: https://example.com/docs
x-audience: public
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var info asyncapi.Info
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &info)
			require.NoError(t, err, "unmarshal should succeed")
			require.Empty(t, validationErrs, "unmarshal should report no errors")

			assert.Empty(t, info.Validate(t.Context()), "info should have no validation errors")
			assert.True(t, info.Valid, "info should be valid")
		})
	}
}

func TestInfo_Validate_InCode_Success(t *testing.T) {
	t.Parallel()

	info := &asyncapi.Info{
		Title:   "Orders",
		Version: "1.0.0",
		Tags:    []*asyncapi.Tag{nil, {Name: pointer.From("orders")}},
	}

	assert.Empty(t, info.Validate(t.Context()), "nil tags should be skipped")
	assert.True(t, info.Valid, "info built in code should be valid")
}

func TestInfo_Validate_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yml      string
		expected []string
	}{
		{
			name:     "empty title",
			yml:      "title: ''\nversion: 1.0.0\n",
			expected: []string{"[1:8] error validation-required-field `info.title` is required"},
		},
		{
			name:     "empty version",
			yml:      "title: Orders\nversion: ''\n",
			expected: []string{"[2:10] error validation-required-field `info.version` is required"},
		},
		{
			name:     "invalid terms of service",
			yml:      "title: Orders\nversion: 1.0.0\ntermsOfService: '%zz'\n",
			expected: []string{"[3:17] error validation-invalid-format info.termsOfService is not a valid uri: parse \"%zz\": invalid URL escape \"%zz\""},
		},
		{
			name:     "invalid contact url",
			yml:      "title: Orders\nversion: 1.0.0\ncontact:\n  url: '%zz'\n",
			expected: []string{"[4:8] error validation-invalid-format contact.url is not a valid uri: parse \"%zz\": invalid URL escape \"%zz\""},
		},
		{
			name:     "invalid contact email",
			yml:      "title: Orders\nversion: 1.0.0\ncontact:\n  email: nope\n",
			expected: []string{"[4:10] error validation-invalid-format contact.email is not a valid email address: mail: missing '@' or angle-addr"},
		},
		{
			name:     "empty license name",
			yml:      "title: Orders\nversion: 1.0.0\nlicense:\n  name: ''\n",
			expected: []string{"[4:9] error validation-required-field `license.name` is required"},
		},
		{
			name:     "invalid license url",
			yml:      "title: Orders\nversion: 1.0.0\nlicense:\n  name: MIT\n  url: '%zz'\n",
			expected: []string{"[5:8] error validation-invalid-format license.url is not a valid uri: parse \"%zz\": invalid URL escape \"%zz\""},
		},
		{
			name:     "empty tag name",
			yml:      "title: Orders\nversion: 1.0.0\ntags:\n  - name: ''\n",
			expected: []string{"[4:11] error validation-required-field `tag.name` is required"},
		},
		{
			name:     "invalid tag reference",
			yml:      "title: Orders\nversion: 1.0.0\ntags:\n  - $ref: '#'\n",
			expected: []string{"[4:11] error validation-invalid-reference invalid reference JSON pointer: empty"},
		},
		{
			name:     "duplicate tag names",
			yml:      "title: Orders\nversion: 1.0.0\ntags:\n  - name: orders\n  - name: orders\n",
			expected: []string{"[5:5] error validation-duplicate-key info.tags name `orders` is not unique"},
		},
		{
			name:     "tag external docs without url",
			yml:      "title: Orders\nversion: 1.0.0\ntags:\n  - name: orders\n    externalDocs:\n      description: d\n",
			expected: []string{"[6:7] error validation-required-field `externalDocs.url` is required"},
		},
		{
			name:     "external docs without url",
			yml:      "title: Orders\nversion: 1.0.0\nexternalDocs:\n  description: d\n",
			expected: []string{"[4:3] error validation-required-field `externalDocs.url` is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var info asyncapi.Info
			validationErrs, err := marshaller.Unmarshal(t.Context(), bytes.NewBufferString(tt.yml), &info)
			require.NoError(t, err, "unmarshal should succeed")

			validationErrs = append(validationErrs, info.Validate(t.Context())...)
			assert.Equal(t, tt.expected, errorStrings(validationErrs), "validation errors should match")
			assert.False(t, info.Valid, "info should not be valid")
		})
	}
}
