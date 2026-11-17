package asyncapi

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/validation"
)

// The checks in this file need referenced objects, so they run in ResolveAllReferences rather than Validate.

// checkOperation checks that a root operation's channel is in the root channels map, and that the operation's
// messages belong to its channel.
func (rc *resolveCall) checkOperation(op *Operation, pointer []string, base location) {
	if rc.isRoot(base) && isRootEntry(pointer, "operations") {
		rc.checkRootScope(&op.Channel, "channels", "operation.channel", base)
	}
	rc.checkChannelMessages(&op.Channel, op.Messages, "operation.messages", base)
}

// checkOperationReply checks a reply's channel and messages like an operation's, and that a reply with a runtime
// address uses a channel without a fixed address.
func (rc *resolveCall) checkOperationReply(reply *OperationReply, pointer []string, base location) {
	if reply.Channel == nil {
		return
	}

	// A reply written inline in a root operation is located in the root operation. A reply defined in components and
	// referenced from one isn't, so it may point to a channel anywhere.
	if rc.isRoot(base) && len(pointer) == 3 && pointer[0] == "operations" && pointer[2] == "reply" {
		rc.checkRootScope(reply.Channel, "channels", "operation.reply.channel", base)
	}
	rc.checkChannelMessages(reply.Channel, reply.Messages, "operation.reply.messages", base)

	if reply.Address != nil {
		if channel := reply.Channel.GetChannel(); channel != nil && channel.Address != nil {
			c := reply.Channel.GetCore()
			rc.addErrors([]error{validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidTarget, fmt.Errorf("operation.reply.channel `%s` must have a null or absent address because operation.reply.address is set, got `%s`", reply.Channel.Reference, *channel.Address), c, c.Reference)}, base)
		}
	}
}

// checkChannel checks that a root channel's servers are in the root servers map.
func (rc *resolveCall) checkChannel(channel *Channel, pointer []string, base location) {
	if !rc.isRoot(base) || !isRootEntry(pointer, "channels") {
		return
	}
	for _, server := range channel.Servers {
		rc.checkRootScope(server, "servers", "channel.servers", base)
	}
}

func isRootEntry(pointer []string, section string) bool {
	return len(pointer) == 2 && pointer[0] == section
}

// checkRootScope checks that a reference points directly to an entry of a root map of the same document, as objects
// in the root of the document must.
func (rc *resolveCall) checkRootScope(ref *ReferenceObject, section, field string, base location) {
	if ref == nil || ref.Reference == "" {
		return
	}

	inDocument := ref.Reference.GetURI() == ""
	if !inDocument {
		abs, err := absoluteReference(ref.Reference, base.uri)
		inDocument = err == nil && rc.root.uri != "" && strings.HasPrefix(abs, rc.root.uri+"#")
	}

	parts := strings.Split(strings.TrimPrefix(string(ref.Reference.GetJSONPointer()), "/"), "/")
	if inDocument && len(parts) == 2 && parts[0] == section && parts[1] != "" {
		return
	}

	c := ref.GetCore()
	rc.addErrors([]error{validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidTarget, fmt.Errorf("%s `%s` must point to an entry in the root `%s` of the same document", field, ref.Reference, section), c, c.Reference)}, base)
}

// checkChannelMessages checks that each message reference points to one of the messages of the referenced channel,
// through the channel's own location, rather than to a message defined anywhere else.
func (rc *resolveCall) checkChannelMessages(channelRef *ReferenceObject, messages []*ReferenceObject, field string, base location) {
	channel := channelRef.GetChannel()
	if channel == nil {
		return
	}

	authored, err := absoluteReference(channelRef.Reference, base.uri)
	if err != nil {
		return
	}
	resolved := string(channelRef.GetResolvedReference())

	for i, message := range messages {
		if message == nil || message.Reference == "" {
			continue
		}
		abs, err := absoluteReference(message.Reference, base.uri)
		if err != nil {
			continue
		}
		if isChannelMessage(abs, authored, channel) || isChannelMessage(abs, resolved, channel) {
			continue
		}

		c := message.GetCore()
		rc.addErrors([]error{validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidTarget, fmt.Errorf("%s[%d] `%s` must point to a message of the channel `%s`", field, i, message.Reference, channelRef.Reference), c, c.Reference)}, base)
	}
}

func isChannelMessage(abs, channelAbs string, channel *Channel) bool {
	if channelAbs == "" || channel.Messages == nil {
		return false
	}
	key, ok := strings.CutPrefix(abs, channelAbs+"/messages/")
	if !ok || strings.Contains(key, "/") {
		return false
	}
	return channel.Messages.Has(jsonpointer.UnescapeString(key))
}
