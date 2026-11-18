package asyncapi_test

import (
	"testing"
	"testing/fstest"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/stretchr/testify/assert"
)

func TestResolveAllReferences_Checks_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      string
		opts     asyncapi.ResolveAllOptions
		expected []string
	}{
		{
			name:     "root operation channel in another document",
			doc:      minimalDoc("operations:\n  o:\n    action: send\n    channel:\n      $ref: './other.yaml#/channels/c'\n"),
			opts:     asyncapi.ResolveAllOptions{AsyncAPILocation: "api.yaml", VirtualFS: fstest.MapFS{"other.yaml": {Data: []byte("channels:\n  c:\n    address: c\n")}}},
			expected: []string{"[9:13] error validation-invalid-target operation.channel `./other.yaml#/channels/c` must point to an entry in the root `channels` of the same document"},
		},
		{
			name:     "operation message with a channel without messages",
			doc:      minimalDoc("channels:\n  c:\n    address: c\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    messages:\n      - $ref: '#/components/messages/m'\ncomponents:\n  messages:\n    m:\n      name: m\n"),
			expected: []string{"[14:15] error validation-invalid-target operation.messages[0] `#/components/messages/m` must point to a message of the channel `#/channels/c`"},
		},
		// Only the reference itself is reported: the channel check skips a reference it can't locate.
		{
			name:     "operation message reference that can't be made absolute",
			doc:      minimalDoc("channels:\n  c:\n    address: c\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    messages:\n      - $ref: 'ht tp://invalid url#/m'\n"),
			opts:     asyncapi.ResolveAllOptions{AsyncAPILocation: "https://example.com/api.yaml"},
			expected: []string{"[14:15] error validation-invalid-reference message reference `ht tp://invalid url#/m` is invalid: invalid relative URL: parse \"ht tp://invalid url\": first path segment in URL cannot contain colon"},
		},
		{
			name: "root operation reply channel outside the root channels",
			doc: minimalDoc(`channels:
  c:
    address: c
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    reply:
      channel:
        $ref: '#/components/channels/r'
components:
  channels:
    r:
      address: r
`),
			expected: []string{"[15:15] error validation-invalid-target operation.reply.channel `#/components/channels/r` must point to an entry in the root `channels` of the same document"},
		},
		{
			name: "reply message that is not a message of the reply channel",
			doc: minimalDoc(`channels:
  c:
    address: c
  r:
    address: r
    messages:
      ok:
        name: ok
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    reply:
      channel:
        $ref: '#/channels/r'
      messages:
        - $ref: '#/components/messages/m'
components:
  messages:
    m:
      name: m
`),
			expected: []string{"[22:17] error validation-invalid-target operation.reply.messages[0] `#/components/messages/m` must point to a message of the channel `#/channels/r`"},
		},
		{
			name: "reply address with a reply channel that has a fixed address",
			doc: minimalDoc(`channels:
  c:
    address: c
  r:
    address: r
operations:
  o:
    action: send
    channel:
      $ref: '#/channels/c'
    reply:
      address:
        location: $message.header#/replyTo
      channel:
        $ref: '#/channels/r'
`),
			expected: []string{"[19:15] error validation-invalid-target operation.reply.channel `#/channels/r` must have a null or absent address because operation.reply.address is set, got `r`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, errs := resolveIgnoringValidation(t, tt.doc, tt.opts)
			assert.Equal(t, tt.expected, errorStrings(errs), "resolution diagnostics should match")
		})
	}
}

func TestResolveAllReferences_Checks_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
	}{
		{
			name: "reply without a channel",
			doc:  minimalDoc("channels:\n  c:\n    address: c\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    reply:\n      address:\n        location: $message.header#/replyTo\n"),
		},
		{
			name: "root operation without a channel",
			doc:  minimalDoc("operations:\n  o:\n    action: send\n"),
		},
		{
			name: "operation message that is not a reference",
			doc:  minimalDoc("channels:\n  c:\n    address: c\n    messages:\n      m:\n        name: m\noperations:\n  o:\n    action: send\n    channel:\n      $ref: '#/channels/c'\n    messages:\n      - {}\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, errs := resolveIgnoringValidation(t, tt.doc, asyncapi.ResolveAllOptions{})
			assert.Empty(t, errorStrings(errs), "cross-object checks should report nothing")
		})
	}
}
