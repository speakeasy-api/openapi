package asyncapi_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// walkSupplementDoc contains the object kinds and nil list entries that complete-3.1.yaml does not: a tag with
// external docs, an operation trait with every field, a message trait with every field, and null tags, reference
// objects and examples.
const walkSupplementDoc = `asyncapi: 3.1.0
info:
  title: T
  version: '1'
channels:
  c:
    servers:
      - null
    tags:
      - null
      - name: documented
        externalDocs:
          url: https://example.com/tags
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    traits:
      - security:
          - type: userPassword
        tags:
          - name: t
        externalDocs:
          url: https://example.com
        bindings:
          kafka: {}
        reply:
          channel:
            $ref: '#/channels/c'
components:
  messageTraits:
    mt:
      correlationId:
        location: $message.payload#/id
      tags:
        - name: t
      externalDocs:
        url: https://example.com
      bindings:
        kafka: {}
      examples:
        - null
        - name: e
`

// walkItemID identifies a walk item by its model type, without the module path, and its location.
func walkItemID(t *testing.T, item asyncapi.WalkItem) string {
	t.Helper()

	typ := ""
	require.NoError(t, item.Match(asyncapi.Matcher{Any: func(model any) error {
		typ = strings.ReplaceAll(fmt.Sprintf("%T", model), "github.com/speakeasy-api/openapi/", "")
		return nil
	}}), "match should succeed")
	return typ + " " + item.Location.ToJSONPointer().String()
}

func walkPointers(t *testing.T, a *asyncapi.AsyncAPI, matcher func(pointers *[]string) asyncapi.Matcher) []string {
	t.Helper()

	pointers := []string{}
	m := matcher(&pointers)
	for item := range asyncapi.Walk(t.Context(), a) {
		before := len(pointers)
		require.NoError(t, item.Match(m), "match should succeed")
		for i := before; i < len(pointers); i++ {
			pointers[i] = string(item.Location.ToJSONPointer())
		}
	}
	return pointers
}

func TestWalk_MissingChannel_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("operations:\n  o:\n    action: send\n"))
	pointers := walkPointers(t, a, func(p *[]string) asyncapi.Matcher {
		return asyncapi.Matcher{ReferenceObject: func(*asyncapi.ReferenceObject) error { *p = append(*p, ""); return nil }}
	})
	assert.Empty(t, pointers, "an omitted channel should not be visited")
}

