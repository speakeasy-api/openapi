package oas3

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"

	_ "embed"

	jsValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

//go:embed schema31.json
var schema31JSON string

//go:embed schema31.base.json
var schema31BaseJSON string

var oasSchemaValidator *jsValidator.Schema
var defaultPrinter = message.NewPrinter(language.English)

func Validate[T Referenceable | Concrete](ctx context.Context, schema *JSONSchema[T], opts ...validation.Option) []error {
	if schema == nil {
		return nil
	}

	if schema.IsLeft() {
		return schema.GetLeft().Validate(ctx, opts...)
	}

	return nil
}

func (js *Schema) Validate(ctx context.Context, opts ...validation.Option) []error {
	initValidation()

	buf := bytes.NewBuffer([]byte{})
	core := js.GetCore()

	if err := json.YAMLToJSON(core.RootNode, 0, buf); err != nil {
		return []error{
			validation.NewValidationError(validation.NewTypeMismatchError("schema is not valid json: %s", err.Error()), core.RootNode),
		}
	}

	jsAny, err := jsValidator.UnmarshalJSON(buf)
	if err != nil {
		return []error{
			validation.NewValidationError(validation.NewTypeMismatchError("schema is not valid json: %s", err.Error()), core.RootNode),
		}
	}

	var errs []error
	err = oasSchemaValidator.Validate(jsAny)
	if err != nil {
		var validationErr *jsValidator.ValidationError
		if errors.As(err, &validationErr) {
			errs = getRootCauses(validationErr, *core)
		} else {
			errs = []error{
				validation.NewValidationError(validation.NewValueValidationError("schema invalid: %s", err.Error()), core.RootNode),
			}
		}
	}

	js.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func getRootCauses(err *jsValidator.ValidationError, js core.Schema) []error {
	errs := []error{}

	for _, cause := range err.Causes {
		if len(cause.Causes) == 0 {
			errJP := jsonpointer.PartsToJSONPointer(cause.InstanceLocation)
			t, err := jsonpointer.GetTarget(js, errJP, jsonpointer.WithStructTags("key"))
			if err != nil {
				// TODO need to potentially handle this in another way
				errs = append(errs, err)
				continue
			}

			valueNode := js.RootNode

			type marshallerNode interface {
				GetValueNodeOrRoot(rootNode *yaml.Node) *yaml.Node
			}

			if mn, ok := t.(marshallerNode); ok {
				valueNode = mn.GetValueNodeOrRoot(js.RootNode)
			} else if ra, ok := t.(marshaller.RootNodeAccessor); ok {
				modelRootNode := ra.GetRootNode()
				if modelRootNode != nil {
					valueNode = modelRootNode
				}
			}

			switch cause.ErrorKind.(type) {
			case *kind.Type:
				errs = append(errs, validation.NewValidationError(validation.NewTypeMismatchError("schema field %s %s", strings.Join(cause.InstanceLocation, "."), cause.ErrorKind.LocalizedString(defaultPrinter)), valueNode))
			case *kind.Required:
				errs = append(errs, validation.NewValidationError(validation.NewMissingFieldError("schema field %s %s", strings.Join(cause.InstanceLocation, "."), cause.ErrorKind.LocalizedString(defaultPrinter)), valueNode))
			default:
				errs = append(errs, validation.NewValidationError(validation.NewValueValidationError("schema field %s %s", strings.Join(cause.InstanceLocation, "."), cause.ErrorKind.LocalizedString(defaultPrinter)), valueNode))
			}
		} else {
			errs = append(errs, getRootCauses(cause, js)...)
		}
	}

	return errs
}

var validationInitialized bool
var initMutex sync.Mutex

func initValidation() {
	initMutex.Lock()
	defer initMutex.Unlock()
	if validationInitialized {
		return
	}

	oasSchema, err := jsValidator.UnmarshalJSON(bytes.NewReader([]byte(schema31JSON)))
	if err != nil {
		panic(err)
	}

	oasSchemaBase, err := jsValidator.UnmarshalJSON(bytes.NewReader([]byte(schema31BaseJSON)))
	if err != nil {
		panic(err)
	}

	c := jsValidator.NewCompiler()
	if err := c.AddResource("https://spec.openapis.org/oas/3.1/meta/base", oasSchemaBase); err != nil {
		panic(err)
	}
	if err := c.AddResource("schema.json", oasSchema); err != nil {
		panic(err)
	}
	oasSchemaValidator = c.MustCompile("schema.json")
	validationInitialized = true
}
