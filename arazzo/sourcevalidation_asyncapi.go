package arazzo

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/asyncapi"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/validation"
)

// asyncTarget is what an AsyncAPI step (operationId or channelPath) selects in an AsyncAPI source.
type asyncTarget struct {
	// messages are the effective (traits applied) messages the step may send or receive.
	messages []*asyncapi.Message
	// complete is false if any reference needed to find the messages wasn't resolved, in which case messages are
	// partial and checks that something is not declared are skipped.
	complete bool
	// headers are the lower cased header names declared by the messages. Only set if headersKnown is true.
	headers map[string]bool
	// headersKnown is false if a message's headers schema can't be reduced to a list of properties, for example
	// because it uses composition or a non JSON Schema format.
	headersKnown bool
	// label identifies the target in messages.
	label string
}

// validateAsyncStep checks a step targeting an AsyncAPI source with operationId or channelPath against the source.
func (a *Arazzo) validateAsyncStep(sources *SourceDocuments, s *Step) []error {
	core := s.GetCore()

	target, errs := a.findAsyncTarget(sources, s)
	if target == nil {
		return errs
	}

	isReceive := s.Action != nil && *s.Action == StepActionReceive

	for i, parameter := range s.Parameters {
		p := a.resolveParameter(parameter)
		if p == nil || p.In == nil {
			continue
		}

		if *p.In != InHeader {
			errs = append(errs, validation.NewSliceError(validation.SeverityWarning, validation.RuleValidationOperationParameters, fmt.Errorf("step.parameters `%s` in `%s` has no meaning for AsyncAPI %s, only `header` parameters are sent with messages", p.Name, *p.In, target.label), core, core.Parameters, i))
			continue
		}

		if target.complete && target.headersKnown && !target.headers[strings.ToLower(p.Name)] {
			errs = append(errs, validation.NewSliceError(validation.SeverityWarning, validation.RuleValidationOperationParameters, fmt.Errorf("step.parameters `%s` in `header` is not a header of any message of %s", p.Name, target.label), core, core.Parameters, i))
		}
	}

	if s.RequestBody != nil && isReceive {
		errs = append(errs, validation.NewValueError(validation.SeverityWarning, validation.RuleValidationOperationParameters, fmt.Errorf("step.requestBody is not sent by a step with action `receive` targeting %s", target.label), core, core.RequestBody))
	}

	if s.CorrelationID != nil && isReceive && target.complete && !anyMessageHasCorrelationID(target.messages) {
		errs = append(errs, validation.NewValueError(validation.SeverityWarning, validation.RuleValidationInvalidReference, fmt.Errorf("step.correlationId is set but no message of %s declares a correlationId", target.label), core, core.CorrelationID))
	}

	return errs
}

// findAsyncTarget finds what the step's operationId or channelPath selects in an AsyncAPI source. Returns a nil
// target if the source wasn't loaded, the target can't be determined (reported by structural validation) or the
// target is invalid, in which case the returned errors explain why.
func (a *Arazzo) findAsyncTarget(sources *SourceDocuments, s *Step) (*asyncTarget, []error) {
	core := s.GetCore()

	if s.ChannelPath != nil {
		matches := operationPathRegex.FindStringSubmatch(s.ChannelPath.String())
		if matches == nil {
			return nil, nil
		}
		name, pointer := matches[1], matches[2]

		doc := sources.Get(name)
		if doc == nil {
			return nil, nil
		}
		if doc.AsyncAPI == nil {
			// Sources declared as another type are reported by Validate.
			if sd := a.SourceDescriptions.Find(name); sd != nil && sd.Type == "" {
				return nil, []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("step.channelPath references sourceDescription `%s` which is an %s document, not an AsyncAPI document", name, doc.Type), core, core.ChannelPath)}
			}
			return nil, nil
		}

		referenced := findChannelByPath(doc.AsyncAPI, pointer)
		if referenced == nil {
			return nil, []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationOperationNotFound, fmt.Errorf("step.channelPath `%s` doesn't reference a channel in sourceDescription `%s`", s.ChannelPath, name), core, core.ChannelPath)}
		}

		target := newAsyncTarget(fmt.Sprintf("channel `%s` in sourceDescription `%s`", pointer, name))
		channel := referenced.GetObject()
		if channel == nil {
			target.complete = false
		} else {
			target.addChannelMessages(channel)
		}
		target.collectHeaders(doc.AsyncAPI)
		return target, nil
	}

	sd, doc := a.operationIDSourceDocument(sources, s)
	if sd == nil || doc == nil || doc.AsyncAPI == nil {
		return nil, nil
	}

	errs := []error{}
	// Validate requires action for sources declared as asyncapi, an omitted type is only known once loaded.
	if s.Action == nil && sd.Type == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("step.action is required for asyncapi steps, operationId references sourceDescription `%s` which is an AsyncAPI document", sd.Name), core, core.OperationID))
	}

	// AsyncAPI identifies an application's operations by their key in the root operations map. Operations in
	// components are reusable definitions, not operations of the application.
	operationID := stepOperationID(s)
	referenced, ok := doc.AsyncAPI.GetOperations().Get(operationID)
	if !ok || referenced == nil {
		return nil, append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationOperationNotFound, fmt.Errorf("step.operationId `%s` not found in sourceDescription `%s`", operationID, sd.Name), core, core.OperationID))
	}

	target := newAsyncTarget(fmt.Sprintf("operation `%s` in sourceDescription `%s`", operationID, sd.Name))
	operation := referenced.GetObject()
	if operation == nil {
		target.complete = false
	} else {
		target.addOperationMessages(operation)
	}
	target.collectHeaders(doc.AsyncAPI)

	return target, errs
}