func TestWalk_Locations_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))

	tests := []struct {
		name     string
		matcher  func(pointers *[]string) asyncapi.Matcher
		expected []string
	}{
		{
			name: "operations",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{Operation: func(*asyncapi.Operation) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{"/operations/placeOrder", "/operations/receiveNothing", "/components/operations/shared"},
		},
		{
			name: "referenced operations",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{ReferencedOperation: func(*asyncapi.ReferencedOperation) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{"/operations/placeOrder", "/operations/receiveNothing", "/operations/shared", "/components/operations/shared"},
		},
		{
			name: "messages",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{Message: func(*asyncapi.Message) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{"/channels/orderPlaced/messages/orderPlacedInline", "/components/messages/orderPlaced", "/components/messages/orderConfirmed"},
		},
		{
			name: "server variables",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{ServerVariable: func(*asyncapi.ServerVariable) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{"/servers/production/variables/region", "/components/serverVariables/port"},
		},
		{
			name: "reference objects",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{ReferenceObject: func(*asyncapi.ReferenceObject) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{
				"/channels/orderPlaced/servers/0",
				"/operations/placeOrder/channel",
				"/operations/placeOrder/messages/0",
				"/operations/placeOrder/reply/channel",
				"/operations/placeOrder/reply/messages/0",
				"/operations/receiveNothing/channel",
				"/components/operations/shared/channel",
				"/components/replies/confirmation/channel",
			},
		},
		{
			name: "schemas",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{SchemaUnion: func(*jsonschema.SchemaUnion) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{
				"/channels/orderPlaced/messages/orderPlacedInline/payload",
				"/components/schemas/Order",
				"/components/schemas/AvroOrder",
				"/components/messages/orderPlaced/headers",
				"/components/messages/orderPlaced/payload",
				"/components/messages/orderConfirmed/payload",
				"/components/messageTraits/common/headers",
			},
		},
		{
			name: "oauth flows",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{OAuthFlow: func(*asyncapi.OAuthFlow) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{
				"/servers/production/security/1/flows/clientCredentials",
				"/components/securitySchemes/oauth/flows/implicit",
				"/components/securitySchemes/oauth/flows/password",
				"/components/securitySchemes/oauth/flows/authorizationCode",
			},
		},
		{
			name: "bindings",
			matcher: func(p *[]string) asyncapi.Matcher {
				return asyncapi.Matcher{Bindings: func(*asyncapi.Bindings) error { *p = append(*p, ""); return nil }}
			},
			expected: []string{
				"/servers/production/bindings",
				"/channels/orderPlaced/bindings",
				"/channels/robotCommands/bindings",
				"/operations/placeOrder/bindings",
				"/components/messages/orderPlaced/bindings",
				"/components/operationTraits/kafka/bindings",
				"/components/serverBindings/kafka",
				"/components/channelBindings/kafka",
				"/components/operationBindings/kafka",
				"/components/messageBindings/kafka",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, walkPointers(t, a, tt.matcher), "walk should visit locations in document order")
		})
	}
}

func TestWalk_VisitsEachLocationOnce_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))

	seen := map[string]int{}
	count := 0
	for item := range asyncapi.Walk(t.Context(), a) {
		require.NoError(t, item.Match(asyncapi.Matcher{Any: func(model any) error {
			seen[fmt.Sprintf("%T %s", model, item.Location.ToJSONPointer())]++
			return nil
		}}), "match should succeed")
		count++
	}

	assert.Len(t, seen, count, "each model should be visited once per authored location")
	assert.Equal(t, 1, seen["*asyncapi.AsyncAPI /"], "root should be visited")
	assert.Equal(t, 1, seen["*asyncapi.Info /info"], "info should be visited")
	assert.Equal(t, 1, seen["*asyncapi.Contact /info/contact"], "contact should be visited")
	assert.Equal(t, 1, seen["*asyncapi.Tag /components/tags/events"], "component tags should be visited")
	assert.Equal(t, 1, seen["*asyncapi.MessageExample /components/messages/orderPlaced/examples/0"], "message examples should be visited")
	assert.Equal(t, 1, seen["*asyncapi.OperationReplyAddress /operations/placeOrder/reply/address"], "reply addresses should be visited")
	assert.Equal(t, 1, seen["*asyncapi.CorrelationID /components/correlationIds/orderId"], "correlation IDs should be visited")
}

func TestWalk_EarlyTermination_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))

	count := 0
	for range asyncapi.Walk(t.Context(), a) {
		count++
		if count == 3 {
			break
		}
	}
	assert.Equal(t, 3, count, "walking should stop when the consumer breaks")
}

func TestWalk_Cancelled_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	count := 0
	for range asyncapi.Walk(ctx, a) {
		count++
	}
	assert.Zero(t, count, "walking a cancelled context should yield nothing")
}

func TestWalk_StopAtEachNode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
	}{
		{name: "complete document", doc: readTestdata(t, "complete-3.1.yaml")},
		{name: "supplementary object kinds", doc: walkSupplementDoc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, _ := unmarshal(t, tt.doc)
			locations := []string{}
			for item := range asyncapi.Walk(t.Context(), a) {
				locations = append(locations, walkItemID(t, item))
			}
			require.NotEmpty(t, locations, "document should contain walk items")

			for stopAt, location := range locations {
				t.Run(strconv.Itoa(stopAt)+":"+location, func(t *testing.T) {
					t.Parallel()

					visited := []string{}
					asyncapi.Walk(t.Context(), a)(func(item asyncapi.WalkItem) bool {
						visited = append(visited, walkItemID(t, item))
						return len(visited) <= stopAt
					})
					assert.Equal(t, locations[:stopAt+1], visited, "a false yield should stop every ancestor without visiting later items")
				})
			}
		})
	}
}

