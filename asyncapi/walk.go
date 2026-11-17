package asyncapi

import (
	"context"
	"iter"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	jsonschema "github.com/speakeasy-api/openapi/jsonschema/asyncapi"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	walkpkg "github.com/speakeasy-api/openapi/walk"
)

// WalkItem represents a single item yielded by the Walk iterator.
type WalkItem struct {
	Match    MatchFunc
	Location Locations
	AsyncAPI *AsyncAPI
}

// MatchFunc represents a particular model in the AsyncAPI document that can be matched.
// Pass it a Matcher with the appropriate functions populated to match the model type(s) you are interested in.
type MatchFunc func(Matcher) error

// LocationContext is the context of where an item is located within its parent.
type LocationContext = walkpkg.LocationContext[MatchFunc]

// Locations is the path from the document root to an item, convertible to a JSON pointer.
type Locations = walkpkg.Locations[MatchFunc]

// Matcher is used to match specific models in the AsyncAPI document. Populate the functions for the model types
// you are interested in. Returning walk.ErrTerminate from a function can be used to signal the caller to stop.
//
// Reference wrappers (the Referenced* types) are visited at their location, followed by their inline object if they
// are not a Reference Object. Reference Objects are not resolved, so the objects they point to are visited only at
// their own authored location. Schemas are visited as SchemaUnion values but are not descended into.
type Matcher struct {
	AsyncAPI                        func(*AsyncAPI) error
	Info                            func(*Info) error
	Contact                         func(*Contact) error
	License                         func(*License) error
	Tag                             func(*Tag) error
	ExternalDocumentation           func(*ExternalDocumentation) error
	Server                          func(*Server) error
	ServerVariable                  func(*ServerVariable) error
	Channel                         func(*Channel) error
	Parameter                       func(*Parameter) error
	Operation                       func(*Operation) error
	OperationTrait                  func(*OperationTrait) error
	OperationReply                  func(*OperationReply) error
	OperationReplyAddress           func(*OperationReplyAddress) error
	Message                         func(*Message) error
	MessageTrait                    func(*MessageTrait) error
	MessageExample                  func(*MessageExample) error
	CorrelationID                   func(*CorrelationID) error
	SecurityScheme                  func(*SecurityScheme) error
	OAuthFlows                      func(*OAuthFlows) error
	OAuthFlow                       func(*OAuthFlow) error
	Bindings                        func(*Bindings) error
	Components                      func(*Components) error
	ReferenceObject                 func(*ReferenceObject) error
	SchemaUnion                     func(*jsonschema.SchemaUnion) error
	ReferencedServer                func(*ReferencedServer) error
	ReferencedServerVariable        func(*ReferencedServerVariable) error
	ReferencedChannel               func(*ReferencedChannel) error
	ReferencedParameter             func(*ReferencedParameter) error
	ReferencedOperation             func(*ReferencedOperation) error
	ReferencedOperationTrait        func(*ReferencedOperationTrait) error
	ReferencedOperationReply        func(*ReferencedOperationReply) error
	ReferencedOperationReplyAddress func(*ReferencedOperationReplyAddress) error
	ReferencedMessage               func(*ReferencedMessage) error
	ReferencedMessageTrait          func(*ReferencedMessageTrait) error
	ReferencedCorrelationID         func(*ReferencedCorrelationID) error
	ReferencedSecurityScheme        func(*ReferencedSecurityScheme) error
	ReferencedBindings              func(*ReferencedBindings) error
	Extensions                      func(*extensions.Extensions) error
	Any                             func(any) error // Any will be called along with the other functions above on a match of a model
}

// Walk returns an iterator that yields an item for each model in the AsyncAPI document, in document order.
// Callers can break out of the loop at any time to stop walking. Walking never reads files or makes network requests.
func Walk(ctx context.Context, asyncAPI *AsyncAPI) iter.Seq[WalkItem] {
	return func(yield func(WalkItem) bool) {
		if asyncAPI == nil {
			return
		}
		w := walker{doc: asyncAPI, yield: yield, cancelled: func() bool { return false }}
		if ctx != nil {
			w.cancelled = func() bool { return ctx.Err() != nil }
		}
		w.walkAsyncAPI(asyncAPI)
	}
}

