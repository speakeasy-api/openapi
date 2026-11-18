package asyncapi_test

import (
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal_Complete_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))
	assert.Empty(t, errorsOnly(validationErrs), "complete document should have no errors")
	assert.True(t, a.Valid, "complete document should be valid")

	servers := a.GetServers()
	require.Equal(t, 2, servers.Len(), "servers should be parsed")
	production := servers.GetOrZero("production").GetObject()
	require.NotNil(t, production, "inline server should be parsed")
	assert.Equal(t, "{region}.broker.example.com:9092", production.GetHost(), "server host should be parsed")
	assert.Equal(t, "kafka", production.GetProtocol(), "server protocol should be parsed")
	assert.Equal(t, "/v1", production.GetPathname(), "server pathname should be parsed")
	assert.Equal(t, []string{"eu", "us"}, production.GetVariables().GetOrZero("region").GetObject().GetEnum(), "server variable enum should be parsed")
	require.Len(t, production.GetSecurity(), 2, "server security should be parsed")
	assert.True(t, production.GetSecurity()[0].IsReference(), "security reference should be parsed")
	oauth := production.GetSecurity()[1].GetObject()
	assert.Equal(t, asyncapi.SecuritySchemeTypeOAuth2, oauth.GetType(), "inline security scheme should be parsed")
	assert.Equal(t, "https://auth.example.com/token", oauth.GetFlows().GetClientCredentials().GetTokenURL(), "oauth flow should be parsed")
	assert.Equal(t, "Write orders", oauth.GetFlows().GetClientCredentials().GetAvailableScopes().GetOrZero("orders:write"), "oauth scopes should be parsed")
	assert.Equal(t, "0.5.0", production.GetBindings().GetObject().GetBindingVersion("kafka"), "binding version should be parsed")
	assert.Equal(t, "#/components/servers/staging", string(servers.GetOrZero("staging").GetReference()), "server reference should be parsed")

	channel := a.GetChannels().GetOrZero("orderPlaced").GetObject()
	require.NotNil(t, channel, "channel should be parsed")
	assert.Equal(t, "orders.{orderId}.placed", channel.GetAddress(), "channel address should be parsed")
	assert.True(t, channel.GetMessages().GetOrZero("orderPlaced").IsReference(), "channel message reference should be parsed")
	assert.Equal(t, "OrderPlacedInline", channel.GetMessages().GetOrZero("orderPlacedInline").GetObject().GetName(), "inline channel message should be parsed")
	assert.Equal(t, "#/servers/production", string(channel.GetServers()[0].GetReference()), "channel server reference should be parsed")

	replies := a.GetChannels().GetOrZero("orderReplies").GetObject()
	assert.Nil(t, replies.Address, "null channel address should be unknown")

	operation := a.GetOperations().GetOrZero("placeOrder").GetObject()
	require.NotNil(t, operation, "operation should be parsed")
	assert.Equal(t, asyncapi.OperationActionSend, operation.GetAction(), "operation action should be parsed")
	assert.Equal(t, "#/channels/orderPlaced", string(operation.GetChannel().GetReference()), "operation channel should be parsed")
	require.Len(t, operation.GetTraits(), 2, "operation traits should be parsed")
	assert.Equal(t, "Inline trait", operation.GetTraits()[1].GetObject().GetTitle(), "inline operation trait should be parsed")
	reply := operation.GetReply().GetObject()
	assert.Equal(t, "$message.header#/replyTo", reply.GetAddress().GetObject().GetLocation(), "reply address should be parsed")
	assert.Equal(t, "#/channels/orderReplies", string(reply.GetChannel().GetReference()), "reply channel should be parsed")

	components := a.GetComponents()
	require.NotNil(t, components, "components should be parsed")
	assert.True(t, components.GetSchemas().GetOrZero("Order").IsSchema(), "native component schema should be parsed")
	assert.True(t, components.GetSchemas().GetOrZero("AvroOrder").IsMultiFormat(), "multi format component schema should be parsed")
	message := components.GetMessages().GetOrZero("orderPlaced").GetObject()
	assert.True(t, message.GetPayload().IsReference(), "message payload reference should be parsed")
	assert.Equal(t, []string{"object"}, message.GetHeaders().GetSchema().GetSchema().GetType(), "message headers should be parsed")
	assert.True(t, message.GetCorrelationID().IsReference(), "message correlation ID reference should be parsed")
	require.Len(t, message.GetExamples(), 1, "message examples should be parsed")
	assert.Equal(t, "simple", message.GetExamples()[0].GetName(), "message example should be parsed")
	assert.True(t, components.GetMessages().GetOrZero("orderConfirmed").GetObject().GetPayload().IsMultiFormat(), "draft 07 payload should be parsed")
	assert.Equal(t, "$message.payload#/id", components.GetCorrelationIDs().GetOrZero("orderId").GetObject().GetLocation(), "correlation ID should be parsed")
	assert.Equal(t, "events", components.GetTags().GetOrZero("events").GetName(), "component tag should be parsed")
	assert.Equal(t, "read", firstKey(components.GetSecuritySchemes().GetOrZero("oauth").GetObject().GetFlows().GetAuthorizationCode().GetAvailableScopes().Keys()), "authorization code scopes should be parsed")
	assert.Equal(t, 1, components.GetMessageBindings().Len(), "component message bindings should be parsed")

	assert.Equal(t, "platform", extensionValue(t, a.GetExtensions(), "x-owner"), "root extensions should be parsed")
	assert.Equal(t, "primary", extensionValue(t, production.GetExtensions(), "x-cluster"), "server extensions should be parsed")
}

