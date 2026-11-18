package asyncapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/asyncapi/core"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
)

type (
	// ReferencedServer is a Server Object or a Reference Object to one.
	ReferencedServer = Reference[Server, *Server, *core.Server]
	// ReferencedServerVariable is a Server Variable Object or a Reference Object to one.
	ReferencedServerVariable = Reference[ServerVariable, *ServerVariable, *core.ServerVariable]
	// ReferencedChannel is a Channel Object or a Reference Object to one.
	ReferencedChannel = Reference[Channel, *Channel, *core.Channel]
	// ReferencedParameter is a Parameter Object or a Reference Object to one.
	ReferencedParameter = Reference[Parameter, *Parameter, *core.Parameter]
	// ReferencedOperation is an Operation Object or a Reference Object to one.
	ReferencedOperation = Reference[Operation, *Operation, *core.Operation]
	// ReferencedOperationTrait is an Operation Trait Object or a Reference Object to one.
	ReferencedOperationTrait = Reference[OperationTrait, *OperationTrait, *core.OperationTrait]
	// ReferencedOperationReply is an Operation Reply Object or a Reference Object to one.
	ReferencedOperationReply = Reference[OperationReply, *OperationReply, *core.OperationReply]
	// ReferencedOperationReplyAddress is an Operation Reply Address Object or a Reference Object to one.
	ReferencedOperationReplyAddress = Reference[OperationReplyAddress, *OperationReplyAddress, *core.OperationReplyAddress]
	// ReferencedMessage is a Message Object or a Reference Object to one.
	ReferencedMessage = Reference[Message, *Message, *core.Message]
	// ReferencedMessageTrait is a Message Trait Object or a Reference Object to one.
	ReferencedMessageTrait = Reference[MessageTrait, *MessageTrait, *core.MessageTrait]
	// ReferencedCorrelationID is a Correlation ID Object or a Reference Object to one.
	ReferencedCorrelationID = Reference[CorrelationID, *CorrelationID, *core.CorrelationID]
	// ReferencedSecurityScheme is a Security Scheme Object or a Reference Object to one.
	ReferencedSecurityScheme = Reference[SecurityScheme, *SecurityScheme, *core.SecurityScheme]
	// ReferencedBindings is a Bindings Object or a Reference Object to one.
	ReferencedBindings = Reference[Bindings, *Bindings, *core.Bindings]
)

// Reference is either an inline object or an AsyncAPI Reference Object to one.
//
// Reference Objects are resolved explicitly by AsyncAPI.ResolveAllReferences. Until then GetObject only returns
// inline objects.
type Reference[T any, V interfaces.Validator[T], C marshaller.CoreModeler] struct {
	marshaller.Model[core.Reference[C]]

	// Reference is the URI of the referenced object. Set when this is a Reference Object.
	Reference *references.Reference
	// Object is the inline object. Set when this is not a Reference Object.
	Object *T

	// resolved and resolvedRef are set by the last call to AsyncAPI.ResolveAllReferences, guarded by resolutionMu.
	resolved    *T
	resolvedRef references.Reference
}

var _ interfaces.Model[core.Reference[*core.Server]] = (*ReferencedServer)(nil)

// IsReference returns true if this is a Reference Object rather than an inline object.
func (r *Reference[T, V, C]) IsReference() bool {
	return r != nil && r.Reference != nil
}

// GetReference returns the reference, or an empty reference if this is an inline object.
func (r *Reference[T, V, C]) GetReference() references.Reference {
	if r == nil || r.Reference == nil {
		return ""
	}
	return *r.Reference
}

// GetObject returns the inline object. For a Reference Object it returns the object it resolved to in the last call
// to AsyncAPI.ResolveAllReferences, following any chain of references, or nil if it hasn't been resolved.
func (r *Reference[T, V, C]) GetObject() *T {
	if r == nil {
		return nil
	}
	if !r.IsReference() {
		return r.Object
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	return r.resolved
}

// IsResolved returns true if this is an inline object, or a Reference Object that has been resolved.
func (r *Reference[T, V, C]) IsResolved() bool {
	return r != nil && r.GetObject() != nil
}

// GetResolvedReference returns the absolute location of the object a Reference Object resolved to, after following
// any chain of references. It is empty for inline objects and unresolved references.
func (r *Reference[T, V, C]) GetResolvedReference() references.Reference {
	if r == nil || !r.IsReference() {
		return ""
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	return r.resolvedRef
}

// GetNavigableNode returns the inline or resolved object so JSON pointers can navigate through references.
func (r *Reference[T, V, C]) GetNavigableNode() (any, error) {
	obj := r.GetObject()
	if obj == nil {
		return nil, fmt.Errorf("reference `%s` is not resolved", r.GetReference())
	}
	return obj, nil
}

func (r *Reference[T, V, C]) setResolved(obj *T, ref references.Reference) {
	resolutionMu.Lock()
	defer resolutionMu.Unlock()
	r.resolved = obj
	r.resolvedRef = ref
}

// Validate validates the reference, or the inline object.
func (r *Reference[T, V, C]) Validate(ctx context.Context, opts ...validation.Option) []error {
	if r == nil {
		return []error{errors.New("reference is nil")}
	}

	c := r.GetCore()
	errs := []error{}

	switch {
	case r.Reference != nil:
		if *r.Reference == "" {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`$ref` must not be empty"), c, c.Reference))
		} else if err := r.Reference.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("reference.$ref is invalid: %w", err), c, c.Reference))
		}
	case r.Object != nil:
		if v, ok := any(r.Object).(V); ok {
			errs = append(errs, v.Validate(ctx, opts...)...)
		}
	}

	r.Valid = noErrors(errs) && coreValid(c)

	return errs
}

