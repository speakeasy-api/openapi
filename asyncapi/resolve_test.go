package asyncapi_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/asyncapi"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// countingFS counts how many files are opened.
type countingFS struct {
	files  fstest.MapFS
	opened atomic.Int32
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.opened.Add(1)
	return c.files.Open(name)
}

// fakeClient serves documents from memory and counts requests, without any network access.
type fakeClient struct {
	documents map[string]string
	requests  atomic.Int32
}

func (c *fakeClient) Do(req *http.Request) (*http.Response, error) {
	c.requests.Add(1)
	body, ok := c.documents[req.URL.String()]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
}

// denyClient refuses every request, as a restrictive caller-provided client would.
type denyClient struct{}

func (denyClient) Do(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("request to %s denied", req.URL)
}

// resolveDoc unmarshals a locally valid document and resolves its references.
func resolveDoc(t *testing.T, doc string, opts asyncapi.ResolveAllOptions) (*asyncapi.AsyncAPI, []error) {
	t.Helper()

	a, validationErrs := unmarshal(t, doc)
	require.Empty(t, errorsOnly(validationErrs), "document should be valid before resolution")

	errs, err := a.ResolveAllReferences(t.Context(), opts)
	require.NoError(t, err, "resolution should not fail")
	return a, errs
}

func nodeString(t *testing.T, node *yaml.Node) string {
	t.Helper()

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	require.NoError(t, enc.Encode(node), "node should encode")
	return buf.String()
}

const inDocumentResolutionDoc = `asyncapi: 3.1.0
info:
  title: T
  version: '1'
servers:
  prod:
    $ref: '#/components/servers/prod'
channels:
  orders:
    $ref: '#/components/channels/orders'
  replies:
    address: null
    messages:
      ack:
        payload:
          $ref: '#/components/schemas/Ack'
operations:
  placeOrder:
    action: send
    channel:
      $ref: '#/channels/orders'
    messages:
      - $ref: '#/channels/orders/messages/placed'
    tags:
      - $ref: '#/components/tags/orders'
    externalDocs:
      $ref: '#/components/externalDocs/guide'
    reply:
      address:
        location: $message.header#/replyTo
      channel:
        $ref: '#/channels/replies'
      messages:
        - $ref: '#/channels/replies/messages/ack'
components:
  servers:
    prod:
      host: broker.example.com
      protocol: kafka
  channels:
    orders:
      address: orders
      servers:
        - $ref: '#/servers/prod'
      messages:
        placed:
          $ref: '#/components/messages/placed'
  messages:
    placed:
      correlationId:
        $ref: '#/components/correlationIds/order'
      payload:
        $ref: '#/components/schemas/Order'
  schemas:
    Order:
      type: object
    Ack:
      type: object
  correlationIds:
    order:
      location: $message.payload#/id
  tags:
    orders:
      name: orders
  externalDocs:
    guide:
      url: https://example.com/guide
`

func TestResolveAllReferences_InDocument_Success(t *testing.T) {
	t.Parallel()

	a, errs := resolveDoc(t, inDocumentResolutionDoc, asyncapi.ResolveAllOptions{})
	assert.Empty(t, errs, "resolution should report nothing")

	components := a.GetComponents()
	placed := components.Messages.GetOrZero("placed").GetObject()
	require.NotNil(t, placed, "component message should be parsed")

	assert.Equal(t, "broker.example.com", a.GetServers().GetOrZero("prod").GetObject().GetHost(), "server reference should resolve")
	assert.Equal(t, "orders", a.GetChannels().GetOrZero("orders").GetObject().GetAddress(), "channel reference should resolve")

	operation := a.GetOperations().GetOrZero("placeOrder").GetObject()
	assert.Equal(t, "orders", operation.Channel.GetChannel().GetAddress(), "operation channel should resolve through the root channel")
	assert.Equal(t, "#/components/channels/orders", string(operation.Channel.GetResolvedReference()), "resolved reference should be the final target")
	assert.Same(t, placed, operation.Messages[0].GetMessage(), "operation message should resolve through the channel to the component")
	assert.Equal(t, "orders", operation.Tags[0].GetResolved().GetName(), "tag reference should resolve")
	assert.Equal(t, "https://example.com/guide", a.GetResolvedExternalDocs(operation.ExternalDocs).GetURL(), "external docs reference should resolve")

	reply := operation.Reply.GetObject()
	assert.NotNil(t, reply.Channel.GetChannel(), "reply channel should resolve")
	assert.NotNil(t, reply.Messages[0].GetMessage(), "reply message should resolve")

	assert.Same(t, components.Schemas.GetOrZero("Order"), a.GetResolvedSchema(placed.Payload).Schema, "payload schema reference should resolve")
	assert.Equal(t, "$message.payload#/id", placed.CorrelationID.GetObject().GetLocation(), "correlation ID reference should resolve")
	assert.Same(t, a.GetServers().GetOrZero("prod").GetObject(), components.Channels.GetOrZero("orders").GetObject().Servers[0].GetServer(), "channel server should resolve through the root server")
}

