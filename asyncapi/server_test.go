package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
)

func TestServer_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "missing host and protocol",
			doc:      minimalDoc("servers:\n  prod:\n    description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField, "error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "host includes a scheme",
			doc:      minimalDoc("servers:\n  prod:\n    host: mqtt://broker.example.com\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "host variable is not defined",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{region}.example.com'\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "pathname variable is not defined",
			doc:      minimalDoc("servers:\n  prod:\n    host: example.com\n    pathname: /{tenant}\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "host with an empty expression",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{}.example.com'\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "variable default is not in enum",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{region}.example.com'\n    protocol: mqtt\n    variables:\n      region:\n        enum: [eu, us]\n        default: au\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "server name has invalid characters",
			doc:      minimalDoc("servers:\n  prod server:\n    host: example.com\n    protocol: mqtt\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "duplicate server tags",
			doc:      minimalDoc("servers:\n  prod:\n    host: example.com\n    protocol: mqtt\n    tags:\n      - name: a\n      - name: a\n"),
			expected: []string{"error:" + validation.RuleValidationDuplicateKey},
		},
	})
}

func TestServer_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "variables defined in host and pathname",
			doc:      minimalDoc("servers:\n  prod:\n    host: '{region}.example.com:{port}'\n    pathname: /{tenant}\n    protocol: mqtt\n    variables:\n      region:\n        default: eu\n      port:\n        $ref: '#/components/serverVariables/port'\n      tenant: {}\n"),
			expected: []string{},
		},
		{
			name:     "server reference",
			doc:      minimalDoc("servers:\n  prod:\n    $ref: '#/components/servers/prod'\n"),
			expected: []string{},
		},
	})
}

func TestServer_Validate_Nil_Success(t *testing.T) {
	t.Parallel()

	runNilValidateCases(t, []nilValidateCase{
		{name: "server", model: (*asyncapi.Server)(nil)},
		{name: "server variable", model: (*asyncapi.ServerVariable)(nil)},
	})
}

func TestServer_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.Server{
		Host:            "{region}.example.com",
		Protocol:        "mqtt",
		ProtocolVersion: pointer.From("5"),
		Pathname:        pointer.From("/orders"),
		Description:     pointer.From("Production broker."),
		Title:           pointer.From("Production"),
		Summary:         pointer.From("The production broker."),
		Variables:       sequencedmap.New(sequencedmap.NewElem("region", &asyncapi.ReferencedServerVariable{Object: &asyncapi.ServerVariable{}})),
		Security:        []*asyncapi.ReferencedSecurityScheme{{Object: &asyncapi.SecurityScheme{Type: asyncapi.SecuritySchemeTypePlain}}},
		Tags:            []*asyncapi.Tag{{Name: pointer.From("prod")}},
		ExternalDocs:    &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")},
		Bindings:        &asyncapi.ReferencedBindings{Object: &asyncapi.Bindings{}},
		Extensions:      exts,
	}

	type getters struct {
		host            string
		protocol        string
		protocolVersion string
		pathname        string
		description     string
		title           string
		summary         string
		variables       *sequencedmap.Map[string, *asyncapi.ReferencedServerVariable]
		security        []*asyncapi.ReferencedSecurityScheme
		tags            []*asyncapi.Tag
		externalDocs    *asyncapi.ExternalDocumentation
		bindings        *asyncapi.ReferencedBindings
		extensions      *extensions.Extensions
	}

	tests := []struct {
		name     string
		server   *asyncapi.Server
		expected getters
	}{
		{
			name:     "nil server returns defaults",
			server:   nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			server:   &asyncapi.Server{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:   "populated fields are returned",
			server: populated,
			expected: getters{
				host:            "{region}.example.com",
				protocol:        "mqtt",
				protocolVersion: "5",
				pathname:        "/orders",
				description:     "Production broker.",
				title:           "Production",
				summary:         "The production broker.",
				variables:       populated.Variables,
				security:        populated.Security,
				tags:            populated.Tags,
				externalDocs:    populated.ExternalDocs,
				bindings:        populated.Bindings,
				extensions:      exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				host:            tt.server.GetHost(),
				protocol:        tt.server.GetProtocol(),
				protocolVersion: tt.server.GetProtocolVersion(),
				pathname:        tt.server.GetPathname(),
				description:     tt.server.GetDescription(),
				title:           tt.server.GetTitle(),
				summary:         tt.server.GetSummary(),
				variables:       tt.server.GetVariables(),
				security:        tt.server.GetSecurity(),
				tags:            tt.server.GetTags(),
				externalDocs:    tt.server.GetExternalDocs(),
				bindings:        tt.server.GetBindings(),
				extensions:      tt.server.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestServerVariable_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))

	type getters struct {
		enum        []string
		defaultVal  string
		description string
		examples    []string
		extensions  *extensions.Extensions
	}

	tests := []struct {
		name     string
		variable *asyncapi.ServerVariable
		expected getters
	}{
		{
			name:     "nil variable returns defaults",
			variable: nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			variable: &asyncapi.ServerVariable{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			variable: &asyncapi.ServerVariable{
				Enum:        []string{"eu", "us"},
				Default:     pointer.From("eu"),
				Description: pointer.From("The region."),
				Examples:    []string{"us"},
				Extensions:  exts,
			},
			expected: getters{
				enum:        []string{"eu", "us"},
				defaultVal:  "eu",
				description: "The region.",
				examples:    []string{"us"},
				extensions:  exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				enum:        tt.variable.GetEnum(),
				defaultVal:  tt.variable.GetDefault(),
				description: tt.variable.GetDescription(),
				examples:    tt.variable.GetExamples(),
				extensions:  tt.variable.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}
