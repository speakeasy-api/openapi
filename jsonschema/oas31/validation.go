package oas31

import (
	"bytes"
	"context"
	"errors"

	_ "embed"

	jsValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON string

//go:embed schema.base.json
var schemaBaseJSON string

var oasSchemaValidator *jsValidator.Schema

func (js *Schema) Validate(ctx context.Context, opts ...validation.Option) []error {
	// TODO we maybe need to unset any $schema node as it will potentially change how the schema is validated

	buf := bytes.NewBuffer([]byte{})
	core := js.GetCore()

	if err := json.YAMLToJSON(core.RootNode, 0, buf); err != nil {
		return []error{
			validation.NewNodeError(err.Error(), core.RootNode),
		}
	}

	jsAny, err := jsValidator.UnmarshalJSON(buf)
	if err != nil {
		return []error{
			validation.NewNodeError(err.Error(), core.RootNode),
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
				validation.NewNodeError(err.Error(), core.RootNode),
			}
		}
	}

	js.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

type marshallerNode interface {
	GetKeyNodeOrRoot(rootNode *yaml.Node) *yaml.Node
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

			mn, ok := t.(marshallerNode)
			if !ok {
				// TODO will this be possible? Maybe if the issue is in an extension?
				panic(errors.New("expected marshallerNode"))
			}

			errs = append(errs, validation.NewNodeError("jsonschema validation error: "+cause.Error(), mn.GetKeyNodeOrRoot(js.RootNode)))
		} else {
			errs = append(errs, getRootCauses(cause, js)...)
		}
	}

	return errs
}

func init() {
	oasSchema, err := jsValidator.UnmarshalJSON(bytes.NewReader([]byte(schemaJSON)))
	if err != nil {
		panic(err)
	}

	oasSchemaBase, err := jsValidator.UnmarshalJSON(bytes.NewReader([]byte(schemaBaseJSON)))
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
}
