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
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

type (
	// ReusableParameter represents a parameter that can either be referenced from components or declared inline in a workflow or step.
	ReusableParameter = Reusable[Parameter, *Parameter, *core.Parameter]
	// ReusableSuccessAction represents a success action that can either be referenced from components or declared inline in a workflow or step.
	ReusableSuccessAction = Reusable[SuccessAction, *SuccessAction, *core.SuccessAction]
	// ReusableFailureAction represents a failure action that can either be referenced from components or declared inline in a workflow or step.
	ReusableFailureAction = Reusable[FailureAction, *FailureAction, *core.FailureAction]
)

type Reusable[T any, V interfaces.Validator[T], C marshaller.CoreModeler] struct {
	// Reference is the expression to the location of the reusable object.
	Reference *expression.Expression
	// Value is any value provided alongside a parameter reusable object.
	Value Value
	// If this reusable object is not a reference, this will be the inline object for this node.
	Object V

	marshaller.Model[core.Reusable[C]]
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
	core := r.GetCore()

	switch reflect.TypeOf((*T)(nil)).Elem().Name() {
	case "Parameter":
	default:
		if r.Value != nil {
			errs = append(errs, validation.NewValueError("value is not allowed when object is not a parameter", core, core.Value))
		}
	}

	if r.Reference != nil {
		errs = append(errs, r.validateReference(ctx, a, opts...)...)
	} else if r.Object != nil {
		errs = append(errs, r.Object.Validate(ctx, opts...)...)
	}

	r.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func (r *Reusable[T, V, C]) validateReference(ctx context.Context, a *Arazzo, opts ...validation.Option) []error {
	core := r.GetCore()
	if err := r.Reference.Validate(true); err != nil {
		return []error{
			validation.NewValueError(err.Error(), core, core.Reference),
		}
	}

	typ, componentType, references, _ := r.Reference.GetParts()

	if typ != expression.ExpressionTypeComponents {
		return []error{
			validation.NewValueError(fmt.Sprintf("reference must be a components expression, got %s", r.Reference.GetType()), core, core.Reference),
		}
	}

	if componentType == "" || len(references) != 1 {
		return []error{
			validation.NewValueError(fmt.Sprintf("reference must be a components expression with 3 parts, got %s", *r.Reference), core, core.Reference),
		}
	}

	componentName := references[0]

	if a.Components == nil {
		return []error{
			validation.NewValueError(fmt.Sprintf("components not present, reference to missing component %s", *r.Reference), core, core.Reference),
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
			referenceValueNode: core.Reference.GetValueNodeOrRoot(core.RootNode),
		}, opts...)
	case "successActions":
		return validateComponentReference(ctx, validateComponentReferenceArgs[*SuccessAction]{
			componentType:      componentType,
			componentName:      componentName,
			typ:                objType,
			components:         a.Components.SuccessActions,
			reference:          r.Reference,
			referenceValueNode: core.Reference.GetValueNodeOrRoot(core.RootNode),
		}, opts...)
	case "failureActions":
		return validateComponentReference(ctx, validateComponentReferenceArgs[*FailureAction]{
			componentType:      componentType,
			componentName:      componentName,
			typ:                objType,
			components:         a.Components.FailureActions,
			reference:          r.Reference,
			referenceValueNode: core.Reference.GetValueNodeOrRoot(core.RootNode),
		}, opts...)
	default:
		return []error{
			validation.NewValueError(fmt.Sprintf("reference to %s is not valid, valid components are [parameters, successActions, failureActions]", componentType), core, core.Reference),
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

func validateComponentReference[T any, V interfaces.Validator[T]](ctx context.Context, args validateComponentReferenceArgs[V], opts ...validation.Option) []error {
	typ := reflect.TypeOf((*T)(nil)).Elem()

	if args.typ != typ {
		return []error{
			validation.NewNodeError(fmt.Sprintf("expected a %s reference got %s", typeToComponentType(args.typ), args.componentType), args.referenceValueNode),
		}
	}

	if args.components == nil {
		return []error{
			validation.NewNodeError(fmt.Sprintf("components.%s not present, reference to missing component %s", args.componentType, *args.reference), args.referenceValueNode),
		}
	}

	component, ok := args.components.Get(args.componentName)
	if !ok {
		return []error{
			validation.NewNodeError(fmt.Sprintf("components.%s.%s not present, reference to missing component %s", args.componentType, args.componentName, *args.reference), args.referenceValueNode),
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
