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

func TestChannel_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "address with a query",
			doc:      minimalDoc("channels:\n  c:\n    address: orders?x=1\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "address parameter is not defined",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "repeated undefined address parameter is reported once",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}.{id}\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "address with an empty expression",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "parameter is not used in the address",
			doc:      minimalDoc("channels:\n  c:\n    address: orders\n    parameters:\n      id: {}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidTarget},
		},
		{
			name:     "parameter name has invalid characters",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{a.b}\n    parameters:\n      a.b: {}\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "parameter location is not a runtime expression",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}\n    parameters:\n      id:\n        location: $request.body#/id\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "parameter default is not in enum",
			doc:      minimalDoc("channels:\n  c:\n    address: orders.{id}\n    parameters:\n      id:\n        enum: ['1', '2']\n        default: '3'\n"),
			expected: []string{"error:" + validation.RuleValidationAllowedValues},
		},
		{
			name:     "server reference without $ref",
			doc:      minimalDoc("channels:\n  c:\n    servers:\n      - $ref: ''\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "null channel reference",
			doc:      minimalDoc("channels:\n  c:\n    $ref: null\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "inline message is validated",
			doc:      minimalDoc("channels:\n  c:\n    messages:\n      m:\n        contentType: 'json;;'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
	})
}

func TestChannel_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "address parameters defined",
			doc:      minimalDoc("channels:\n  c:\n    address: users.{userId}.{event}\n    parameters:\n      userId:\n        location: $message.payload#/user/id\n      event:\n        $ref: '#/components/parameters/event'\n"),
			expected: []string{},
		},
		{
			name:     "address with an MQTT multi-level wildcard",
			doc:      minimalDoc("channels:\n  c:\n    address: sensors/#\n"),
			expected: []string{},
		},
		{
			name:     "null address",
			doc:      minimalDoc("channels:\n  c:\n    address: null\n"),
			expected: []string{},
		},
		{
			name:     "channel without address",
			doc:      minimalDoc("channels:\n  c: {}\n"),
			expected: []string{},
		},
	})
}

func TestChannel_Validate_Nil_Success(t *testing.T) {
	t.Parallel()

	runNilValidateCases(t, []nilValidateCase{
		{name: "channel", model: (*asyncapi.Channel)(nil)},
		{name: "parameter", model: (*asyncapi.Parameter)(nil)},
	})
}

func TestChannel_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.Channel{
		Address:      pointer.From("orders.{id}"),
		Messages:     sequencedmap.New(sequencedmap.NewElem("placed", &asyncapi.ReferencedMessage{Object: &asyncapi.Message{}})),
		Title:        pointer.From("Orders"),
		Summary:      pointer.From("Order events."),
		Description:  pointer.From("Carries order events."),
		Servers:      []*asyncapi.ReferenceObject{asyncapi.NewReferenceObject("#/servers/prod")},
		Parameters:   sequencedmap.New(sequencedmap.NewElem("id", &asyncapi.ReferencedParameter{Object: &asyncapi.Parameter{}})),
		Tags:         []*asyncapi.Tag{{Name: pointer.From("orders")}},
		ExternalDocs: &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")},
		Bindings:     &asyncapi.ReferencedBindings{Object: &asyncapi.Bindings{}},
		Extensions:   exts,
	}

	type getters struct {
		address      string
		messages     *sequencedmap.Map[string, *asyncapi.ReferencedMessage]
		title        string
		summary      string
		description  string
		servers      []*asyncapi.ReferenceObject
		parameters   *sequencedmap.Map[string, *asyncapi.ReferencedParameter]
		tags         []*asyncapi.Tag
		externalDocs *asyncapi.ExternalDocumentation
		bindings     *asyncapi.ReferencedBindings
		extensions   *extensions.Extensions
	}

	tests := []struct {
		name     string
		channel  *asyncapi.Channel
		expected getters
	}{
		{
			name:     "nil channel returns defaults",
			channel:  nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			channel:  &asyncapi.Channel{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:    "populated fields are returned",
			channel: populated,
			expected: getters{
				address:      "orders.{id}",
				messages:     populated.Messages,
				title:        "Orders",
				summary:      "Order events.",
				description:  "Carries order events.",
				servers:      populated.Servers,
				parameters:   populated.Parameters,
				tags:         populated.Tags,
				externalDocs: populated.ExternalDocs,
				bindings:     populated.Bindings,
				extensions:   exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				address:      tt.channel.GetAddress(),
				messages:     tt.channel.GetMessages(),
				title:        tt.channel.GetTitle(),
				summary:      tt.channel.GetSummary(),
				description:  tt.channel.GetDescription(),
				servers:      tt.channel.GetServers(),
				parameters:   tt.channel.GetParameters(),
				tags:         tt.channel.GetTags(),
				externalDocs: tt.channel.GetExternalDocs(),
				bindings:     tt.channel.GetBindings(),
				extensions:   tt.channel.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestParameter_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))

	type getters struct {
		enum        []string
		defaultVal  string
		description string
		examples    []string
		location    string
		extensions  *extensions.Extensions
	}

	tests := []struct {
		name      string
		parameter *asyncapi.Parameter
		expected  getters
	}{
		{
			name:      "nil parameter returns defaults",
			parameter: nil,
			expected:  getters{extensions: extensions.New()},
		},
		{
			name:      "unset fields return defaults",
			parameter: &asyncapi.Parameter{},
			expected:  getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			parameter: &asyncapi.Parameter{
				Enum:        []string{"eu", "us"},
				Default:     pointer.From("eu"),
				Description: pointer.From("The region."),
				Examples:    []string{"us"},
				Location:    pointer.From("$message.payload#/region"),
				Extensions:  exts,
			},
			expected: getters{
				enum:        []string{"eu", "us"},
				defaultVal:  "eu",
				description: "The region.",
				examples:    []string{"us"},
				location:    "$message.payload#/region",
				extensions:  exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				enum:        tt.parameter.GetEnum(),
				defaultVal:  tt.parameter.GetDefault(),
				description: tt.parameter.GetDescription(),
				examples:    tt.parameter.GetExamples(),
				location:    tt.parameter.GetLocation(),
				extensions:  tt.parameter.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}