func TestUnmarshal_Streetlights30_Success(t *testing.T) {
	t.Parallel()

	a, validationErrs := unmarshal(t, readTestdata(t, "streetlights-3.0.yaml"))
	assert.Empty(t, errorsOnly(validationErrs), "3.0 document should have no errors")
	assert.True(t, a.Valid, "3.0 document should be valid")
	assert.Equal(t, "3.0.0", a.GetAsyncAPI(), "version should be parsed")
	assert.Equal(t, asyncapi.OperationActionReceive, a.GetOperations().GetOrZero("receiveLightMeasurement").GetObject().GetAction(), "operation should be parsed")
}

func TestMarshal_Complete_RoundTrip_Success(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"complete-3.1.yaml", "streetlights-3.0.yaml"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			doc := readTestdata(t, name)
			a, _ := unmarshal(t, doc)
			assert.Equal(t, doc, marshal(t, a), "document should round-trip unchanged")
		})
	}
}

func TestMarshal_Complete_JSONRoundTrip_Success(t *testing.T) {
	t.Parallel()

	yamlDoc, _ := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))
	yamlDoc.GetCore().Config.OutputFormat = yml.OutputFormatJSON
	jsonDoc := marshal(t, yamlDoc)
	require.True(t, strings.HasPrefix(jsonDoc, "{"), "fixture should convert to JSON")

	a, validationErrs := unmarshal(t, jsonDoc)
	assert.Empty(t, errorsOnly(validationErrs), "JSON document should have no errors")
	assert.Equal(t, jsonDoc, marshal(t, a), "JSON document should round-trip unchanged")
}

func TestMarshal_NullAddressAndEmptyMessages_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`channels:
  replies:
    address: null
  plain: {}
operations:
  none:
    action: receive
    channel:
      $ref: '#/channels/replies'
    messages: []
  all:
    action: receive
    channel:
      $ref: '#/channels/plain'
`)
	a, validationErrs := unmarshal(t, doc)
	assert.Empty(t, errorsOnly(validationErrs), "document should be valid")

	assert.Nil(t, a.GetChannels().GetOrZero("replies").GetObject().Address, "null address should be unknown")
	assert.Nil(t, a.GetChannels().GetOrZero("plain").GetObject().Address, "missing address should be unknown")

	none := a.GetOperations().GetOrZero("none").GetObject().GetMessages()
	assert.NotNil(t, none, "empty messages should be distinguishable from omitted messages")
	assert.Empty(t, none, "empty messages should select no messages")
	assert.Nil(t, a.GetOperations().GetOrZero("all").GetObject().GetMessages(), "omitted messages should be nil")

	a.GetOperations().GetOrZero("all").GetObject().Summary = pointerTo("changed")
	out := marshal(t, a)
	assert.Contains(t, out, "    address: null\n", "null address should be preserved after mutation")
	assert.Contains(t, out, "    messages: []\n", "empty messages should be preserved after mutation")
	assert.Contains(t, out, "    summary: changed\n", "mutation should be written")
}

