package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/values"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
)

func TestMessage_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "headers are not an object schema",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      headers:\n        type: string\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "invalid content type",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      contentType: 'json;;'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "correlation ID location is not a runtime expression",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      correlationId:\n        location: $message.body#/id\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "correlation ID location has an invalid pointer",
			doc:      minimalDoc("components:\n  correlationIds:\n    c:\n      location: '$message.payload#id'\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidSyntax},
		},
		{
			name:     "example without headers or payload",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      examples:\n        - name: empty\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "example headers are not an object",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      examples:\n        - headers: [1]\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "payload schema is validated",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      payload:\n        type: string\n        default: 1\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "message trait headers are not an object schema",
			doc:      minimalDoc("components:\n  messageTraits:\n    t:\n      headers:\n        type: array\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "message trait cannot set a payload",
			doc:      minimalDoc("components:\n  messageTraits:\n    t:\n      payload:\n        type: object\n"),
			expected: []string{"error:" + validation.RuleValidationUnknownProperties},
		},
		{
			name:     "headers are a boolean schema",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      headers: true\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "message trait headers are a boolean schema",
			doc:      minimalDoc("components:\n  messageTraits:\n    t:\n      headers: false\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
	})
}

func TestMessage_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "multi format headers and payload",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      headers:\n        schemaFormat: application/schema+json;version=draft-07\n        schema:\n          type: object\n      payload:\n        schemaFormat: application/vnd.aai.asyncapi+json;version=3.1.0\n        schema:\n          type: string\n"),
			expected: []string{},
		},
		{
			name:     "example with payload only",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      examples:\n        - payload: 42\n"),
			expected: []string{},
		},
		{
			name:     "correlation ID for the whole header",
			doc:      minimalDoc("components:\n  correlationIds:\n    c:\n      location: $message.header\n"),
			expected: []string{},
		},
	})
}

func TestMessage_Validate_Nil_Success(t *testing.T) {
	t.Parallel()

	runNilValidateCases(t, []nilValidateCase{
		{name: "message", model: (*asyncapi.Message)(nil)},
		{name: "message trait", model: (*asyncapi.MessageTrait)(nil)},
		{name: "message example", model: (*asyncapi.MessageExample)(nil)},
		{name: "correlation ID", model: (*asyncapi.CorrelationID)(nil)},
	})
}