func TestWalk_SupplementLocations_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, walkSupplementDoc)
	locations := []string{}
	for item := range asyncapi.Walk(t.Context(), a) {
		locations = append(locations, walkItemID(t, item))
	}

	assert.Equal(t, []string{
		"*asyncapi.AsyncAPI /",
		"*asyncapi.Info /info",
		"*extensions.Extensions /info",
		"*asyncapi.Reference[asyncapi.Channel,*asyncapi.Channel,*asyncapi/core.Channel] /channels/c",
		"*asyncapi.Channel /channels/c",
		"*asyncapi.Tag /channels/c/tags/1",
		"*asyncapi.ExternalDocumentation /channels/c/tags/1/externalDocs",
		"*extensions.Extensions /channels/c/tags/1/externalDocs",
		"*extensions.Extensions /channels/c/tags/1",
		"*extensions.Extensions /channels/c",
		"*asyncapi.Reference[asyncapi.Operation,*asyncapi.Operation,*asyncapi/core.Operation] /operations/o",
		"*asyncapi.Operation /operations/o",
		"*asyncapi.ReferenceObject /operations/o/channel",
		"*asyncapi.Reference[asyncapi.OperationTrait,*asyncapi.OperationTrait,*asyncapi/core.OperationTrait] /operations/o/traits/0",
		"*asyncapi.OperationTrait /operations/o/traits/0",
		"*asyncapi.Reference[asyncapi.SecurityScheme,*asyncapi.SecurityScheme,*asyncapi/core.SecurityScheme] /operations/o/traits/0/security/0",
		"*asyncapi.SecurityScheme /operations/o/traits/0/security/0",
		"*extensions.Extensions /operations/o/traits/0/security/0",
		"*asyncapi.Tag /operations/o/traits/0/tags/0",
		"*extensions.Extensions /operations/o/traits/0/tags/0",
		"*asyncapi.ExternalDocumentation /operations/o/traits/0/externalDocs",
		"*extensions.Extensions /operations/o/traits/0/externalDocs",
		"*asyncapi.Reference[asyncapi.Bindings,*asyncapi.Bindings,*asyncapi/core.Bindings] /operations/o/traits/0/bindings",
		"*asyncapi.Bindings /operations/o/traits/0/bindings",
		"*extensions.Extensions /operations/o/traits/0/bindings",
		"*asyncapi.Reference[asyncapi.OperationReply,*asyncapi.OperationReply,*asyncapi/core.OperationReply] /operations/o/traits/0/reply",
		"*asyncapi.OperationReply /operations/o/traits/0/reply",
		"*asyncapi.ReferenceObject /operations/o/traits/0/reply/channel",
		"*extensions.Extensions /operations/o/traits/0/reply",
		"*extensions.Extensions /operations/o/traits/0",
		"*extensions.Extensions /operations/o",
		"*asyncapi.Components /components",
		"*asyncapi.Reference[asyncapi.MessageTrait,*asyncapi.MessageTrait,*asyncapi/core.MessageTrait] /components/messageTraits/mt",
		"*asyncapi.MessageTrait /components/messageTraits/mt",
		"*asyncapi.Reference[asyncapi.CorrelationID,*asyncapi.CorrelationID,*asyncapi/core.CorrelationID] /components/messageTraits/mt/correlationId",
		"*asyncapi.CorrelationID /components/messageTraits/mt/correlationId",
		"*extensions.Extensions /components/messageTraits/mt/correlationId",
		"*asyncapi.Tag /components/messageTraits/mt/tags/0",
		"*extensions.Extensions /components/messageTraits/mt/tags/0",
		"*asyncapi.ExternalDocumentation /components/messageTraits/mt/externalDocs",
		"*extensions.Extensions /components/messageTraits/mt/externalDocs",
		"*asyncapi.Reference[asyncapi.Bindings,*asyncapi.Bindings,*asyncapi/core.Bindings] /components/messageTraits/mt/bindings",
		"*asyncapi.Bindings /components/messageTraits/mt/bindings",
		"*extensions.Extensions /components/messageTraits/mt/bindings",
		"*asyncapi.MessageExample /components/messageTraits/mt/examples/1",
		"*extensions.Extensions /components/messageTraits/mt/examples/1",
		"*extensions.Extensions /components/messageTraits/mt",
		"*extensions.Extensions /components",
		"*extensions.Extensions /",
	}, locations, "walk should visit supplementary items in document order and skip null entries")
}

func TestWalk_NilContext_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, readTestdata(t, "complete-3.1.yaml"))

	expected := 0
	for range asyncapi.Walk(t.Context(), a) {
		expected++
	}

	var ctx context.Context
	count := 0
	for range asyncapi.Walk(ctx, a) {
		count++
	}
	assert.Equal(t, expected, count, "walking without a context should visit every item")
}

func TestWalk_MatchAnyError_Error(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc(""))
	errAny := errors.New("any failed")

	for item := range asyncapi.Walk(t.Context(), a) {
		infoCalled := false
		err := item.Match(asyncapi.Matcher{
			Any:  func(any) error { return errAny },
			Info: func(*asyncapi.Info) error { infoCalled = true; return nil },
		})
		require.ErrorIs(t, err, errAny, "match should return the Any error")
		assert.False(t, infoCalled, "specific matchers should not be called after Any fails")
	}
}

func TestWalk_Nil_Success(t *testing.T) {
	t.Parallel()

	count := 0
	for range asyncapi.Walk(t.Context(), nil) {
		count++
	}
	assert.Zero(t, count, "walking a nil document should yield nothing")
}
