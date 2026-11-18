package asyncapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A channel reference that can't be made absolute fails to resolve, so this state is only reachable directly.
func TestCheckChannelMessages_InvalidChannelReference_Success(t *testing.T) {
	t.Parallel()

	rc := traitsInternalCall(nil)
	rc.root.uri = "https://example.com/api.yaml"

	channel := NewReferenceObject("ht tp://invalid url#/channels/c")
	channel.setResolved(&Channel{}, "")
	messages := []*ReferenceObject{NewReferenceObject("#/components/messages/m")}

	rc.checkChannelMessages(channel, messages, "operation.messages", rc.root)
	assert.Empty(t, rc.errs, "messages of a channel whose location is unknown should not be checked")
}
