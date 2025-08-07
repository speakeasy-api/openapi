package openapi

import (
	"context"
	"net/url"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi/core"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/validation"
)

type Link struct {
	marshaller.Model[core.Link]

	// OperationID is a identified to an existing operation in the API. Mutually exclusive with OperationRef.
	OperationID *string
	// OperationRef is a reference to an existing operation in the API. Mutually exclusive with OperationID.
	OperationRef *string
	// Parameters is a map of parameter names to values or runtime expressions to populate the referenced operation.
	Parameters *sequencedmap.Map[string, expression.ValueOrExpression]
	// RequestBody is either a value or a runtime expression to populate the referenced operation.
	RequestBody expression.ValueOrExpression
	// Description is a description of the link. May contain CommonMark syntax.
	Description *string
	// Server is a server object to be used by the target operation.
	Server *Server

	// Extensions provides a list of extensions to the Link object.
	Extensions *extensions.Extensions
}

var _ interfaces.Model[core.Link] = (*Link)(nil)

// GetOperationID returns the value of the OperationID field. Returns empty string if not set.
func (l *Link) GetOperationID() string {
	if l == nil || l.OperationID == nil {
		return ""
	}
	return *l.OperationID
}

// GetOperationRef returns the value of the OperationRef field. Returns empty string if not set.
func (l *Link) GetOperationRef() string {
	if l == nil || l.OperationRef == nil {
		return ""
	}
	return *l.OperationRef
}

// GetDescription returns the value of the Description field. Returns empty string if not set.
func (l *Link) GetDescription() string {
	if l == nil || l.Description == nil {
		return ""
	}
	return *l.Description
}

// GetParameters returns the value of the Parameters field. Returns nil if not set.
func (l *Link) GetParameters() *sequencedmap.Map[string, expression.ValueOrExpression] {
	if l == nil {
		return nil
	}
	return l.Parameters
}

// GetRequestBody returns the value of the RequestBody field. Returns nil if not set.
func (l *Link) GetRequestBody() expression.ValueOrExpression {
	if l == nil {
		return nil
	}
	return l.RequestBody
}

// GetServer returns the value of the Server field. Returns nil if not set.
func (l *Link) GetServer() *Server {
	if l == nil {
		return nil
	}
	return l.Server
}

// GetExtensions returns the value of the Extensions field. Returns an empty extensions map if not set.
func (l *Link) GetExtensions() *extensions.Extensions {
	if l == nil || l.Extensions == nil {
		return extensions.New()
	}
	return l.Extensions
}

func (l *Link) ResolveOperation(ctx context.Context) (*Operation, error) {
	// TODO implement resolving the operation
	return nil, nil
}

func (l *Link) Validate(ctx context.Context, opts ...validation.Option) []error {
	core := l.GetCore()
	errs := []error{}

	op := validation.NewOptions(opts...)
	o := validation.GetContextObject[OpenAPI](op)

	if core.OperationID.Present && core.OperationRef.Present {
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError("operationID and operationRef are mutually exclusive"), core, core.OperationID))
	}

	if l.OperationID != nil {
		if o == nil {
			panic("OpenAPI object is required to validate operationId")
		}

		foundOp := false

		for _, pi := range o.Paths.All() {
			// TODO replace with walk through operations so we don't have to resolve references to check if operationId exists
			if !pi.IsReference() {
				for _, op := range pi.Object.All() {
					if op.OperationID != nil && *op.OperationID == *l.OperationID {
						foundOp = true
						break
					}
				}
			}
		}

		if !foundOp {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("operationId %s does not exist in document", *l.OperationID), core, core.OperationID))
		}
	}

	// TODO should we validate the reference resolves here? Or as part of the resolution operation? Or make it optional?
	if l.OperationRef != nil {
		if _, err := url.Parse(*l.OperationRef); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("operationRef is not a valid uri: %s", err), core, core.OperationRef))
		}
	}

	for key, exp := range l.GetParameters().All() {
		_, expression, err := expression.GetValueOrExpressionValue(exp)
		if err != nil {
			errs = append(errs, validation.NewMapValueError(validation.NewValueValidationError(err.Error()), core, core.Parameters, key))
		}
		if expression != nil {
			if err := expression.Validate(); err != nil {
				errs = append(errs, validation.NewMapValueError(validation.NewValueValidationError(err.Error()), core, core.Parameters, key))
			}
		}
	}

	_, rbe, err := expression.GetValueOrExpressionValue(l.RequestBody)
	if err != nil {
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError(err.Error()), core, core.RequestBody))
	}
	if rbe != nil {
		if err := rbe.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError(err.Error()), core, core.RequestBody))
		}
	}

	if l.Server != nil {
		errs = append(errs, l.Server.Validate(ctx, opts...)...)
	}

	l.Valid = len(errs) == 0 && core.GetValid()

	return errs
}
