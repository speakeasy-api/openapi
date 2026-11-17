package asyncapi_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/speakeasy-api/openapi/asyncapi"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestWalk_Nil_Success(t *testing.T) {
	t.Parallel()

	count := 0
	for range asyncapi.Walk(t.Context(), nil) {
		count++
	}
	assert.Zero(t, count, "walking a nil document should yield nothing")
}