func TestResolveAllReferences_External_Success(t *testing.T) {
	t.Parallel()

	files := &countingFS{files: fstest.MapFS{
		"channels.yaml":        {Data: []byte("orders:\n  address: orders\n  messages:\n    placed:\n      $ref: './messages/placed.yaml#/OrderPlaced'\n")},
		"messages/placed.yaml": {Data: []byte("OrderPlaced:\n  name: OrderPlaced\n  payload:\n    $ref: '#/schemas/Order'\nschemas:\n  Order:\n    type: object\n")},
		"order.avsc":           {Data: []byte(`{"type": "record", "name": "Order", "fields": []}`)},
	}}
	client := &fakeClient{documents: map[string]string{
		"https://schemas.example.com/events/shipped.yaml": "shipped:\n  $ref: './common.yaml#/shipped'\n",
		"https://schemas.example.com/events/common.yaml":  "shipped:\n  name: OrderShipped\n",
	}}

	doc := `asyncapi: 3.1.0
info:
  title: T
  version: '1'
channels:
  orders:
    $ref: './channels.yaml#/orders'
  shipping:
    address: shipping
    messages:
      shipped:
        $ref: 'https://schemas.example.com/events/shipped.yaml#/shipped'
operations:
  placeOrder:
    action: send
    channel:
      $ref: '#/channels/orders'
    messages:
      - $ref: '#/channels/orders/messages/placed'
components:
  schemas:
    AvroOrder:
      schemaFormat: application/vnd.apache.avro;version=1.9.0
      schema:
        $ref: './order.avsc'
`
	a, errs := resolveDoc(t, doc, asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: files, HTTPClient: client})
	assert.Empty(t, errs, "resolution should report nothing")

	message := a.GetOperations().GetOrZero("placeOrder").GetObject().Messages[0].GetMessage()
	require.NotNil(t, message, "message should resolve through a relative chain of documents")
	assert.Equal(t, "OrderPlaced", message.GetName(), "message should come from the nested document")
	assert.Equal(t, "messages/placed.yaml#/OrderPlaced", string(a.GetChannels().GetOrZero("orders").GetObject().Messages.GetOrZero("placed").GetResolvedReference()), "nested reference should resolve against its own document")
	assert.Equal(t, []string{"object"}, a.GetResolvedSchema(message.Payload).Schema.GetSchema().GetSchema().GetType(), "reference within an external document should resolve against that document")

	shipped := a.GetChannels().GetOrZero("shipping").GetObject().Messages.GetOrZero("shipped").GetObject()
	assert.Equal(t, "OrderShipped", shipped.GetName(), "remote chain should resolve relative to the remote document")

	avro := a.GetResolvedSchema(a.GetComponents().Schemas.GetOrZero("AvroOrder"))
	require.NotNil(t, avro, "foreign schema resource should resolve")
	assert.Equal(t, "application/vnd.apache.avro;version=1.9.0", avro.SchemaFormat, "declared format should be kept")
	assert.Nil(t, avro.Schema, "foreign schema should not be interpreted")
	assert.Equal(t, "{\"type\": \"record\", \"name\": \"Order\", \"fields\": []}\n", nodeString(t, avro.Raw.Content[0]), "foreign schema content should be preserved unchanged")
}

