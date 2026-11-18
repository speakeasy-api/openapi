package asyncapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
)

func TestComponents_Validate_Error(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "component key has invalid characters",
			doc:      minimalDoc("components:\n  messages:\n    order placed:\n      name: OrderPlaced\n"),
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
		{
			name:     "component schema is validated",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      type: integer\n      default: one\n"),
			expected: []string{"error:" + validation.RuleValidationTypeMismatch},
		},
		{
			name:     "foreign schema format is preserved but not validated",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/vnd.apache.avro;version=1.9.0\n      schema:\n        type: record\n"),
			expected: []string{"warning:" + validation.RuleValidationUnsupportedFormat},
		},
		{
			name:     "component tag is validated",
			doc:      minimalDoc("components:\n  tags:\n    t:\n      description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "component external docs are validated",
			doc:      minimalDoc("components:\n  externalDocs:\n    d:\n      description: d\n"),
			expected: []string{"error:" + validation.RuleValidationRequiredField},
		},
		{
			name:     "unknown component type",
			doc:      minimalDoc("components:\n  requestBodies: {}\n"),
			expected: []string{"warning:" + validation.RuleValidationUnknownProperties},
		},
		{
			name:     "reference tag with an unknown property",
			doc:      minimalDoc("components:\n  tags:\n    t:\n      $ref: '#/components/tags/u'\n      colour: red\n"),
			expected: []string{"warning:" + validation.RuleValidationUnknownProperties},
		},
	})
}

func TestComponents_Validate_Success(t *testing.T) {
	t.Parallel()

	runValidationCases(t, []validationCase{
		{
			name:     "component keys use every allowed character",
			doc:      minimalDoc("components:\n  messages:\n    order-placed.v1_X:\n      name: OrderPlaced\n"),
			expected: []string{},
		},
		{
			name:     "component references",
			doc:      minimalDoc("components:\n  messages:\n    m:\n      $ref: '#/components/messages/n'\n  tags:\n    t:\n      $ref: '#/components/tags/u'\n"),
			expected: []string{},
		},
	})
}

func TestComponents_Validate_Nil_Success(t *testing.T) {
	t.Parallel()

	runNilValidateCases(t, []nilValidateCase{
		{name: "components", model: (*asyncapi.Components)(nil)},
	})
}