type walker struct {
	doc       *AsyncAPI
	yield     func(WalkItem) bool
	cancelled func() bool
}

// visit yields an item, returning false if walking should stop.
func (w *walker) visit(match MatchFunc, loc Locations) bool {
	if w.cancelled() {
		return false
	}
	return w.yield(WalkItem{Match: match, Location: loc, AsyncAPI: w.doc})
}

// child returns a new location for a field of the parent, without sharing the parent's backing array.
func child(loc Locations, parent MatchFunc, field string) Locations {
	out := make(Locations, len(loc), len(loc)+1)
	copy(out, loc)
	return append(out, LocationContext{ParentMatchFunc: parent, ParentField: field})
}

func childKey(loc Locations, parent MatchFunc, field, key string) Locations {
	out := child(loc, parent, field)
	out[len(out)-1].ParentKey = pointer.From(key)
	return out
}

func childIndex(loc Locations, parent MatchFunc, field string, index int) Locations {
	out := child(loc, parent, field)
	out[len(out)-1].ParentIndex = pointer.From(index)
	return out
}

func walkMap[V any](m *sequencedmap.Map[string, V], loc Locations, parent MatchFunc, field string, each func(V, Locations) bool) bool {
	if m == nil {
		return true
	}
	for key, value := range m.All() {
		if !each(value, childKey(loc, parent, field, key)) {
			return false
		}
	}
	return true
}

func walkSlice[V any](s []V, loc Locations, parent MatchFunc, field string, each func(V, Locations) bool) bool {
	for i, value := range s {
		if !each(value, childIndex(loc, parent, field, i)) {
			return false
		}
	}
	return true
}

// walkReference visits a reference wrapper and, if it holds an inline object, that object at the same location.
func walkReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](w *walker, ref *Reference[T, V, C], loc Locations, walkObject func(*T, Locations) bool) bool {
	if ref == nil {
		return true
	}
	if !w.visit(getMatchFunc(ref), loc) {
		return false
	}
	if ref.IsReference() || ref.Object == nil {
		return true
	}
	return walkObject(ref.Object, loc)
}

func (w *walker) walkExtensions(ext *extensions.Extensions, loc Locations, parent MatchFunc) bool {
	return w.visit(getMatchFunc(ext), child(loc, parent, ""))
}

func (w *walker) walkAsyncAPI(a *AsyncAPI) bool {
	mf := getMatchFunc(a)
	if !w.visit(mf, nil) {
		return false
	}

	loc := Locations{}
	if !w.walkInfo(&a.Info, child(loc, mf, "info")) {
		return false
	}
	if !walkMap(a.Servers, loc, mf, "servers", w.walkReferencedServer) {
		return false
	}
	if !walkMap(a.Channels, loc, mf, "channels", w.walkReferencedChannel) {
		return false
	}
	if !walkMap(a.Operations, loc, mf, "operations", w.walkReferencedOperation) {
		return false
	}
	if a.Components != nil && !w.walkComponents(a.Components, child(loc, mf, "components")) {
		return false
	}
	return w.walkExtensions(a.Extensions, loc, mf)
}

func (w *walker) walkInfo(info *Info, loc Locations) bool {
	mf := getMatchFunc(info)
	if !w.visit(mf, loc) {
		return false
	}
	if info.Contact != nil && !walkSimple(w, info.Contact, info.Contact.Extensions, child(loc, mf, "contact")) {
		return false
	}
	if info.License != nil && !walkSimple(w, info.License, info.License.Extensions, child(loc, mf, "license")) {
		return false
	}
	if !walkSlice(info.Tags, loc, mf, "tags", w.walkTag) {
		return false
	}
	if !w.walkExternalDocs(info.ExternalDocs, child(loc, mf, "externalDocs")) {
		return false
	}
	return w.walkExtensions(info.Extensions, loc, mf)
}

// walkSimple visits a model with no children other than its extensions.
func walkSimple[T any](w *walker, model *T, ext *extensions.Extensions, loc Locations) bool {
	if model == nil {
		return true
	}
	mf := getMatchFunc(model)
	if !w.visit(mf, loc) {
		return false
	}
	return w.walkExtensions(ext, loc, mf)
}