func TestResolveAllReferences_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      string
		location string
		files    fstest.MapFS
		opts     asyncapi.ResolveAllOptions
		expected []string
	}{
		{
			name:     "missing target",
			doc:      minimalDoc("channels:\n  c:\n    $ref: '#/components/channels/missing'\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "wrong kind of target",
			doc:      minimalDoc("channels:\n  c:\n    $ref: '#/components/messages/m'\ncomponents:\n  messages:\n    m:\n      name: m\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "reference cycle",
			doc:      minimalDoc("components:\n  channels:\n    a:\n      $ref: '#/components/channels/b'\n    b:\n      $ref: '#/components/channels/a'\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationCircularReference, "error:" + validation.RuleValidationCircularReference},
		},
		{
			name:     "external references disabled",
			doc:      minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n"),
			location: "api.yaml",
			files:    fstest.MapFS{"c.yaml": {Data: []byte("c:\n  address: c\n")}},
			opts:     asyncapi.ResolveAllOptions{DisableExternalRefs: true},
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "relative reference without the document location",
			doc:      minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n"),
			files:    fstest.MapFS{"c.yaml": {Data: []byte("c:\n  address: c\n")}},
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "missing external document",
			doc:      minimalDoc("channels:\n  c:\n    $ref: './missing.yaml#/c'\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "remote reference denied by the client",
			doc:      minimalDoc("channels:\n  c:\n    $ref: 'https://example.com/c.yaml#/c'\n"),
			location: "api.yaml",
			opts:     asyncapi.ResolveAllOptions{HTTPClient: denyClient{}},
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "root operation channel in components",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel:\n      $ref: '#/components/channels/c'\ncomponents:\n  channels:\n    c:\n      address: c\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidTarget},
		},
		{
			name:     "operation message outside its channel",
			doc:      minimalDoc("channels:\n  c:\n    messages:\n      m:\n        name: m\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    messages:\n      - $ref: '#/components/messages/m'\ncomponents:\n  messages:\n    m:\n      name: m\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidTarget},
		},
		{
			name:     "root channel server in components",
			doc:      minimalDoc("channels:\n  c:\n    servers:\n      - $ref: '#/components/servers/s'\ncomponents:\n  servers:\n    s:\n      host: example.com\n      protocol: mqtt\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidTarget},
		},
		{
			name:     "runtime reply address with an addressed channel",
			doc:      minimalDoc("channels:\n  c:\n    address: c\n  r:\n    address: replies\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    reply:\n      address:\n        location: $message.header#/replyTo\n      channel:\n        $ref: '#/channels/r'\n"),
			location: "api.yaml",
			expected: []string{"error:" + validation.RuleValidationInvalidTarget},
		},
		{
			name:     "foreign schema resource declares a different format",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/vnd.apache.avro;version=1.9.0\n      schema:\n        $ref: './s.json'\n"),
			location: "api.yaml",
			files:    fstest.MapFS{"s.json": {Data: []byte(`{"schemaFormat": "application/raml+yaml;version=1.0", "schema": {}}`)}},
			expected: []string{"error:" + validation.RuleValidationInvalidReference},
		},
		{
			name:     "textual schema resource is not resolved",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/vnd.google.protobuf;version=3\n      schema:\n        $ref: './s.proto'\n"),
			location: "api.yaml",
			files:    fstest.MapFS{"s.proto": {Data: []byte("syntax = \"proto3\";\n")}},
			expected: []string{"warning:" + validation.RuleValidationUnsupportedFormat},
		},
		{
			name:     "external object is validated",
			doc:      minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n"),
			location: "api.yaml",
			files:    fstest.MapFS{"c.yaml": {Data: []byte("c:\n  address: orders?x=1\n")}},
			expected: []string{"error:" + validation.RuleValidationInvalidFormat},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.opts
			opts.AsyncAPILocation = tt.location
			opts.VirtualFS = tt.files

			_, errs := resolveDoc(t, tt.doc, opts)
			assert.Equal(t, tt.expected, diagnostics(errs), "resolution diagnostics should match")
		})
	}
}

func TestResolveAllReferences_ExternalDiagnosticLocation_Success(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"c.yaml": {Data: []byte("c:\n  address: orders?x=1\n")}}
	_, errs := resolveDoc(t, minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n"), asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: files})
	require.Len(t, errs, 1, "the external channel should be reported")

	var vErr *validation.Error
	require.ErrorAs(t, errs[0], &vErr, "diagnostic should be a validation error")
	assert.Equal(t, "c.yaml", vErr.DocumentLocation, "diagnostic should name the document it was found in")
	assert.Equal(t, 2, vErr.GetLineNumber(), "diagnostic should point at the line in that document")
}

const operationTraitsDoc = `asyncapi: 3.1.0
info:
  title: T
  version: '1'
channels:
  c:
    address: c
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    x-limits:
      burst: null
    traits:
      - $ref: '#/components/operationTraits/base'
      - summary: second
        description: null
        x-limits:
          rate: 20
components:
  operationTraits:
    base:
      summary: first
      description: from base
      tags:
        - name: a
      x-limits:
        rate: 10
        burst: 5
`

func TestResolveAllReferences_OperationTraits_Success(t *testing.T) {
	t.Parallel()

	a, errs := resolveDoc(t, operationTraitsDoc, asyncapi.ResolveAllOptions{})
	assert.Empty(t, errs, "resolution should report nothing")

	operation := a.GetOperations().GetOrZero("o").GetObject()
	effective := operation.GetEffective()
	require.NotNil(t, effective, "traits should be applied")
	assert.NotSame(t, operation, effective, "effective operation should be a separate snapshot")

	// Later traits win, the target wins at leaves, arrays are replaced and null deletes, including a target null
	// deleting an inherited value.
	expected := `summary: second
tags:
  - name: a
x-limits:
  rate: 20
action: send
channel:
  $ref: '#/channels/c'
`
	assert.Equal(t, expected, nodeString(t, effective.GetCore().GetRootNode()), "effective operation should merge traits in order")
	assert.Equal(t, "second", effective.GetSummary(), "effective summary should come from the later trait")
	assert.Nil(t, effective.Description, "null in a later trait should delete the earlier value")
	assert.Equal(t, "c", effective.Channel.GetChannel().GetAddress(), "effective operation references should be resolved")

	for field, index := range map[string]int{"summary": 1, "tags": 0, "x-limits": -1, "action": -1, "channel": -1} {
		source, ok := operation.GetTraitSource(field)
		assert.True(t, ok, "field %s should have a source", field)
		assert.Equal(t, index, source, "field %s should come from the expected trait", field)
	}
	_, ok := operation.GetTraitSource("description")
	assert.False(t, ok, "deleted field should have no source")

	assert.Equal(t, operationTraitsDoc, marshal(t, a), "applying traits should not change the document")
}

func TestResolveAllReferences_MessageTraits_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`components:
  messages:
    m:
      name: m
      headers:
        default: 5
      traits:
        - $ref: '#/components/messageTraits/json'
  messageTraits:
    json:
      contentType: application/json
      headers:
        type: object
`)
	a, errs := resolveDoc(t, doc, asyncapi.ResolveAllOptions{})

	// The trait's headers type and the message's headers default are each valid on their own, but not together.
	assert.Equal(t, []string{"error:" + validation.RuleValidationTypeMismatch}, diagnostics(errs), "only the inherited diagnostic should be reported")

	message := a.GetComponents().Messages.GetOrZero("m").GetObject()
	effective := message.GetEffective()
	require.NotNil(t, effective, "traits should be applied")
	assert.Equal(t, "application/json", effective.GetContentType(), "trait should supply the content type")
	assert.Equal(t, "m", effective.GetName(), "message should keep its own name")
	source, ok := message.GetTraitSource("contentType")
	assert.True(t, ok, "content type should have a source")
	assert.Equal(t, 0, source, "content type should come from the first trait")
}

func TestResolveAllReferences_ForbiddenTraitFields_Ignored(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`channels:
  c:
    address: c
  other:
    address: other
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    traits:
      - action: receive
        channel:
          $ref: '#/channels/other'
        summary: from trait
`)
	a, validationErrs := unmarshal(t, doc)
	assert.Len(t, errorsOnly(validationErrs), 2, "forbidden trait fields should be reported by Validate")

	_, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")

	effective := a.GetOperations().GetOrZero("o").GetObject().GetEffective()
	assert.Equal(t, asyncapi.OperationActionSend, effective.GetAction(), "trait must not supply the action")
	assert.Equal(t, "#/channels/c", string(effective.Channel.Reference), "trait must not supply the channel")
	assert.Equal(t, "from trait", effective.GetSummary(), "allowed trait fields should apply")
}