func TestMarshal_Complete_Mutate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(a *asyncapi.AsyncAPI)
		original string
		expected string
	}{
		{
			name:     "server host",
			mutate:   func(a *asyncapi.AsyncAPI) { a.Servers.GetOrZero("production").Object.Host = "broker.example.com" },
			original: "    host: \"{region}.broker.example.com:9092\"\n",
			// The original double-quoted scalar style is kept.
			expected: "    host: \"broker.example.com\"\n",
		},
		{
			name: "server variable default",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Servers.GetOrZero("production").Object.Variables.GetOrZero("region").Object.Default = pointerTo("us")
			},
			original: "        default: eu\n",
			expected: "        default: us\n",
		},
		{
			name: "channel address",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Channels.GetOrZero("robotCommands").Object.Address = pointerTo("robot/cmd")
			},
			original: "    address: robot/commands\n",
			expected: "    address: robot/cmd\n",
		},
		{
			name: "channel parameter",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.Parameters.GetOrZero("orderId").Object.Location = pointerTo("$message.header#/orderId")
			},
			original: "      location: $message.payload#/id\n  correlationIds:\n",
			expected: "      location: $message.header#/orderId\n  correlationIds:\n",
		},
		{
			name: "operation action",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Operations.GetOrZero("receiveNothing").Object.Action = asyncapi.OperationActionSend
			},
			original: "  receiveNothing:\n    action: receive\n",
			expected: "  receiveNothing:\n    action: send\n",
		},
		{
			name: "operation channel reference",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Operations.GetOrZero("receiveNothing").Object.Channel.Reference = "#/channels/robotCommands"
			},
			original: "    channel:\n      $ref: \"#/channels/orderReplies\"\n    messages: []\n",
			expected: "    channel:\n      $ref: \"#/channels/robotCommands\"\n    messages: []\n",
		},
		{
			name: "operation reply address",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Operations.GetOrZero("placeOrder").Object.Reply.Object.Address.Object.Location = "$message.header#/reply"
			},
			original: "        location: $message.header#/replyTo\n      channel:\n",
			expected: "        location: $message.header#/reply\n      channel:\n",
		},
		{
			name: "operation trait",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Operations.GetOrZero("placeOrder").Object.Traits[1].Object.Title = pointerTo("Renamed trait")
			},
			original: "      - title: Inline trait\n",
			expected: "      - title: Renamed trait\n",
		},
		{
			name: "message name",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.Messages.GetOrZero("orderPlaced").Object.Name = pointerTo("OrderCreated")
			},
			original: "      name: OrderPlaced\n",
			expected: "      name: OrderCreated\n",
		},
		{
			name: "message example name",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.Messages.GetOrZero("orderPlaced").Object.Examples[0].Name = pointerTo("basic")
			},
			original: "        - name: simple\n",
			expected: "        - name: basic\n",
		},
		{
			name: "message trait content type",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.MessageTraits.GetOrZero("common").Object.ContentType = pointerTo("application/avro")
			},
			original: "        type: object\n      contentType: application/json\n  serverBindings:\n",
			expected: "        type: object\n      contentType: application/avro\n  serverBindings:\n",
		},
		{
			name: "correlation ID",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.CorrelationIDs.GetOrZero("orderId").Object.Description = pointerTo("By order")
			},
			original: "      description: Correlates by order ID.\n",
			expected: "      description: By order\n",
		},
		{
			name: "security scheme",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.SecuritySchemes.GetOrZero("bearer").Object.BearerFormat = pointerTo("opaque")
			},
			original: "      bearerFormat: JWT\n",
			expected: "      bearerFormat: opaque\n",
		},
		{
			name: "oauth scope",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Components.SecuritySchemes.GetOrZero("oauth").Object.Flows.AuthorizationCode.AvailableScopes.Set("write", "Write access")
			},
			original: "            read: Read access\n",
			expected: "            read: Read access\n            write: Write access\n",
		},
		{
			name:     "component tag",
			mutate:   func(a *asyncapi.AsyncAPI) { a.Components.Tags.GetOrZero("events").Description = pointerTo("Events.") },
			original: "      description: Event tags.\n",
			expected: "      description: Events.\n",
		},
		{
			name: "reference replaced by inline object",
			mutate: func(a *asyncapi.AsyncAPI) {
				a.Servers.Set("staging", &asyncapi.ReferencedServer{Object: &asyncapi.Server{Host: "staging.example.com", Protocol: "kafka"}})
			},
			original: "  staging:\n    $ref: \"#/components/servers/staging\"\n",
			expected: "  staging:\n    host: staging.example.com\n    protocol: kafka\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := readTestdata(t, "complete-3.1.yaml")
			require.Contains(t, doc, tt.original, "fixture should contain the original value")

			a, _ := unmarshal(t, doc)
			tt.mutate(a)
			out := marshal(t, a)
			assert.Equal(t, strings.Replace(doc, tt.original, tt.expected, 1), out, "only the mutated value should change")

			_, validationErrs := unmarshal(t, out)
			assert.Empty(t, errorsOnly(validationErrs), "mutated document should still be valid")
		})
	}
}