func newAsyncTarget(label string) *asyncTarget {
	return &asyncTarget{complete: true, headers: map[string]bool{}, headersKnown: true, label: label}
}

// addOperationMessages adds the operation's messages, or all the messages of its channel if it omits `messages`. An
// explicitly empty `messages` list means the operation supports no messages.
func (t *asyncTarget) addOperationMessages(operation *asyncapi.Operation) {
	if refs := operation.GetMessages(); refs != nil {
		for _, ref := range refs {
			t.addMessage(ref.GetMessage())
		}
		return
	}

	channel := operation.GetChannel().GetChannel()
	if channel == nil {
		t.complete = false
		return
	}
	t.addChannelMessages(channel)
}

func (t *asyncTarget) addChannelMessages(channel *asyncapi.Channel) {
	for _, referenced := range channel.GetMessages().All() {
		t.addMessage(referenced.GetObject())
	}
}

func (t *asyncTarget) addMessage(message *asyncapi.Message) {
	if message == nil {
		t.complete = false
		return
	}
	t.messages = append(t.messages, message.GetEffective())
}

func (t *asyncTarget) collectHeaders(doc *asyncapi.AsyncAPI) {
	for _, message := range t.messages {
		names, ok := messageHeaderNames(doc, message)
		if !ok {
			t.headersKnown = false
			continue
		}
		for name := range names {
			t.headers[name] = true
		}
	}
}

// messageHeaderNames returns the lower cased names of the headers declared by the message's headers schema. ok is
// false if the schema can't be reduced to a list of properties.
func messageHeaderNames(doc *asyncapi.AsyncAPI, message *asyncapi.Message) (names map[string]bool, ok bool) {
	names = map[string]bool{}

	union := message.GetHeaders()
	if union == nil {
		return names, true
	}
	if resolved := doc.GetResolvedSchema(union); resolved != nil {
		union = resolved.Schema
	}

	schema := union.GetSchema()
	if schema == nil {
		schema = union.GetMultiFormat().GetSchema().GetSchema()
	}

	s := schema.GetSchema()
	if s == nil || s.Ref != nil || s.Properties == nil || len(s.AllOf) > 0 || len(s.AnyOf) > 0 || len(s.OneOf) > 0 || s.If != nil || s.PatternProperties.Len() > 0 {
		return nil, false
	}

	for name := range s.Properties.Keys() {
		names[strings.ToLower(name)] = true
	}
	return names, true
}

func anyMessageHasCorrelationID(messages []*asyncapi.Message) bool {
	for _, message := range messages {
		if message.GetCorrelationID() != nil {
			return true
		}
	}
	return false
}

// findChannelByPath finds the channel referenced by a JSON Pointer such as `/channels/userSignedUp` or
// `/components/channels/userSignedUp`. Channels are found by their key, not their address.
func findChannelByPath(doc *asyncapi.AsyncAPI, pointer string) *asyncapi.ReferencedChannel {
	if !strings.HasPrefix(pointer, "/") {
		return nil
	}

	tokens := strings.Split(pointer[1:], "/")
	for i, token := range tokens {
		tokens[i] = jsonpointer.UnescapeString(token)
	}

	var channel *asyncapi.ReferencedChannel
	switch {
	case len(tokens) == 2 && tokens[0] == "channels":
		channel, _ = doc.GetChannels().Get(tokens[1])
	case len(tokens) == 3 && tokens[0] == "components" && tokens[1] == "channels":
		channel, _ = doc.GetComponents().GetChannels().Get(tokens[2])
	}
	return channel
}