func TestResolveAllReferences_UntraitedSources_Success(t *testing.T) {
	t.Parallel()

	a, errs := resolveDoc(t, operationTraitsDoc, asyncapi.ResolveAllOptions{})
	require.Empty(t, errs, "resolution should report nothing")

	operation := a.GetOperations().GetOrZero("o").GetObject()
	operation.Traits = nil

	assert.Same(t, operation, operation.GetEffective(), "an operation without traits is its own effective view")
	source, ok := operation.GetTraitSource("action")
	assert.True(t, ok, "a field written on the operation should have a source")
	assert.Equal(t, -1, source, "a field written on the operation should come from the operation")
	_, ok = operation.GetTraitSource("summary")
	assert.False(t, ok, "a field from a removed trait should have no source")

	message := asyncapi.Message{}
	_, ok = message.GetTraitSource("name")
	assert.False(t, ok, "an absent field should have no source")
}

func TestResolveAllReferences_InlinedReference_Success(t *testing.T) {
	t.Parallel()

	a, errs := resolveDoc(t, inDocumentResolutionDoc, asyncapi.ResolveAllOptions{})
	require.Empty(t, errs, "resolution should report nothing")

	server := a.GetServers().GetOrZero("prod")
	require.NotEmpty(t, server.GetResolvedReference(), "reference should be resolved")

	server.Object = server.GetObject()
	server.Reference = nil
	assert.Empty(t, server.GetResolvedReference(), "an inline object has no resolved reference")
}

func TestResolveAllReferences_Refresh_Success(t *testing.T) {
	t.Parallel()

	doc := minimalDoc(`channels:
  a:
    address: a
  b:
    address: b
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/a'
    traits:
      - summary: one
`)
	a, errs := resolveDoc(t, doc, asyncapi.ResolveAllOptions{})
	require.Empty(t, errs, "resolution should report nothing")

	operation := a.GetOperations().GetOrZero("o").GetObject()
	previous := operation.GetEffective()
	assert.Equal(t, "a", operation.Channel.GetChannel().GetAddress(), "channel should resolve")
	assert.Equal(t, "one", previous.GetSummary(), "trait should apply")

	operation.Channel.Reference = "#/channels/b"
	operation.Traits[0].Object.Summary = pointer.From("two")

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Empty(t, errs, "resolution should report nothing")
	assert.Equal(t, "b", operation.Channel.GetChannel().GetAddress(), "edited reference should resolve to its new target")
	assert.Equal(t, "two", operation.GetEffective().GetSummary(), "effective view should reflect the edited trait")
	assert.Equal(t, "one", previous.GetSummary(), "previously returned effective views are snapshots")

	operation.Channel.Reference = "#/channels/missing"
	errs, err = a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Equal(t, []string{"error:" + validation.RuleValidationInvalidReference}, diagnostics(errs), "missing target should be reported")
	assert.Nil(t, operation.Channel.GetChannel(), "stale target should not be returned")
}