func (w *walker) walkTag(tag *Tag, loc Locations) bool {
	if tag == nil {
		return true
	}
	mf := getMatchFunc(tag)
	if !w.visit(mf, loc) {
		return false
	}
	if !w.walkExternalDocs(tag.ExternalDocs, child(loc, mf, "externalDocs")) {
		return false
	}
	return w.walkExtensions(tag.Extensions, loc, mf)
}

func (w *walker) walkExternalDocs(docs *ExternalDocumentation, loc Locations) bool {
	if docs == nil {
		return true
	}
	mf := getMatchFunc(docs)
	if !w.visit(mf, loc) {
		return false
	}
	return w.walkExtensions(docs.Extensions, loc, mf)
}

func (w *walker) walkReferenceObject(ref *ReferenceObject, loc Locations) bool {
	if ref == nil {
		return true
	}
	return w.visit(getMatchFunc(ref), loc)
}

func (w *walker) walkSchema(schema *jsonschema.SchemaUnion, loc Locations) bool {
	if schema == nil {
		return true
	}
	return w.visit(getMatchFunc(schema), loc)
}

func (w *walker) walkReferencedServer(ref *ReferencedServer, loc Locations) bool {
	return walkReference(w, ref, loc, w.walkServer)
}

func (w *walker) walkServer(server *Server, loc Locations) bool {
	mf := getMatchFunc(server)
	if !w.visit(mf, loc) {
		return false
	}
	if !walkMap(server.Variables, loc, mf, "variables", w.walkReferencedServerVariable) {
		return false
	}
	if !walkSlice(server.Security, loc, mf, "security", w.walkReferencedSecurityScheme) {
		return false
	}
	if !walkSlice(server.Tags, loc, mf, "tags", w.walkTag) {
		return false
	}
	if !w.walkExternalDocs(server.ExternalDocs, child(loc, mf, "externalDocs")) {
		return false
	}
	if !w.walkReferencedBindings(server.Bindings, child(loc, mf, "bindings")) {
		return false
	}
	return w.walkExtensions(server.Extensions, loc, mf)
}

func (w *walker) walkReferencedServerVariable(ref *ReferencedServerVariable, loc Locations) bool {
	return walkReference(w, ref, loc, func(v *ServerVariable, loc Locations) bool {
		return walkSimple(w, v, v.Extensions, loc)
	})
}

func (w *walker) walkReferencedChannel(ref *ReferencedChannel, loc Locations) bool {
	return walkReference(w, ref, loc, w.walkChannel)
}

func (w *walker) walkChannel(channel *Channel, loc Locations) bool {
	mf := getMatchFunc(channel)
	if !w.visit(mf, loc) {
		return false
	}
	if !walkMap(channel.Messages, loc, mf, "messages", w.walkReferencedMessage) {
		return false
	}
	if !walkSlice(channel.Servers, loc, mf, "servers", w.walkReferenceObject) {
		return false
	}
	if !walkMap(channel.Parameters, loc, mf, "parameters", w.walkReferencedParameter) {
		return false
	}
	if !walkSlice(channel.Tags, loc, mf, "tags", w.walkTag) {
		return false
	}
	if !w.walkExternalDocs(channel.ExternalDocs, child(loc, mf, "externalDocs")) {
		return false
	}
	if !w.walkReferencedBindings(channel.Bindings, child(loc, mf, "bindings")) {
		return false
	}
	return w.walkExtensions(channel.Extensions, loc, mf)
}

func (w *walker) walkReferencedParameter(ref *ReferencedParameter, loc Locations) bool {
	return walkReference(w, ref, loc, func(p *Parameter, loc Locations) bool {
		return walkSimple(w, p, p.Extensions, loc)
	})
}

func (w *walker) walkReferencedOperation(ref *ReferencedOperation, loc Locations) bool {
	return walkReference(w, ref, loc, w.walkOperation)
}