func TestMessage_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.Message{
		Headers:       jsonschema.NewSchemaUnionFromSchema(jsonschema.NewJSONSchemaFromBool(true)),
		Payload:       jsonschema.NewSchemaUnionFromSchema(jsonschema.NewJSONSchemaFromBool(false)),
		CorrelationID: &asyncapi.ReferencedCorrelationID{Object: &asyncapi.CorrelationID{Location: "$message.header#/id"}},
		ContentType:   pointer.From("application/json"),
		Name:          pointer.From("OrderPlaced"),
		Title:         pointer.From("Order placed"),
		Summary:       pointer.From("An order was placed."),
		Description:   pointer.From("Sent when an order is placed."),
		Tags:          []*asyncapi.Tag{{Name: pointer.From("orders")}},
		ExternalDocs:  &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")},
		Bindings:      &asyncapi.ReferencedBindings{Object: &asyncapi.Bindings{}},
		Examples:      []*asyncapi.MessageExample{{Name: pointer.From("example")}},
		Traits:        []*asyncapi.ReferencedMessageTrait{{Object: &asyncapi.MessageTrait{}}},
		Extensions:    exts,
	}

	type getters struct {
		headers       *jsonschema.SchemaUnion
		payload       *jsonschema.SchemaUnion
		correlationID *asyncapi.ReferencedCorrelationID
		contentType   string
		name          string
		title         string
		summary       string
		description   string
		tags          []*asyncapi.Tag
		externalDocs  *asyncapi.ExternalDocumentation
		bindings      *asyncapi.ReferencedBindings
		examples      []*asyncapi.MessageExample
		traits        []*asyncapi.ReferencedMessageTrait
		extensions    *extensions.Extensions
	}

	tests := []struct {
		name     string
		message  *asyncapi.Message
		expected getters
	}{
		{
			name:     "nil message returns defaults",
			message:  nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			message:  &asyncapi.Message{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:    "populated fields are returned",
			message: populated,
			expected: getters{
				headers:       populated.Headers,
				payload:       populated.Payload,
				correlationID: populated.CorrelationID,
				contentType:   "application/json",
				name:          "OrderPlaced",
				title:         "Order placed",
				summary:       "An order was placed.",
				description:   "Sent when an order is placed.",
				tags:          populated.Tags,
				externalDocs:  populated.ExternalDocs,
				bindings:      populated.Bindings,
				examples:      populated.Examples,
				traits:        populated.Traits,
				extensions:    exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				headers:       tt.message.GetHeaders(),
				payload:       tt.message.GetPayload(),
				correlationID: tt.message.GetCorrelationID(),
				contentType:   tt.message.GetContentType(),
				name:          tt.message.GetName(),
				title:         tt.message.GetTitle(),
				summary:       tt.message.GetSummary(),
				description:   tt.message.GetDescription(),
				tags:          tt.message.GetTags(),
				externalDocs:  tt.message.GetExternalDocs(),
				bindings:      tt.message.GetBindings(),
				examples:      tt.message.GetExamples(),
				traits:        tt.message.GetTraits(),
				extensions:    tt.message.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestMessageTrait_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.MessageTrait{
		Headers:       jsonschema.NewSchemaUnionFromSchema(jsonschema.NewJSONSchemaFromBool(true)),
		CorrelationID: &asyncapi.ReferencedCorrelationID{Object: &asyncapi.CorrelationID{Location: "$message.header#/id"}},
		ContentType:   pointer.From("application/json"),
		Name:          pointer.From("Envelope"),
		Title:         pointer.From("Envelope trait"),
		Summary:       pointer.From("Common envelope."),
		Description:   pointer.From("Applied to every message."),
		Tags:          []*asyncapi.Tag{{Name: pointer.From("envelope")}},
		ExternalDocs:  &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com/docs")},
		Bindings:      &asyncapi.ReferencedBindings{Object: &asyncapi.Bindings{}},
		Examples:      []*asyncapi.MessageExample{{Name: pointer.From("example")}},
		Extensions:    exts,
	}

	type getters struct {
		headers       *jsonschema.SchemaUnion
		correlationID *asyncapi.ReferencedCorrelationID
		contentType   string
		name          string
		title         string
		summary       string
		description   string
		tags          []*asyncapi.Tag
		externalDocs  *asyncapi.ExternalDocumentation
		bindings      *asyncapi.ReferencedBindings
		examples      []*asyncapi.MessageExample
		extensions    *extensions.Extensions
	}

	tests := []struct {
		name     string
		trait    *asyncapi.MessageTrait
		expected getters
	}{
		{
			name:     "nil trait returns defaults",
			trait:    nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			trait:    &asyncapi.MessageTrait{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:  "populated fields are returned",
			trait: populated,
			expected: getters{
				headers:       populated.Headers,
				correlationID: populated.CorrelationID,
				contentType:   "application/json",
				name:          "Envelope",
				title:         "Envelope trait",
				summary:       "Common envelope.",
				description:   "Applied to every message.",
				tags:          populated.Tags,
				externalDocs:  populated.ExternalDocs,
				bindings:      populated.Bindings,
				examples:      populated.Examples,
				extensions:    exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				headers:       tt.trait.GetHeaders(),
				correlationID: tt.trait.GetCorrelationID(),
				contentType:   tt.trait.GetContentType(),
				name:          tt.trait.GetName(),
				title:         tt.trait.GetTitle(),
				summary:       tt.trait.GetSummary(),
				description:   tt.trait.GetDescription(),
				tags:          tt.trait.GetTags(),
				externalDocs:  tt.trait.GetExternalDocs(),
				bindings:      tt.trait.GetBindings(),
				examples:      tt.trait.GetExamples(),
				extensions:    tt.trait.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestMessageExample_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.MessageExample{
		Headers:    yml.CreateStringNode("headers"),
		Payload:    yml.CreateStringNode("payload"),
		Name:       pointer.From("example"),
		Summary:    pointer.From("An example."),
		Extensions: exts,
	}

	type getters struct {
		headers    values.Value
		payload    values.Value
		name       string
		summary    string
		extensions *extensions.Extensions
	}

	tests := []struct {
		name     string
		example  *asyncapi.MessageExample
		expected getters
	}{
		{
			name:     "nil example returns defaults",
			example:  nil,
			expected: getters{extensions: extensions.New()},
		},
		{
			name:     "unset fields return defaults",
			example:  &asyncapi.MessageExample{},
			expected: getters{extensions: extensions.New()},
		},
		{
			name:    "populated fields are returned",
			example: populated,
			expected: getters{
				headers:    populated.Headers,
				payload:    populated.Payload,
				name:       "example",
				summary:    "An example.",
				extensions: exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				headers:    tt.example.GetHeaders(),
				payload:    tt.example.GetPayload(),
				name:       tt.example.GetName(),
				summary:    tt.example.GetSummary(),
				extensions: tt.example.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}

func TestCorrelationID_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))

	type getters struct {
		description string
		location    string
		extensions  *extensions.Extensions
	}

	tests := []struct {
		name          string
		correlationID *asyncapi.CorrelationID
		expected      getters
	}{
		{
			name:          "nil correlation ID returns defaults",
			correlationID: nil,
			expected:      getters{extensions: extensions.New()},
		},
		{
			name:          "unset fields return defaults",
			correlationID: &asyncapi.CorrelationID{},
			expected:      getters{extensions: extensions.New()},
		},
		{
			name: "populated fields are returned",
			correlationID: &asyncapi.CorrelationID{
				Description: pointer.From("Order ID."),
				Location:    "$message.header#/orderId",
				Extensions:  exts,
			},
			expected: getters{
				description: "Order ID.",
				location:    "$message.header#/orderId",
				extensions:  exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				description: tt.correlationID.GetDescription(),
				location:    tt.correlationID.GetLocation(),
				extensions:  tt.correlationID.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}
