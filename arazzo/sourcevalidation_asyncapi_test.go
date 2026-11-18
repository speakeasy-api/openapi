package arazzo_test

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArazzo_ValidateSources_AsyncAPI_Success(t *testing.T) {
	t.Parallel()

	assert.Empty(t, validateSourcesFixture(t, "testdata/sources/async-valid.arazzo.yaml"), "mixed OpenAPI and AsyncAPI workflow should be valid against its sources")
}

func TestArazzo_ValidateSources_AsyncAPI_Error(t *testing.T) {
	t.Parallel()

	expected := []string{
		"error: step.operationId `missing` not found in sourceDescription `orders`",
		"error: step.operationId `reusableSend` not found in sourceDescription `orders`",
		"error: step.channelPath `{$sourceDescriptions.orders.url}#/channels/orders.confirmed` doesn't reference a channel in sourceDescription `orders`",
		"error: step.channelPath `{$sourceDescriptions.orders.url}#/operations/placeOrder` doesn't reference a channel in sourceDescription `orders`",
		"error: step.operationPath can't reference AsyncAPI sourceDescription `orders`, use operationId or channelPath for AsyncAPI steps",
		"error: step.action is required for asyncapi steps, operationId references sourceDescription `untypedOrders` which is an AsyncAPI document",
		"warning: step.parameters `unknownHeader` in `header` is not a header of any message of operation `placeOrder` in sourceDescription `orders`",
		"warning: step.parameters `orderId` in `path` has no meaning for AsyncAPI operation `placeOrder` in sourceDescription `orders`, only `header` parameters are sent with messages",
		"warning: step.requestBody is not sent by a step with action `receive` targeting operation `confirmOrder` in sourceDescription `orders`",
		"warning: step.correlationId is set but no message of channel `/components/channels/orderStatus` in sourceDescription `orders` declares a correlationId",
		"warning: step.parameters `requestId` in `header` is not a header of any message of operation `noMessages` in sourceDescription `orders`",
		"error: step.action is only applicable to asyncapi steps, sourceDescription `untypedPetstore` is an openapi document",
	}

	assert.Equal(t, expected, validateSourcesFixture(t, "testdata/sources/async-invalid.arazzo.yaml"), "source validation errors should match")
}

const asyncAPIPartialSource = `asyncapi: 3.1.0
info:
  title: Partial
  version: 1.0.0
channels:
  events:
    address: events
    messages:
      known:
        payload:
          type: object
      missing:
        $ref: "./missing.yaml#/components/messages/Missing"
operations:
  listen:
    action: receive
    channel:
      $ref: "#/channels/events"
  unresolvedOperation:
    $ref: "./missing.yaml#/operations/listen"
`

const asyncAPIEdgeCasesSource = `asyncapi: 3.1.0
info:
  title: Edge cases
  version: 1.0.0
channels:
  refHeaders:
    address: ref
    messages:
      withRefHeaders:
        headers:
          $ref: "#/components/schemas/Headers"
  multiFormatHeaders:
    address: multi
    messages:
      withMultiFormatHeaders:
        headers:
          schemaFormat: application/schema+json;version=draft-07
          schema:
            type: object
            properties:
              traceId:
                type: string
  unresolvedChannel:
    $ref: "./missing.yaml#/channels/unresolved"
operations:
  danglingChannel:
    action: send
    channel:
      $ref: "./missing.yaml#/channels/unresolved"
components:
  schemas:
    Headers:
      type: object
      properties:
        traceId:
          type: string
`

const asyncAPIEdgeCasesOpenAPI = `openapi: 3.1.0
info:
  title: Petstore
  version: 1.0.0
paths: {}
`