func TestResolveAllReferences_NoImplicitIO_Success(t *testing.T) {
	t.Parallel()

	files := &countingFS{files: fstest.MapFS{"c.yaml": {Data: []byte("c:\n  address: c\n")}}}
	client := &fakeClient{documents: map[string]string{"https://example.com/m.yaml": "m:\n  name: remote\n"}}
	doc := minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n  d:\n    messages:\n      m:\n        $ref: 'https://example.com/m.yaml#/m'\n")

	a, validationErrs := unmarshal(t, doc)
	assert.Empty(t, errorsOnly(validationErrs), "document should be valid")
	assert.Empty(t, a.Validate(t.Context()), "validation should report nothing")
	for item := range asyncapi.Walk(t.Context(), a) {
		_ = item
	}
	assert.Nil(t, a.GetChannels().GetOrZero("c").GetObject(), "references should not be resolved implicitly")

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", DisableExternalRefs: true, VirtualFS: files, HTTPClient: client})
	require.NoError(t, err, "resolution should not fail")
	require.Len(t, errs, 2, "disabled external references should be reported")
	for _, e := range errs {
		require.ErrorIs(t, e, asyncapi.ErrExternalReferencesDisabled, "disabled references should be identifiable")
	}
	assert.Zero(t, files.opened.Load(), "no files should be read before resolution is allowed")
	assert.Zero(t, client.requests.Load(), "no requests should be made before resolution is allowed")

	errs, err = a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: files, HTTPClient: client})
	require.NoError(t, err, "resolution should not fail")
	assert.Empty(t, errs, "resolution should report nothing")
	assert.Equal(t, int32(1), files.opened.Load(), "the referenced file should be read once")
	assert.Equal(t, int32(1), client.requests.Load(), "the referenced URL should be fetched once")
}

func TestResolveAllReferences_Cancelled_Error(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, inDocumentResolutionDoc)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := a.ResolveAllReferences(ctx, asyncapi.ResolveAllOptions{})
	require.ErrorIs(t, err, context.Canceled, "cancelled resolution should return the cancellation")
}

func TestResolveAllReferences_WalkUnchanged_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, inDocumentResolutionDoc)
	count := func() int {
		n := 0
		for range asyncapi.Walk(t.Context(), a) {
			n++
		}
		return n
	}

	before := count()
	_, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Equal(t, before, count(), "walking should visit only authored locations after resolution")
}

func TestResolveAllReferences_ConcurrentReads_Success(t *testing.T) {
	t.Parallel()

	a, _ := resolveDoc(t, inDocumentResolutionDoc, asyncapi.ResolveAllOptions{})
	operation := a.GetOperations().GetOrZero("placeOrder").GetObject()

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = operation.Channel.GetChannel()
				_ = operation.Messages[0].GetMessage()
				_ = a.GetChannels().GetOrZero("orders").GetObject()
				_ = a.GetResolvedExternalDocs(operation.ExternalDocs)
			}
		}()
	}

	_, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	wg.Wait()
	require.NoError(t, err, "resolution should not fail")
	assert.NotNil(t, operation.Channel.GetChannel(), "channel should be resolved")
}

func TestResolveAllReferences_EveryKindFromAnotherDocument_Success(t *testing.T) {
	t.Parallel()

	location := "testdata/resolve/api.yaml"
	data, err := os.ReadFile(location)
	require.NoError(t, err, "fixture should be readable")

	a, errs := resolveDoc(t, string(data), asyncapi.ResolveAllOptions{AsyncAPILocation: location})
	assert.Empty(t, errorsOnly(errs), "resolution should report no errors")

	info := a.GetInfo()
	assert.Equal(t, "orders", info.Tags[0].GetResolved().GetName(), "tag should resolve through a chain in another document")
	assert.Equal(t, "https://example.com/guide", a.GetResolvedExternalDocs(info.ExternalDocs).GetURL(), "external docs should resolve through a chain in another document")

	server := a.GetServers().GetOrZero("prod").GetObject()
	require.NotNil(t, server, "server should resolve")
	assert.Equal(t, "{region}.example.com", server.GetHost(), "server should come from the other document")
	assert.Equal(t, "eu", server.Variables.GetOrZero("region").GetObject().GetDefault(), "server variable should resolve")
	assert.Equal(t, "X-API-Key", server.Security[0].GetObject().GetName(), "server security scheme should resolve")
	assert.Equal(t, "orders", server.Tags[0].GetResolved().GetName(), "server tag should resolve")
	assert.Equal(t, "https://example.com/guide", a.GetResolvedExternalDocs(server.ExternalDocs).GetURL(), "server external docs should resolve")
	assert.Equal(t, "0.5.0", server.Bindings.GetObject().GetBindingVersion("kafka"), "server bindings should resolve")

	channel := a.GetChannels().GetOrZero("orders").GetObject()
	assert.Same(t, server, channel.Servers[0].GetServer(), "channel server should resolve")
	assert.Equal(t, "The order ID.", channel.Parameters.GetOrZero("orderId").GetObject().GetDescription(), "parameter should resolve")
	assert.NotNil(t, channel.Bindings.GetObject(), "channel bindings should resolve")

	operation := a.GetOperations().GetOrZero("placeOrder").GetObject()
	assert.Equal(t, "X-API-Key", operation.Security[0].GetObject().GetName(), "operation security scheme should resolve")
	assert.Equal(t, "Tracked operation.", operation.Traits[0].GetObject().GetSummary(), "operation trait should resolve")
	assert.Equal(t, "Tracked operation.", operation.GetEffective().GetSummary(), "operation trait should apply")
	assert.Equal(t, "$message.header#/replyTo", operation.Reply.GetObject().Address.GetObject().GetLocation(), "reply and reply address should resolve")

	message := operation.Messages[0].GetMessage()
	require.NotNil(t, message, "message should resolve")
	assert.Equal(t, "OrderPlaced", message.GetName(), "message should come from the other document")
	assert.Equal(t, "$message.payload#/id", message.CorrelationID.GetObject().GetLocation(), "correlation ID should resolve")
	assert.Equal(t, "application/json", message.Traits[0].GetObject().GetContentType(), "message trait should resolve")
	assert.Equal(t, "application/json", message.GetEffective().GetContentType(), "message trait should apply")
}