func TestComponents_Getters_Success(t *testing.T) {
	t.Parallel()

	exts := extensions.New(extensions.NewElem("x-team", nil))
	populated := &asyncapi.Components{
		Schemas:           sequencedmap.New(sequencedmap.NewElem("s", jsonschema.NewSchemaUnionFromSchema(jsonschema.NewJSONSchemaFromBool(true)))),
		Servers:           sequencedmap.New(sequencedmap.NewElem("prod", &asyncapi.ReferencedServer{})),
		Channels:          sequencedmap.New(sequencedmap.NewElem("c", &asyncapi.ReferencedChannel{})),
		Operations:        sequencedmap.New(sequencedmap.NewElem("o", &asyncapi.ReferencedOperation{})),
		Messages:          sequencedmap.New(sequencedmap.NewElem("m", &asyncapi.ReferencedMessage{})),
		SecuritySchemes:   sequencedmap.New(sequencedmap.NewElem("s", &asyncapi.ReferencedSecurityScheme{})),
		ServerVariables:   sequencedmap.New(sequencedmap.NewElem("v", &asyncapi.ReferencedServerVariable{})),
		Parameters:        sequencedmap.New(sequencedmap.NewElem("p", &asyncapi.ReferencedParameter{})),
		CorrelationIDs:    sequencedmap.New(sequencedmap.NewElem("c", &asyncapi.ReferencedCorrelationID{})),
		Replies:           sequencedmap.New(sequencedmap.NewElem("r", &asyncapi.ReferencedOperationReply{})),
		ReplyAddresses:    sequencedmap.New(sequencedmap.NewElem("a", &asyncapi.ReferencedOperationReplyAddress{})),
		ExternalDocs:      sequencedmap.New(sequencedmap.NewElem("d", &asyncapi.ExternalDocumentation{})),
		Tags:              sequencedmap.New(sequencedmap.NewElem("t", &asyncapi.Tag{})),
		OperationTraits:   sequencedmap.New(sequencedmap.NewElem("ot", &asyncapi.ReferencedOperationTrait{})),
		MessageTraits:     sequencedmap.New(sequencedmap.NewElem("mt", &asyncapi.ReferencedMessageTrait{})),
		ServerBindings:    sequencedmap.New(sequencedmap.NewElem("sb", &asyncapi.ReferencedBindings{})),
		ChannelBindings:   sequencedmap.New(sequencedmap.NewElem("cb", &asyncapi.ReferencedBindings{})),
		OperationBindings: sequencedmap.New(sequencedmap.NewElem("ob", &asyncapi.ReferencedBindings{})),
		MessageBindings:   sequencedmap.New(sequencedmap.NewElem("mb", &asyncapi.ReferencedBindings{})),
		Extensions:        exts,
	}

	type getters struct {
		schemas           *sequencedmap.Map[string, *jsonschema.SchemaUnion]
		servers           *sequencedmap.Map[string, *asyncapi.ReferencedServer]
		channels          *sequencedmap.Map[string, *asyncapi.ReferencedChannel]
		operations        *sequencedmap.Map[string, *asyncapi.ReferencedOperation]
		messages          *sequencedmap.Map[string, *asyncapi.ReferencedMessage]
		securitySchemes   *sequencedmap.Map[string, *asyncapi.ReferencedSecurityScheme]
		serverVariables   *sequencedmap.Map[string, *asyncapi.ReferencedServerVariable]
		parameters        *sequencedmap.Map[string, *asyncapi.ReferencedParameter]
		correlationIDs    *sequencedmap.Map[string, *asyncapi.ReferencedCorrelationID]
		replies           *sequencedmap.Map[string, *asyncapi.ReferencedOperationReply]
		replyAddresses    *sequencedmap.Map[string, *asyncapi.ReferencedOperationReplyAddress]
		externalDocs      *sequencedmap.Map[string, *asyncapi.ExternalDocumentation]
		tags              *sequencedmap.Map[string, *asyncapi.Tag]
		operationTraits   *sequencedmap.Map[string, *asyncapi.ReferencedOperationTrait]
		messageTraits     *sequencedmap.Map[string, *asyncapi.ReferencedMessageTrait]
		serverBindings    *sequencedmap.Map[string, *asyncapi.ReferencedBindings]
		channelBindings   *sequencedmap.Map[string, *asyncapi.ReferencedBindings]
		operationBindings *sequencedmap.Map[string, *asyncapi.ReferencedBindings]
		messageBindings   *sequencedmap.Map[string, *asyncapi.ReferencedBindings]
		extensions        *extensions.Extensions
	}

	tests := []struct {
		name       string
		components *asyncapi.Components
		expected   getters
	}{
		{
			name:       "nil components return defaults",
			components: nil,
			expected:   getters{extensions: extensions.New()},
		},
		{
			name:       "unset fields return defaults",
			components: &asyncapi.Components{},
			expected:   getters{extensions: extensions.New()},
		},
		{
			name:       "populated fields are returned",
			components: populated,
			expected: getters{
				schemas:           populated.Schemas,
				servers:           populated.Servers,
				channels:          populated.Channels,
				operations:        populated.Operations,
				messages:          populated.Messages,
				securitySchemes:   populated.SecuritySchemes,
				serverVariables:   populated.ServerVariables,
				parameters:        populated.Parameters,
				correlationIDs:    populated.CorrelationIDs,
				replies:           populated.Replies,
				replyAddresses:    populated.ReplyAddresses,
				externalDocs:      populated.ExternalDocs,
				tags:              populated.Tags,
				operationTraits:   populated.OperationTraits,
				messageTraits:     populated.MessageTraits,
				serverBindings:    populated.ServerBindings,
				channelBindings:   populated.ChannelBindings,
				operationBindings: populated.OperationBindings,
				messageBindings:   populated.MessageBindings,
				extensions:        exts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := getters{
				schemas:           tt.components.GetSchemas(),
				servers:           tt.components.GetServers(),
				channels:          tt.components.GetChannels(),
				operations:        tt.components.GetOperations(),
				messages:          tt.components.GetMessages(),
				securitySchemes:   tt.components.GetSecuritySchemes(),
				serverVariables:   tt.components.GetServerVariables(),
				parameters:        tt.components.GetParameters(),
				correlationIDs:    tt.components.GetCorrelationIDs(),
				replies:           tt.components.GetReplies(),
				replyAddresses:    tt.components.GetReplyAddresses(),
				externalDocs:      tt.components.GetExternalDocs(),
				tags:              tt.components.GetTags(),
				operationTraits:   tt.components.GetOperationTraits(),
				messageTraits:     tt.components.GetMessageTraits(),
				serverBindings:    tt.components.GetServerBindings(),
				channelBindings:   tt.components.GetChannelBindings(),
				operationBindings: tt.components.GetOperationBindings(),
				messageBindings:   tt.components.GetMessageBindings(),
				extensions:        tt.components.GetExtensions(),
			}
			assert.Equal(t, tt.expected, actual, "getters should return expected values")
		})
	}
}