func (w *walker) walkOperation(operation *Operation, loc Locations) bool {
	mf := getMatchFunc(operation)
	if !w.visit(mf, loc) {
		return false
	}
	// Channel is a required value field, so skip it when the document omits it rather than visiting an empty object.
	if operation.Channel.Reference != "" || operation.Channel.GetCore().GetRootNode() != nil {
		if !w.walkReferenceObject(&operation.Channel, child(loc, mf, "channel")) {
			return false
		}
	}
	if !w.walkOperationFields(operation.Security, operation.Tags, operation.ExternalDocs, operation.Bindings, nil, loc, mf) {
		return false
	}
	if !walkSlice(operation.Traits, loc, mf, "traits", w.walkReferencedOperationTrait) {
		return false
	}
	if !walkSlice(operation.Messages, loc, mf, "messages", w.walkReferenceObject) {
		return false
	}
	if !w.walkReferencedOperationReply(operation.Reply, child(loc, mf, "reply")) {
		return false
	}
	return w.walkExtensions(operation.Extensions, loc, mf)
}

func (w *walker) walkOperationFields(security []*ReferencedSecurityScheme, tags []*Tag, externalDocs *ExternalDocumentation, bindings *ReferencedBindings, reply *ReferencedOperationReply, loc Locations, mf MatchFunc) bool {
	if !walkSlice(security, loc, mf, "security", w.walkReferencedSecurityScheme) {
		return false
	}
	if !walkSlice(tags, loc, mf, "tags", w.walkTag) {
		return false
	}
	if !w.walkExternalDocs(externalDocs, child(loc, mf, "externalDocs")) {
		return false
	}
	if !w.walkReferencedBindings(bindings, child(loc, mf, "bindings")) {
		return false
	}
	return w.walkReferencedOperationReply(reply, child(loc, mf, "reply"))
}

func (w *walker) walkReferencedOperationTrait(ref *ReferencedOperationTrait, loc Locations) bool {
	return walkReference(w, ref, loc, func(trait *OperationTrait, loc Locations) bool {
		mf := getMatchFunc(trait)
		if !w.visit(mf, loc) {
			return false
		}
		if !w.walkOperationFields(trait.Security, trait.Tags, trait.ExternalDocs, trait.Bindings, trait.Reply, loc, mf) {
			return false
		}
		return w.walkExtensions(trait.Extensions, loc, mf)
	})
}

func (w *walker) walkReferencedOperationReply(ref *ReferencedOperationReply, loc Locations) bool {
	return walkReference(w, ref, loc, func(reply *OperationReply, loc Locations) bool {
		mf := getMatchFunc(reply)
		if !w.visit(mf, loc) {
			return false
		}
		if !w.walkReferencedOperationReplyAddress(reply.Address, child(loc, mf, "address")) {
			return false
		}
		if !w.walkReferenceObject(reply.Channel, child(loc, mf, "channel")) {
			return false
		}
		if !walkSlice(reply.Messages, loc, mf, "messages", w.walkReferenceObject) {
			return false
		}
		return w.walkExtensions(reply.Extensions, loc, mf)
	})
}

func (w *walker) walkReferencedOperationReplyAddress(ref *ReferencedOperationReplyAddress, loc Locations) bool {
	return walkReference(w, ref, loc, func(address *OperationReplyAddress, loc Locations) bool {
		return walkSimple(w, address, address.Extensions, loc)
	})
}

func (w *walker) walkReferencedMessage(ref *ReferencedMessage, loc Locations) bool {
	return walkReference(w, ref, loc, w.walkMessage)
}

func (w *walker) walkMessage(message *Message, loc Locations) bool {
	mf := getMatchFunc(message)
	if !w.visit(mf, loc) {
		return false
	}
	if !w.walkSchema(message.Headers, child(loc, mf, "headers")) {
		return false
	}
	if !w.walkSchema(message.Payload, child(loc, mf, "payload")) {
		return false
	}
	if !w.walkMessageFields(message.CorrelationID, message.Tags, message.ExternalDocs, message.Bindings, message.Examples, loc, mf) {
		return false
	}
	if !walkSlice(message.Traits, loc, mf, "traits", w.walkReferencedMessageTrait) {
		return false
	}
	return w.walkExtensions(message.Extensions, loc, mf)
}