// resolveCancellingFS cancels resolution when a file is opened, while still serving the file.
type resolveCancellingFS struct {
	files  fstest.MapFS
	cancel context.CancelFunc
}

func (c *resolveCancellingFS) Open(name string) (fs.File, error) {
	c.cancel()
	return c.files.Open(name)
}

func TestResolveAllReferences_CancelledWhileFollowing_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		doc   string
		files fstest.MapFS
	}{
		{
			name:  "reference object chain",
			doc:   minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n"),
			files: fstest.MapFS{"c.yaml": {Data: []byte("c:\n  $ref: '#/d'\nd:\n  address: d\n")}},
		},
		{
			name:  "schema chain",
			doc:   minimalDoc("components:\n  schemas:\n    s:\n      $ref: './s.yaml#/s'\n"),
			files: fstest.MapFS{"s.yaml": {Data: []byte("s:\n  $ref: '#/t'\nt:\n  type: object\n")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, validationErrs := unmarshal(t, tt.doc)
			require.Empty(t, errorsOnly(validationErrs), "document should be valid before resolution")

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			errs, err := a.ResolveAllReferences(ctx, asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: &resolveCancellingFS{files: tt.files, cancel: cancel}})
			require.ErrorIs(t, err, context.Canceled, "cancelled resolution should return the cancellation")
			assert.Empty(t, errs, "cancellation should not be reported as a reference problem")
		})
	}
}

