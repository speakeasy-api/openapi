package arazzo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/expression"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type (
	// ReusableParameter represents a parameter that can either be referenced from components or declared inline in a workflow or step.
	ReusableParameter = Reusable[Parameter, *Parameter, core.Parameter]
	// ReusableSuccessAction represents a success action that can either be referenced from components or declared inline in a workflow or step.
	ReusableSuccessAction = Reusable[SuccessAction, *SuccessAction, core.SuccessAction]
	// ReusableFailureAction represents a failure action that can either be referenced from components or declared inline in a workflow or step.
	ReusableFailureAction = Reusable[FailureAction, *FailureAction, core.FailureAction]
)

type Reusable[T any, V validator[T], C any] struct {
	// Reference is the expression to the location of the reusable object.
	Reference *expression.Expression
	// Value is any value provided alongside a parameter reusable object.
	Value Value
	// If this reusable object is not a reference, this will be the inline object for this node.
	Object V

	// Valid indicates whether this reusable model passed validation.
	Valid bool

	core core.Reusable[C]
}

// GetCore will return the low level representation of the reusable object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (r *Reusable[T, V, C]) GetCore() *core.Reusable[C] {
	return &r.core
}

// Get will return either the inline object or the object referenced by the reference.
func (r *Reusable[T, V, C]) Get(components *Components) *T {
	if r.IsReference() {
		return r.GetReferencedObject(components)
	} else {
		return r.Object
	}
}

func (r *Reusable[T, V, C]) IsReference() bool {
	return r.Reference != nil
}

func (r *Reusable[T, V, C]) GetReferencedObject(components *Components) *T {
	if !r.IsReference() {
		return nil
	}

	typ, componentType, references, _ := r.Reference.GetParts()

	if typ != expression.ExpressionTypeComponents {
		return nil
	}

	if componentType == "" || len(references) != 1 {
		return nil
	}

	var component any

	switch componentType {
	case "parameters":
		param, ok := components.Parameters.Get(references[0])
		if !ok {
			return nil
		}
		component = param
	case "successActions":
		successAction, ok := components.SuccessActions.Get(references[0])
		if !ok {
			return nil
		}
		component = successAction
	case "failureActions":
		failureAction, ok := components.FailureActions.Get(references[0])
		if !ok {
			return nil
		}
		component = failureAction
	default:
		return nil
	}

	paramT, ok := component.(*T)
	if !ok {
		return nil
	}
	return paramT
}

// Validate will validate the reusable object against the Arazzo specification.
func (r *Reusable[T, V, C]) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	a := validation.GetContextObject[Arazzo](o)
	if a == nil {
		return []error{
			errors.New("An Arazzo object must be passed via validation options to validate a Reusable Object"),
		}
	}

	errs := []error{}

	switch reflect.TypeOf((*T)(nil)).Elem().Name() {
	case "Parameter":
	default:
		if r.Value != nil {
			errs = append(errs, &validation.Error{
				Message: "value is not allowed when object is not a parameter",
				Line:    r.core.Value.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Value.GetValueNodeOrRoot(r.core.RootNode).Column,
			})
		}
	}

	if r.Reference != nil {
		errs = append(errs, r.validateReference(ctx, a, opts...)...)
	} else if r.Object != nil {
		errs = append(errs, r.Object.Validate(ctx, opts...)...)
	}

	if len(errs) == 0 {
		r.Valid = true
	}

	return errs
}

func (r *Reusable[T, V, C]) validateReference(ctx context.Context, a *Arazzo, opts ...validation.Option) []error {
	if err := r.Reference.Validate(true); err != nil {
		return []error{
			validation.Error{
				Message: err.Error(),
				Line:    r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Column,
			},
		}
	}

	typ, componentType, references, _ := r.Reference.GetParts()

	if typ != expression.ExpressionTypeComponents {
		return []error{
			validation.Error{
				Message: fmt.Sprintf("reference must be a components expression, got %s", r.Reference.GetType()),
				Line:    r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Column,
			},
		}
	}

	if componentType == "" || len(references) != 1 {
		return []error{
			validation.Error{
				Message: fmt.Sprintf("reference must be a components expression with 3 parts, got %s", *r.Reference),
				Line:    r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Column,
			},
		}
	}

	componentName := references[0]

	if a.Components == nil {
		return []error{
			validation.Error{
				Message: fmt.Sprintf("components not present, reference to missing component %s", *r.Reference),
				Line:    r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Column,
			},
		}
	}

	objType := reflect.TypeOf(r.Object).Elem()

	switch componentType {
	case "parameters":
		return validateComponentReference(ctx, validateComponentReferenceArgs[*Parameter]{
			componentType:      componentType,
			componentName:      componentName,
			typ:                objType,
			components:         a.Components.Parameters,
			reference:          r.Reference,
			referenceValueNode: r.core.Reference.GetValueNodeOrRoot(r.core.RootNode),
		}, opts...)
	case "successActions":
		return validateComponentReference(ctx, validateComponentReferenceArgs[*SuccessAction]{
			componentType:      componentType,
			componentName:      componentName,
			typ:                objType,
			components:         a.Components.SuccessActions,
			reference:          r.Reference,
			referenceValueNode: r.core.Reference.GetValueNodeOrRoot(r.core.RootNode),
		}, opts...)
	case "failureActions":
		return validateComponentReference(ctx, validateComponentReferenceArgs[*FailureAction]{
			componentType:      componentType,
			componentName:      componentName,
			typ:                objType,
			components:         a.Components.FailureActions,
			reference:          r.Reference,
			referenceValueNode: r.core.Reference.GetValueNodeOrRoot(r.core.RootNode),
		}, opts...)
	default:
		return []error{
			validation.Error{
				Message: fmt.Sprintf("reference to %s is not valid, valid components are [parameters, successActions, failureActions]", componentType),
				Line:    r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Line,
				Column:  r.core.Reference.GetValueNodeOrRoot(r.core.RootNode).Column,
			},
		}
	}
}

type validateComponentReferenceArgs[T any] struct {
	componentType      string
	componentName      string
	typ                reflect.Type
	components         *sequencedmap.Map[string, T]
	reference          *expression.Expression
	referenceValueNode *yaml.Node
}

func validateComponentReference[T any, V validator[T]](ctx context.Context, args validateComponentReferenceArgs[V], opts ...validation.Option) []error {
	typ := reflect.TypeOf((*T)(nil)).Elem()

	if args.typ != typ {
		return []error{
			validation.Error{
				Message: fmt.Sprintf("expected a %s reference got %s", typeToComponentType(args.typ), args.componentType),
				Line:    args.referenceValueNode.Line,
				Column:  args.referenceValueNode.Column,
			},
		}
	}

	if args.components == nil {
		return []error{
			validation.Error{
				Message: fmt.Sprintf("components.%s not present, reference to missing component %s", args.componentType, *args.reference),
				Line:    args.referenceValueNode.Line,
				Column:  args.referenceValueNode.Column,
			},
		}
	}

	component, ok := args.components.Get(args.componentName)
	if !ok {
		return []error{
			validation.Error{
				Message: fmt.Sprintf("components.%s.%s not present, reference to missing component %s", args.componentType, args.componentName, *args.reference),
				Line:    args.referenceValueNode.Line,
				Column:  args.referenceValueNode.Column,
			},
		}
	}

	return component.Validate(ctx, opts...)
}

func typeToComponentType(typ reflect.Type) string {
	s := typ.Name()

	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:] + "s"
}