func (w *walker) walkMessageFields(correlationID *ReferencedCorrelationID, tags []*Tag, externalDocs *ExternalDocumentation, bindings *ReferencedBindings, examples []*MessageExample, loc Locations, mf MatchFunc) bool {
	if !w.walkReferencedCorrelationID(correlationID, child(loc, mf, "correlationId")) {
		return false
	}
	if !walkSlice(tags, loc, mf, "tags", w.walkTag) {
		return false
	}
	if !w.walkExternalDocs(externalDocs, child(loc, mf, "externalDocs")) {
		return false
	}
	if !w.walkReferencedBindings(bindings, child(loc, mf, "bindings")) {
		return false
	}
	return walkSlice(examples, loc, mf, "examples", func(example *MessageExample, loc Locations) bool {
		if example == nil {
			return true
		}
		return walkSimple(w, example, example.Extensions, loc)
	})
}

func (w *walker) walkReferencedMessageTrait(ref *ReferencedMessageTrait, loc Locations) bool {
	return walkReference(w, ref, loc, func(trait *MessageTrait, loc Locations) bool {
		mf := getMatchFunc(trait)
		if !w.visit(mf, loc) {
			return false
		}
		if !w.walkSchema(trait.Headers, child(loc, mf, "headers")) {
			return false
		}
		if !w.walkMessageFields(trait.CorrelationID, trait.Tags, trait.ExternalDocs, trait.Bindings, trait.Examples, loc, mf) {
			return false
		}
		return w.walkExtensions(trait.Extensions, loc, mf)
	})
}

func (w *walker) walkReferencedCorrelationID(ref *ReferencedCorrelationID, loc Locations) bool {
	return walkReference(w, ref, loc, func(id *CorrelationID, loc Locations) bool {
		return walkSimple(w, id, id.Extensions, loc)
	})
}

func (w *walker) walkReferencedSecurityScheme(ref *ReferencedSecurityScheme, loc Locations) bool {
	return walkReference(w, ref, loc, func(scheme *SecurityScheme, loc Locations) bool {
		mf := getMatchFunc(scheme)
		if !w.visit(mf, loc) {
			return false
		}
		if flows := scheme.Flows; flows != nil {
			flowsLoc := child(loc, mf, "flows")
			flowsMF := getMatchFunc(flows)
			if !w.visit(flowsMF, flowsLoc) {
				return false
			}
			for _, f := range []struct {
				name string
				flow *OAuthFlow
			}{
				{"implicit", flows.Implicit},
				{"password", flows.Password},
				{"clientCredentials", flows.ClientCredentials},
				{"authorizationCode", flows.AuthorizationCode},
			} {
				if f.flow != nil && !walkSimple(w, f.flow, f.flow.Extensions, child(flowsLoc, flowsMF, f.name)) {
					return false
				}
			}
			if !w.walkExtensions(flows.Extensions, flowsLoc, flowsMF) {
				return false
			}
		}
		return w.walkExtensions(scheme.Extensions, loc, mf)
	})
}

func (w *walker) walkReferencedBindings(ref *ReferencedBindings, loc Locations) bool {
	return walkReference(w, ref, loc, func(bindings *Bindings, loc Locations) bool {
		return walkSimple(w, bindings, bindings.Extensions, loc)
	})
}