func TestResolveAllReferences_Messages_Error(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"null.yaml":  {Data: []byte("c:\n  $ref: null\n")},
		"mf.json":    {Data: []byte(`{"schemaFormat": "application/vnd.apache.avro;version=1.9.0", "schema": {}}`)},
		"alias.yaml": {Data: []byte("x: &f application/vnd.apache.avro;version=1.9.0\nschemaFormat: *f\n")},
	}
	const avroSchema = "    avro:\n      schemaFormat: application/vnd.apache.avro;version=1.9.0\n      schema: {}\n"

	tests := []struct {
		name     string
		doc      string
		location string
		expected []string
	}{
		{
			name:     "reference cycle",
			doc:      minimalDoc("components:\n  channels:\n    a:\n      $ref: '#/components/channels/b'\n    b:\n      $ref: '#/components/channels/a'\n"),
			location: "api.yaml",
			expected: []string{
				"[8:13] error validation-circular-reference circular reference: api.yaml#/components/channels/b -> api.yaml#/components/channels/a -> api.yaml#/components/channels/b",
				"[10:13] error validation-circular-reference circular reference: api.yaml#/components/channels/a -> api.yaml#/components/channels/b -> api.yaml#/components/channels/a",
			},
		},
		{
			name:     "wrong kind of target",
			doc:      minimalDoc("channels:\n  c:\n    $ref: '#/components/messages/m'\ncomponents:\n  messages:\n    m:\n      name: m\n"),
			location: "api.yaml",
			expected: []string{"[7:11] error validation-invalid-reference channel reference `#/components/messages/m` points to a message, not a channel"},
		},
		{
			name:     "invalid document location",
			doc:      minimalDoc("channels:\n  c:\n    $ref: './c.yaml#/c'\n"),
			location: "http://[::1",
			expected: []string{"[7:11] error validation-invalid-reference channel reference `./c.yaml#/c` is invalid: invalid reference format: parse \"http://[::1\": missing ']' in host"},
		},

		{
			name:     "target is an empty reference",
			doc:      minimalDoc("channels:\n  c:\n    $ref: './null.yaml#/c'\n"),
			location: "api.yaml",
			expected: []string{"[7:11] error validation-invalid-reference channel reference `./null.yaml#/c` doesn't point to a channel"},
		},
		{
			name:     "target is unset",
			doc:      minimalDoc("channels:\n  c:\n    externalDocs:\n      $ref: '#/info/externalDocs'\n"),
			location: "api.yaml",
			expected: []string{"[8:13] error validation-invalid-reference external documentation reference `#/info/externalDocs` resolved to nothing"},
		},
		{
			name:     "missing tag",
			doc:      minimalDoc("channels:\n  c:\n    tags:\n      - $ref: '#/components/tags/missing'\n"),
			location: "api.yaml",
			expected: []string{"[8:15] error validation-invalid-reference tag reference `#/components/tags/missing` target not found: not found -- struct is nil at /components/tags"},
		},
		{
			name:     "missing external documentation",
			doc:      minimalDoc("channels:\n  c:\n    externalDocs:\n      $ref: '#/components/externalDocs/missing'\n"),
			location: "api.yaml",
			expected: []string{"[8:13] error validation-invalid-reference external documentation reference `#/components/externalDocs/missing` target not found: not found -- struct is nil at /components/externalDocs"},
		},
		{
			name:     "missing schema",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      $ref: '#/components/schemas/missing'\n"),
			location: "api.yaml",
			expected: []string{"[8:7] error validation-invalid-reference schema reference `#/components/schemas/missing` target not found: not found -- key missing not found in sequencedmap.Map"},
		},
		{
			name:     "missing JSON Schema resource",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/schema+json;version=draft-07\n      schema:\n        $ref: './missing.json'\n"),
			location: "api.yaml",
			expected: []string{"[10:9] error validation-invalid-reference failed to resolve schema application/schema+json;version=draft-07 reference `./missing.json`: open missing.json: file does not exist"},
		},
		{
			name:     "JSON Schema resource declares a different format",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/schema+json;version=draft-07\n      schema:\n        $ref: './mf.json'\n"),
			location: "api.yaml",
			expected: []string{"[10:9] error validation-invalid-reference schema reference `./mf.json` is declared as `application/schema+json;version=draft-07` but its target is `application/vnd.apache.avro;version=1.9.0`"},
		},
		{
			name:     "missing in-document foreign schema",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/raml+yaml;version=1.0\n      schema:\n        $ref: '#/components/schemas/missing'\n"),
			location: "api.yaml",
			expected: []string{"[10:9] error validation-invalid-reference schema application/raml+yaml;version=1.0 reference `#/components/schemas/missing` target not found: not found -- key missing not found in sequencedmap.Map"},
		},
		{
			name:     "in-document foreign schema declares a different format",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/raml+yaml;version=1.0\n      schema:\n        $ref: '#/components/schemas/avro'\n" + avroSchema),
			location: "api.yaml",
			expected: []string{"[10:9] error validation-invalid-reference schema reference `#/components/schemas/avro` is declared as `application/raml+yaml;version=1.0` but its target is `application/vnd.apache.avro;version=1.9.0`"},
		},
		{
			name:     "missing foreign schema resource",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/vnd.apache.avro;version=1.9.0\n      schema:\n        $ref: './missing.avsc'\n"),
			location: "api.yaml",
			expected: []string{"[10:9] error validation-invalid-reference failed to resolve raw application/vnd.apache.avro;version=1.9.0 reference `./missing.avsc`: open missing.avsc: file does not exist"},
		},
		{
			name:     "foreign schema resource declares a different format through an alias",
			doc:      minimalDoc("components:\n  schemas:\n    s:\n      schemaFormat: application/raml+yaml;version=1.0\n      schema:\n        $ref: './alias.yaml'\n"),
			location: "api.yaml",
			expected: []string{"[10:9] error validation-invalid-reference schema reference `./alias.yaml` is declared as `application/raml+yaml;version=1.0` but its target is `application/vnd.apache.avro;version=1.9.0`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, errs := resolveDoc(t, tt.doc, asyncapi.ResolveAllOptions{AsyncAPILocation: tt.location, VirtualFS: files})
			actual := make([]string, 0, len(errs))
			for _, err := range errs {
				actual = append(actual, err.Error())
			}
			assert.Equal(t, tt.expected, actual, "resolution errors should match")
		})
	}
}

// An absolute path doesn't need the document's location, so it is attempted rather than refused. The path is built
// for the current platform, as `/abs/c.yaml` isn't absolute on Windows.
func TestResolveAllReferences_AbsolutePathWithoutLocation_Error(t *testing.T) {
	t.Parallel()

	abs, err := filepath.Abs(filepath.Join("missing", "c.yaml"))
	require.NoError(t, err, "absolute path should be built")

	_, errs := resolveDoc(t, minimalDoc("channels:\n  c:\n    $ref: '"+abs+"#/c'\n"), asyncapi.ResolveAllOptions{VirtualFS: fstest.MapFS{}})
	require.Len(t, errs, 1, "the missing file should be reported")
	assert.Contains(t, errs[0].Error(), "channel reference", "the error should be about the channel reference")
	assert.NotContains(t, errs[0].Error(), "document's location is unknown", "an absolute path should not need the document's location")
}

