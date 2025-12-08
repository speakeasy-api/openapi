package swagger

import (
	"context"
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/swagger/core"
	"github.com/speakeasy-api/openapi/validation"
)

type (
	// ReferencedParameter represents a parameter that can either be referenced from elsewhere or declared inline.
	ReferencedParameter = Reference[Parameter, *Parameter, *core.Parameter]
	// ReferencedResponse represents a response that can either be referenced from elsewhere or declared inline.
	ReferencedResponse = Reference[Response, *Response, *core.Response]
)

// NewReferencedParameterFromRef creates a new ReferencedParameter from a reference string.
func NewReferencedParameterFromRef(ref references.Reference) *ReferencedParameter {
	return &ReferencedParameter{
		Reference: &ref,
	}
}

// NewReferencedParameterFromParameter creates a new ReferencedParameter from a Parameter.
func NewReferencedParameterFromParameter(parameter *Parameter) *ReferencedParameter {
	return &ReferencedParameter{
		Object: parameter,
	}
}

// NewReferencedResponseFromRef creates a new ReferencedResponse from a reference string.
func NewReferencedResponseFromRef(ref references.Reference) *ReferencedResponse {
	return &ReferencedResponse{
		Reference: &ref,
	}
}

// NewReferencedResponseFromResponse creates a new ReferencedResponse from a Response.
func NewReferencedResponseFromResponse(response *Response) *ReferencedResponse {
	return &ReferencedResponse{
		Object: response,
	}
}

// Reference represents an object that can either be referenced from elsewhere or declared inline.
type Reference[T any, V interfaces.Validator[T], C marshaller.CoreModeler] struct {
	marshaller.Model[core.Reference[C]]

	// Reference is the reference string ($ref).
	Reference *references.Reference

	// If this was an inline object instead of a reference this will contain that object.
	Object *T
}

var _ interfaces.Model[core.Reference[*core.Parameter]] = (*Reference[Parameter, *Parameter, *core.Parameter])(nil)

// IsReference returns true if the reference is a reference (via $ref) to an object as opposed to an inline object.
func (r *Reference[T, V, C]) IsReference() bool {
	if r == nil {
		return false
	}
	return r.Reference != nil
}

// GetReference returns the value of the Reference field. Returns empty string if not set.
func (r *Reference[T, V, C]) GetReference() references.Reference {
	if r == nil || r.Reference == nil {
		return ""
	}
	return *r.Reference
}

// GetObject returns the referenced object. If this is a reference, this will return nil.
func (r *Reference[T, V, C]) GetObject() *T {
	if r == nil {
		return nil
	}

	if r.IsReference() {
		return nil
	}

	return r.Object
}

// Validate validates the Reference object against the Swagger Specification.
func (r *Reference[T, V, C]) Validate(ctx context.Context, opts ...validation.Option) []error {
	if r == nil {
		return []error{errors.New("reference is nil")}
	}

	c := r.GetCore()
	if c == nil {
		return []error{errors.New("reference core is nil")}
	}

	errs := []error{}

	if c.Reference.Present && r.Object != nil {
		// Use the validator interface V to validate the object
		var validator V
		if v, ok := any(r.Object).(V); ok {
			validator = v
			errs = append(errs, validator.Validate(ctx, opts...)...)
		}
	}

	r.Valid = len(errs) == 0 && c.GetValid()

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
		r.Reference = pointer.From(references.Reference(*s.Reference.Value))
	} else {
		if err := marshaller.Populate(s.Object, &r.Object); err != nil {
			return err
		}
	}

	r.SetCore(s)

	return nil
}