func (w *walker) walkComponents(c *Components, loc Locations) bool {
	mf := getMatchFunc(c)
	if !w.visit(mf, loc) {
		return false
	}

	steps := []func() bool{
		func() bool { return walkMap(c.Schemas, loc, mf, "schemas", w.walkSchema) },
		func() bool { return walkMap(c.Servers, loc, mf, "servers", w.walkReferencedServer) },
		func() bool { return walkMap(c.Channels, loc, mf, "channels", w.walkReferencedChannel) },
		func() bool { return walkMap(c.Operations, loc, mf, "operations", w.walkReferencedOperation) },
		func() bool { return walkMap(c.Messages, loc, mf, "messages", w.walkReferencedMessage) },
		func() bool {
			return walkMap(c.SecuritySchemes, loc, mf, "securitySchemes", w.walkReferencedSecurityScheme)
		},
		func() bool {
			return walkMap(c.ServerVariables, loc, mf, "serverVariables", w.walkReferencedServerVariable)
		},
		func() bool { return walkMap(c.Parameters, loc, mf, "parameters", w.walkReferencedParameter) },
		func() bool {
			return walkMap(c.CorrelationIDs, loc, mf, "correlationIds", w.walkReferencedCorrelationID)
		},
		func() bool { return walkMap(c.Replies, loc, mf, "replies", w.walkReferencedOperationReply) },
		func() bool {
			return walkMap(c.ReplyAddresses, loc, mf, "replyAddresses", w.walkReferencedOperationReplyAddress)
		},
		func() bool { return walkMap(c.ExternalDocs, loc, mf, "externalDocs", w.walkExternalDocs) },
		func() bool { return walkMap(c.Tags, loc, mf, "tags", w.walkTag) },
		func() bool {
			return walkMap(c.OperationTraits, loc, mf, "operationTraits", w.walkReferencedOperationTrait)
		},
		func() bool {
			return walkMap(c.MessageTraits, loc, mf, "messageTraits", w.walkReferencedMessageTrait)
		},
		func() bool { return walkMap(c.ServerBindings, loc, mf, "serverBindings", w.walkReferencedBindings) },
		func() bool { return walkMap(c.ChannelBindings, loc, mf, "channelBindings", w.walkReferencedBindings) },
		func() bool {
			return walkMap(c.OperationBindings, loc, mf, "operationBindings", w.walkReferencedBindings)
		},
		func() bool { return walkMap(c.MessageBindings, loc, mf, "messageBindings", w.walkReferencedBindings) },
	}
	for _, step := range steps {
		if !step() {
			return false
		}
	}

	return w.walkExtensions(c.Extensions, loc, mf)
}

type matchHandler[T any] struct {
	GetSpecific func(m Matcher) func(*T) error
}

func handle[T any](get func(m Matcher) func(*T) error) any {
	return matchHandler[T]{GetSpecific: get}
}