func TestResolveAllReferences_SchemaFormats_Success(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"order.json": {Data: []byte(`{"type": "object"}`)},
		"union.avsc": {Data: []byte(`["null", "string"]`)},
	}
	doc := minimalDoc(`components:
  schemas:
    jsonSchema:
      schemaFormat: application/schema+json;version=draft-07
      schema:
        $ref: './order.json'
    inDocument:
      schemaFormat: application/vnd.apache.avro;version=1.9.0
      schema:
        $ref: '#/components/schemas/avro'
    avro:
      schemaFormat: application/vnd.apache.avro;version=1.9.0
      schema:
        type: string
    avroUnion:
      schemaFormat: application/vnd.apache.avro;version=1.9.0
      schema:
        $ref: './union.avsc'
`)
	a, errs := resolveDoc(t, doc, asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: files})
	assert.Empty(t, errs, "resolution should report nothing")
	schemas := a.GetComponents().Schemas

	jsonSchema := a.GetResolvedSchema(schemas.GetOrZero("jsonSchema"))
	require.NotNil(t, jsonSchema, "JSON Schema resource should resolve")
	assert.Equal(t, "application/schema+json;version=draft-07", jsonSchema.SchemaFormat, "declared format should be kept")
	assert.Equal(t, []string{"object"}, jsonSchema.Schema.GetSchema().GetSchema().GetType(), "JSON Schema resource should be interpreted")
	assert.Equal(t, "order.json", string(jsonSchema.Reference), "reference should be absolute")

	inDocument := a.GetResolvedSchema(schemas.GetOrZero("inDocument"))
	require.NotNil(t, inDocument, "in-document foreign schema should resolve")
	assert.Same(t, schemas.GetOrZero("avro"), inDocument.Schema, "in-document foreign schema should resolve to the component")
	assert.Equal(t, "application/vnd.apache.avro;version=1.9.0", inDocument.SchemaFormat, "declared format should be kept")
	assert.Equal(t, "api.yaml#/components/schemas/avro", string(inDocument.Reference), "reference should be absolute")

	avroUnion := a.GetResolvedSchema(schemas.GetOrZero("avroUnion"))
	require.NotNil(t, avroUnion, "foreign schema resource that isn't a mapping should resolve")
	assert.Nil(t, avroUnion.Schema, "foreign schema should not be interpreted")
	assert.Equal(t, "[\"null\", \"string\"]\n", nodeString(t, avroUnion.Raw.Content[0]), "foreign schema content should be preserved unchanged")
}

func TestResolveAllReferences_Getters_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, inDocumentResolutionDoc)
	operation := a.GetOperations().GetOrZero("placeOrder").GetObject()
	payload := a.GetComponents().Messages.GetOrZero("placed").GetObject().Payload
	inline := &asyncapi.ExternalDocumentation{URL: pointer.From("https://example.com")}

	var missing *asyncapi.AsyncAPI
	assert.Nil(t, missing.GetResolvedSchema(payload), "a nil document should have no resolved schema")
	assert.Nil(t, a.GetResolvedSchema(nil), "a nil schema should have no resolution")
	assert.Nil(t, a.GetResolvedSchema(payload), "an unresolved document should have no resolved schema")
	assert.Nil(t, missing.GetResolvedExternalDocs(inline), "a nil document should have no resolved external docs")
	assert.Nil(t, a.GetResolvedExternalDocs(nil), "nil external docs should have no resolution")
	assert.Same(t, inline, a.GetResolvedExternalDocs(inline), "inline external docs should be returned as is")
	assert.Nil(t, a.GetResolvedExternalDocs(operation.ExternalDocs), "an unresolved reference should have no resolved external docs")
}

func TestResolveAllReferences_NilDocument_Success(t *testing.T) {
	t.Parallel()

	var a *asyncapi.AsyncAPI
	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolving a nil document should not fail")
	assert.Nil(t, errs, "resolving a nil document should report nothing")
}

func TestResolveAllReferences_UnsyncableDocument_Error(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, minimalDoc("components:\n  schemas:\n    s:\n      type: object\n"))
	a.GetComponents().Schemas.Set("empty", jsonschema.NewSchemaUnionFromSchema(&jsonschema.JSONSchema{}))

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.Error(t, err, "a document that can't be synced should fail")
	assert.True(t, strings.HasPrefix(err.Error(), "failed to sync AsyncAPI document: "), "error should say syncing failed")
	assert.Nil(t, errs, "nothing should be reported when resolution doesn't start")
}

func TestResolveAllReferences_InvalidVersion_Success(t *testing.T) {
	t.Parallel()

	a, _ := unmarshal(t, "asyncapi: three\ninfo:\n  title: T\n  version: '1'\nchannels:\n  c:\n    $ref: '#/components/channels/c'\ncomponents:\n  channels:\n    c:\n      address: c\n")

	errs, err := a.ResolveAllReferences(t.Context(), asyncapi.ResolveAllOptions{})
	require.NoError(t, err, "resolution should not fail")
	assert.Empty(t, errs, "resolution should report nothing")
	assert.Equal(t, "c", a.GetChannels().GetOrZero("c").GetObject().GetAddress(), "references should resolve regardless of the version")
}