func TestMarshal_NewDocument_Complete_Success(t *testing.T) {
	t.Parallel()

	a := asyncapi.NewAsyncAPI()
	a.Info = asyncapi.Info{Title: "Orders", Version: "1.0.0"}
	a.Channels = sequencedmap.New(sequencedmap.NewElem("orders", &asyncapi.ReferencedChannel{Object: &asyncapi.Channel{
		Address: pointerTo("orders"),
		Messages: sequencedmap.New(sequencedmap.NewElem("order", &asyncapi.ReferencedMessage{Object: &asyncapi.Message{
			ContentType: pointerTo("application/json"),
		}})),
	}}))
	a.Operations = sequencedmap.New(sequencedmap.NewElem("sendOrder", &asyncapi.ReferencedOperation{Object: &asyncapi.Operation{
		Action:   asyncapi.OperationActionSend,
		Channel:  *asyncapi.NewReferenceObject("#/channels/orders"),
		Messages: []*asyncapi.ReferenceObject{asyncapi.NewReferenceObject("#/channels/orders/messages/order")},
	}}))

	assert.Empty(t, a.Validate(t.Context()), "document built in code should be valid")
	assert.True(t, a.Valid, "document built in code should be valid")

	expected := `asyncapi: 3.1.0
info:
  title: Orders
  version: 1.0.0
channels:
  orders:
    address: orders
    messages:
      order:
        contentType: application/json
operations:
  sendOrder:
    action: send
    channel:
      $ref: '#/channels/orders'
    messages:
      - $ref: '#/channels/orders/messages/order'
`
	out := marshal(t, a)
	assert.Equal(t, expected, out, "document built in code should marshal")

	_, validationErrs := unmarshal(t, out)
	assert.Empty(t, validationErrs, "marshalled document should be valid")
}

func TestUnmarshal_UnknownProperties_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`channels:
  orders:
    address: orders
    addr: typo
operations:
  send:
    action: send
    channel:
      $ref: '#/channels/orders'
      ignored: sibling
x-ok: true
`)
	a, validationErrs := unmarshal(t, doc)
	assert.Equal(t, []string{"warning:" + validation.RuleValidationUnknownProperties}, diagnostics(validationErrs), "unknown channel property should be a warning, reference siblings are ignored")
	assert.True(t, a.Valid, "warnings should not make the document invalid")
	assert.Equal(t, doc, marshal(t, a), "unknown properties should be preserved")
}

func TestUnmarshal_BindingVersionGate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected []string
	}{
		{name: "ros2 is known in 3.1", version: "3.1.0", expected: []string{"hint:" + validation.RuleValidationUnsupportedFormat}},
		{name: "ros2 is unknown in 3.0", version: "3.0.0", expected: []string{"warning:" + validation.RuleValidationUnknownProperties}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := "asyncapi: " + tt.version + "\ninfo:\n  title: T\n  version: '1'\nchannels:\n  robot:\n    bindings:\n      ros2:\n        role: subscriber\n"
			a, validationErrs := unmarshal(t, doc)
			assert.Equal(t, tt.expected, diagnostics(validationErrs), "binding diagnostics should depend on the document version")
			assert.True(t, a.Valid, "binding diagnostics should not make the document invalid")
		})
	}
}

func extensionValue(t *testing.T, ext *extensions.Extensions, key string) string {
	t.Helper()

	node := ext.GetOrZero(key)
	require.NotNil(t, node, "extension should exist")
	return node.Value
}

func firstKey(keys func(func(string) bool)) string {
	for k := range keys {
		return k
	}
	return ""
}

func pointerTo[T any](v T) *T {
	return &v
}
