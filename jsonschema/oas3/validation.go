package oas3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

// custom file to cover for missing openapi 3.0 json schema
//
//go:embed schema30.dialect.json
var schema30DialectJSON string

// sourced from https://spec.openapis.org/oas/3.1/dialect/2024-11-10.html
//
//go:embed schema31.dialect.json
var schema31DialectJSON string

// source from https://spec.openapis.org/oas/3.1/meta/2024-11-10.html
//
//go:embed schema31.meta.json
var schema31MetaJSON string

// sourced from https://spec.openapis.org/oas/3.2/dialect/2025-09-17.html
//
//go:embed schema32.dialect.json
var schema32DialectJSON string

// source from https://spec.openapis.org/oas/3.2/meta/2025-09-17.html
//
//go:embed schema32.meta.json
var schema32MetaJSON string

var (
	oasSchemaValidator = make(map[string]*jsValidator.Schema)
	defaultPrinter     = message.NewPrinter(language.English)
)

const (
	JSONSchema30SchemaID = "https://spec.openapis.org/oas/3.0/dialect/2024-10-18"
	JSONSchema31SchemaID = "https://spec.openapis.org/oas/3.1/meta/2024-11-10"
	JSONSchema32SchemaID = "https://spec.openapis.org/oas/3.2/meta/2025-09-17"
)

type ParentDocumentVersion struct {
	OpenAPI *string
	Arazzo  *string
}

func Validate[T Referenceable | Concrete](ctx context.Context, schema *JSONSchema[T], opts ...validation.Option) []error {
	if schema == nil {
		return nil
	}

	if schema.IsSchema() {
		return schema.GetSchema().Validate(ctx, opts...)
	}

	return nil
}

func (js *Schema) Validate(ctx context.Context, opts ...validation.Option) []error {
	o := validation.NewOptions(opts...)

	dv := validation.GetContextObject[ParentDocumentVersion](o)

	var schema string
	if js.Schema != nil {
		switch *js.Schema {
		case JSONSchema31SchemaID:
			schema = *js.Schema
		case JSONSchema30SchemaID:
			schema = *js.Schema
		default:
			// Currently not supported
		}
	}
	if schema == "" && dv != nil {
		switch {
		case dv.OpenAPI != nil:
			switch {
			case strings.HasPrefix(*dv.OpenAPI, "3.2"):
				schema = JSONSchema32SchemaID
			case strings.HasPrefix(*dv.OpenAPI, "3.1"):
				schema = JSONSchema31SchemaID
			case strings.HasPrefix(*dv.OpenAPI, "3.0"):
				schema = JSONSchema30SchemaID
			default:
				// Currently not supported
			}
		case dv.Arazzo != nil:
			// Currently not supported for Arazzo documents
		}
	}
	if schema == "" {
		// Default to OpenAPI 3.1 schema TODO: consider maybe defaulting to draft-2020-12 instead
		schema = JSONSchema31SchemaID
	}

	oasSchemaValidator := initValidation(schema)

	buf := bytes.NewBuffer([]byte{})
	core := js.GetCore()

	if err := json.YAMLToJSON(core.RootNode, 0, buf); err != nil {
		return []error{
			validation.NewValidationError(fmt.Errorf("schema is not valid json: %w", err), core.RootNode),
		}
	}

	jsAny, err := jsValidator.UnmarshalJSON(buf)
	if err != nil {
		return []error{
			validation.NewValidationError(fmt.Errorf("schema is not valid json: %w", err), core.RootNode),
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
			var errJP jsonpointer.JSONPointer
			switch {
			case len(cause.InstanceLocation) > 0:
				errJP = jsonpointer.PartsToJSONPointer(cause.InstanceLocation)
			case cause.ErrorKind != nil:
				errJP = jsonpointer.PartsToJSONPointer(cause.ErrorKind.KeywordPath())
			default:
				errJP = jsonpointer.JSONPointer("/")
			}

			t, err := jsonpointer.GetTarget(js, errJP, jsonpointer.WithStructTags("key"))
			if err != nil {
				errs = append(errs, validation.NewValidationError(err, js.GetRootNode()))
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

			parentName := "schema." + strings.Join(cause.InstanceLocation, ".")
			msg := cause.ErrorKind.LocalizedString(defaultPrinter)

			var newErr error
			switch t := cause.ErrorKind.(type) {
			case *kind.Type:
				var want string
				if len(t.Want) == 1 {
					want = t.Want[0]
				} else {
					want = fmt.Sprintf("one of [%s]", strings.Join(t.Want, ", "))
				}

				msg = fmt.Sprintf("expected %s, got %s", want, t.Got)

				newErr = validation.NewValidationError(validation.NewTypeMismatchError(parentName, msg), valueNode)
			case *kind.Required:
				newErr = validation.NewValidationError(validation.NewMissingFieldError("%s %s", parentName, msg), valueNode)
			default:
				newErr = validation.NewValidationError(validation.NewValueValidationError("%s %s", parentName, msg), valueNode)
			}
			if newErr != nil {
				errs = append(errs, newErr)
			}
		} else {
			errs = append(errs, getRootCauses(cause, js)...)
		}
	}

	return errs
}

var (
	validationInitialized = make(map[string]bool)
	initMutex             sync.Mutex
)

func initValidation(schema string) *jsValidator.Schema {
	initMutex.Lock()
	defer initMutex.Unlock()
	if validationInitialized[schema] {
		return oasSchemaValidator[schema]
	}

	var schemaResource any

	c := jsValidator.NewCompiler()

	switch schema {
	case JSONSchema32SchemaID:
		oasSchemaMeta, err := jsValidator.UnmarshalJSON(bytes.NewBufferString(schema32MetaJSON))
		if err != nil {
			panic(err)
		}
		if err := c.AddResource(JSONSchema32SchemaID, oasSchemaMeta); err != nil {
			panic(err)
		}

		schemaResource, err = jsValidator.UnmarshalJSON(bytes.NewBufferString(schema32DialectJSON))
		if err != nil {
			panic(err)
		}
	case JSONSchema31SchemaID:
		oasSchemaMeta, err := jsValidator.UnmarshalJSON(bytes.NewBufferString(schema31MetaJSON))
		if err != nil {
			panic(err)
		}
		if err := c.AddResource(JSONSchema31SchemaID, oasSchemaMeta); err != nil {
			panic(err)
		}

		schemaResource, err = jsValidator.UnmarshalJSON(bytes.NewBufferString(schema31DialectJSON))
		if err != nil {
			panic(err)
		}
	case JSONSchema30SchemaID:
		var err error
		schemaResource, err = jsValidator.UnmarshalJSON(bytes.NewBufferString(schema30DialectJSON))
		if err != nil {
			panic(err)
		}
	default:
		panic("unsupported schema")
	}

	if err := c.AddResource("schema.json", schemaResource); err != nil {
		panic(err)
	}
	oasSchemaValidator[schema] = c.MustCompile("schema.json")
	validationInitialized[schema] = true

	return oasSchemaValidator[schema]
}