var matchRegistry = map[reflect.Type]any{
	reflect.TypeFor[*AsyncAPI]():              handle(func(m Matcher) func(*AsyncAPI) error { return m.AsyncAPI }),
	reflect.TypeFor[*Info]():                  handle(func(m Matcher) func(*Info) error { return m.Info }),
	reflect.TypeFor[*Contact]():               handle(func(m Matcher) func(*Contact) error { return m.Contact }),
	reflect.TypeFor[*License]():               handle(func(m Matcher) func(*License) error { return m.License }),
	reflect.TypeFor[*Tag]():                   handle(func(m Matcher) func(*Tag) error { return m.Tag }),
	reflect.TypeFor[*ExternalDocumentation](): handle(func(m Matcher) func(*ExternalDocumentation) error { return m.ExternalDocumentation }),
	reflect.TypeFor[*Server]():                handle(func(m Matcher) func(*Server) error { return m.Server }),
	reflect.TypeFor[*ServerVariable]():        handle(func(m Matcher) func(*ServerVariable) error { return m.ServerVariable }),
	reflect.TypeFor[*Channel]():               handle(func(m Matcher) func(*Channel) error { return m.Channel }),
	reflect.TypeFor[*Parameter]():             handle(func(m Matcher) func(*Parameter) error { return m.Parameter }),
	reflect.TypeFor[*Operation]():             handle(func(m Matcher) func(*Operation) error { return m.Operation }),
	reflect.TypeFor[*OperationTrait]():        handle(func(m Matcher) func(*OperationTrait) error { return m.OperationTrait }),
	reflect.TypeFor[*OperationReply]():        handle(func(m Matcher) func(*OperationReply) error { return m.OperationReply }),
	reflect.TypeFor[*OperationReplyAddress](): handle(func(m Matcher) func(*OperationReplyAddress) error { return m.OperationReplyAddress }),
	reflect.TypeFor[*Message]():               handle(func(m Matcher) func(*Message) error { return m.Message }),
	reflect.TypeFor[*MessageTrait]():          handle(func(m Matcher) func(*MessageTrait) error { return m.MessageTrait }),
	reflect.TypeFor[*MessageExample]():        handle(func(m Matcher) func(*MessageExample) error { return m.MessageExample }),
	reflect.TypeFor[*CorrelationID]():         handle(func(m Matcher) func(*CorrelationID) error { return m.CorrelationID }),
	reflect.TypeFor[*SecurityScheme]():        handle(func(m Matcher) func(*SecurityScheme) error { return m.SecurityScheme }),
	reflect.TypeFor[*OAuthFlows]():            handle(func(m Matcher) func(*OAuthFlows) error { return m.OAuthFlows }),
	reflect.TypeFor[*OAuthFlow]():             handle(func(m Matcher) func(*OAuthFlow) error { return m.OAuthFlow }),
	reflect.TypeFor[*Bindings]():              handle(func(m Matcher) func(*Bindings) error { return m.Bindings }),
	reflect.TypeFor[*Components]():            handle(func(m Matcher) func(*Components) error { return m.Components }),
	reflect.TypeFor[*ReferenceObject]():       handle(func(m Matcher) func(*ReferenceObject) error { return m.ReferenceObject }),
	reflect.TypeFor[*jsonschema.SchemaUnion](): handle(func(m Matcher) func(*jsonschema.SchemaUnion) error {
		return m.SchemaUnion
	}),
	reflect.TypeFor[*ReferencedServer](): handle(func(m Matcher) func(*ReferencedServer) error { return m.ReferencedServer }),
	reflect.TypeFor[*ReferencedServerVariable](): handle(func(m Matcher) func(*ReferencedServerVariable) error {
		return m.ReferencedServerVariable
	}),
	reflect.TypeFor[*ReferencedChannel](): handle(func(m Matcher) func(*ReferencedChannel) error { return m.ReferencedChannel }),
	reflect.TypeFor[*ReferencedParameter](): handle(func(m Matcher) func(*ReferencedParameter) error {
		return m.ReferencedParameter
	}),
	reflect.TypeFor[*ReferencedOperation](): handle(func(m Matcher) func(*ReferencedOperation) error {
		return m.ReferencedOperation
	}),
	reflect.TypeFor[*ReferencedOperationTrait](): handle(func(m Matcher) func(*ReferencedOperationTrait) error {
		return m.ReferencedOperationTrait
	}),
	reflect.TypeFor[*ReferencedOperationReply](): handle(func(m Matcher) func(*ReferencedOperationReply) error {
		return m.ReferencedOperationReply
	}),
	reflect.TypeFor[*ReferencedOperationReplyAddress](): handle(func(m Matcher) func(*ReferencedOperationReplyAddress) error {
		return m.ReferencedOperationReplyAddress
	}),
	reflect.TypeFor[*ReferencedMessage](): handle(func(m Matcher) func(*ReferencedMessage) error { return m.ReferencedMessage }),
	reflect.TypeFor[*ReferencedMessageTrait](): handle(func(m Matcher) func(*ReferencedMessageTrait) error {
		return m.ReferencedMessageTrait
	}),
	reflect.TypeFor[*ReferencedCorrelationID](): handle(func(m Matcher) func(*ReferencedCorrelationID) error {
		return m.ReferencedCorrelationID
	}),
	reflect.TypeFor[*ReferencedSecurityScheme](): handle(func(m Matcher) func(*ReferencedSecurityScheme) error {
		return m.ReferencedSecurityScheme
	}),
	reflect.TypeFor[*ReferencedBindings](): handle(func(m Matcher) func(*ReferencedBindings) error {
		return m.ReferencedBindings
	}),
	reflect.TypeFor[*extensions.Extensions](): handle(func(m Matcher) func(*extensions.Extensions) error { return m.Extensions }),
}

func getMatchFunc[T any](target *T) MatchFunc {
	if h, ok := matchRegistry[reflect.TypeOf(target)]; ok {
		if handler, ok := h.(matchHandler[T]); ok {
			return func(m Matcher) error {
				if m.Any != nil {
					if err := m.Any(target); err != nil {
						return err
					}
				}
				if specific := handler.GetSpecific(m); specific != nil {
					return specific(target)
				}
				return nil
			}
		}
	}

	return func(m Matcher) error {
		if m.Any != nil {
			return m.Any(target)
		}
		return nil
	}
}