func (r *Reference[T, V, C]) Populate(source any) error {
	var s *core.Reference[C]
	switch src := source.(type) {
	case *core.Reference[C]:
		s = src
	case core.Reference[C]:
		s = &src
	default:
		return fmt.Errorf("expected *core.Reference[C] or core.Reference[C], got %T", source)
	}

	if s.Reference.Present {
		// `$ref: null` is kept as an empty reference so validation reports it instead of population panicking.
		ref := ""
		if s.Reference.Value != nil {
			ref = *s.Reference.Value
		}
		r.Reference = pointer.From(references.Reference(ref))
	} else if err := marshaller.PopulateWithContext(s.Object, &r.Object, nil); err != nil {
		return err
	}

	r.SetCore(s)

	return nil
}

// ReferenceObject is a field that must be an AsyncAPI Reference Object, such as an operation's channel.
type ReferenceObject struct {
	marshaller.Model[core.ReferenceObject]

	// Reference is the URI of the referenced object.
	Reference references.Reference

	// resolved and resolvedRef are set by the last call to AsyncAPI.ResolveAllReferences, guarded by resolutionMu.
	resolved    any
	resolvedRef references.Reference
}

// GetResolvedObject returns the object the reference resolved to in the last call to AsyncAPI.ResolveAllReferences:
// a *Channel, *Message or *Server depending on where the Reference Object is used. Returns nil if it hasn't been
// resolved.
func (r *ReferenceObject) GetResolvedObject() any {
	if r == nil {
		return nil
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	return r.resolved
}

// GetResolvedReference returns the absolute location of the resolved object, or an empty reference if unresolved.
func (r *ReferenceObject) GetResolvedReference() references.Reference {
	if r == nil {
		return ""
	}
	resolutionMu.RLock()
	defer resolutionMu.RUnlock()
	return r.resolvedRef
}

// IsResolved returns true if the reference has been resolved.
func (r *ReferenceObject) IsResolved() bool {
	return r.GetResolvedObject() != nil
}

// GetChannel returns the resolved channel, or nil if unresolved or the reference is not to a channel.
func (r *ReferenceObject) GetChannel() *Channel {
	channel, _ := r.GetResolvedObject().(*Channel)
	return channel
}

// GetMessage returns the resolved message, or nil if unresolved or the reference is not to a message.
func (r *ReferenceObject) GetMessage() *Message {
	message, _ := r.GetResolvedObject().(*Message)
	return message
}

// GetServer returns the resolved server, or nil if unresolved or the reference is not to a server.
func (r *ReferenceObject) GetServer() *Server {
	server, _ := r.GetResolvedObject().(*Server)
	return server
}

func (r *ReferenceObject) setResolved(obj any, ref references.Reference) {
	resolutionMu.Lock()
	defer resolutionMu.Unlock()
	r.resolved = obj
	r.resolvedRef = ref
}

var _ interfaces.Model[core.ReferenceObject] = (*ReferenceObject)(nil)

// NewReferenceObject creates a Reference Object for the provided reference.
func NewReferenceObject(ref references.Reference) *ReferenceObject {
	return &ReferenceObject{Reference: ref}
}

// GetReference returns the reference, or an empty reference if not set.
func (r *ReferenceObject) GetReference() references.Reference {
	if r == nil {
		return ""
	}
	return r.Reference
}

// Validate validates the Reference Object.
func (r *ReferenceObject) Validate(_ context.Context, _ ...validation.Option) []error {
	if r == nil {
		return nil
	}

	c := r.GetCore()
	errs := []error{}

	if r.Reference == "" {
		if c.Reference.Present {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`$ref` is required"), c, c.Reference))
		}
	} else if err := r.Reference.Validate(); err != nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, fmt.Errorf("reference.$ref is invalid: %w", err), c, c.Reference))
	}

	r.Valid = noErrors(errs) && coreValid(c)

	return errs
}