func TestArazzo_ValidateSources_AsyncAPIEdgeCases_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		step     string
		expected []string
	}{
		{
			name:     "parameter without a location is skipped",
			step:     "channelPath: \"{$sourceDescriptions.events.url}#/channels/refHeaders\"\n        parameters:\n          - name: anything\n            value: abc",
			expected: []string{},
		},
		{
			name:     "malformed channelPath is left to structural validation",
			step:     "channelPath: not-a-path",
			expected: []string{},
		},
		{
			name:     "channelPath to an unknown source is left to structural validation",
			step:     "channelPath: \"{$sourceDescriptions.unknown.url}#/channels/refHeaders\"",
			expected: []string{},
		},
		{
			name:     "channelPath to a source declared as another type is left to Validate",
			step:     "channelPath: \"{$sourceDescriptions.petstore.url}#/channels/refHeaders\"",
			expected: []string{},
		},
		{
			name:     "channelPath pointer without a leading slash",
			step:     "channelPath: \"{$sourceDescriptions.events.url}#channels/refHeaders\"",
			expected: []string{"error: step.channelPath `{$sourceDescriptions.events.url}#channels/refHeaders` doesn't reference a channel in sourceDescription `events`"},
		},
		{
			name:     "unresolved channel skips undeclared header checks",
			step:     "channelPath: \"{$sourceDescriptions.events.url}#/channels/unresolvedChannel\"\n        parameters:\n          - name: undeclared\n            in: header\n            value: abc",
			expected: []string{},
		},
		{
			name:     "operation with an unresolved channel skips undeclared header checks",
			step:     "operationId: $sourceDescriptions.events.danglingChannel\n        parameters:\n          - name: undeclared\n            in: header\n            value: abc",
			expected: []string{},
		},
		{
			name:     "headers schema from a reference",
			step:     "channelPath: \"{$sourceDescriptions.events.url}#/channels/refHeaders\"\n        parameters:\n          - name: TraceId\n            in: header\n            value: abc\n          - name: undeclared\n            in: header\n            value: abc",
			expected: []string{"warning: step.parameters `undeclared` in `header` is not a header of any message of channel `/channels/refHeaders` in sourceDescription `events`"},
		},
		{
			name:     "headers schema in a multi-format schema",
			step:     "channelPath: \"{$sourceDescriptions.events.url}#/channels/multiFormatHeaders\"\n        parameters:\n          - name: traceId\n            in: header\n            value: abc\n          - name: undeclared\n            in: header\n            value: abc",
			expected: []string{"warning: step.parameters `undeclared` in `header` is not a header of any message of channel `/channels/multiFormatHeaders` in sourceDescription `events`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc := `arazzo: 1.1.0
info:
  title: Edge cases
  version: 1.0.0
sourceDescriptions:
  - name: events
    url: events.yaml
    type: asyncapi
  - name: petstore
    url: petstore.yaml
    type: openapi
workflows:
  - workflowId: main
    steps:
      - stepId: step
        action: send
        ` + tt.step + `
`
			a, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(doc), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{
				VirtualFS: fstest.MapFS{
					"events.yaml":   {Data: []byte(asyncAPIEdgeCasesSource)},
					"petstore.yaml": {Data: []byte(asyncAPIEdgeCasesOpenAPI)},
				},
			})
			require.Len(t, loadErrs, 1, "should warn that the AsyncAPI source has unresolved references")
			assert.Contains(t, severityMessages(loadErrs)[0], "warning: sourceDescription `events` has references that could not be resolved, some checks may be skipped", "load warning should match")
			require.NotNil(t, sources.Get("events"), "AsyncAPI source should be loaded")

			assert.Equal(t, tt.expected, severityMessages(a.ValidateSources(ctx, sources)), "source validation messages should match")
		})
	}
}

func TestArazzo_ValidateSources_AsyncAPIPartialTarget_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
	}{
		{name: "operation with an unresolved message", target: "operationId: $sourceDescriptions.events.listen"},
		{name: "unresolved operation", target: "operationId: $sourceDescriptions.events.unresolvedOperation"},
		{name: "channel with an unresolved message", target: "channelPath: \"{$sourceDescriptions.events.url}#/channels/events\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			doc := `arazzo: 1.1.0
info:
  title: Partial
  version: 1.0.0
sourceDescriptions:
  - name: events
    url: events.yaml
    type: asyncapi
workflows:
  - workflowId: main
    steps:
      - stepId: listen
        ` + tt.target + `
        action: receive
        correlationId: abc
        parameters:
          - name: anything
            in: header
            value: abc
`
			a, _, err := arazzo.Unmarshal(ctx, bytes.NewBufferString(doc), arazzo.WithSkipValidation())
			require.NoError(t, err, "unmarshal should succeed")

			sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, a, arazzo.LoadSourcesOptions{
				VirtualFS: fstest.MapFS{"events.yaml": {Data: []byte(asyncAPIPartialSource)}},
			})
			require.Len(t, loadErrs, 1, "should warn about the unresolved references")
			require.NotNil(t, sources.Get("events"), "source should still be loaded")

			assert.Empty(t, a.ValidateSources(ctx, sources), "undeclared headers and correlation IDs shouldn't be reported for partial targets")
		})
	}
}
