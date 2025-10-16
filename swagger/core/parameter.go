package core

import (
	"github.com/speakeasy-api/openapi/extensions/core"
	oascore "github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
)

// Parameter describes a single operation parameter.
type Parameter struct {
	marshaller.CoreModel `model:"parameter"`

	// Common fields for all parameter types
	Name        marshaller.Node[string]  `key:"name"`
	In          marshaller.Node[string]  `key:"in"`
	Description marshaller.Node[*string] `key:"description"`
	Required    marshaller.Node[*bool]   `key:"required"`

	// For body parameters
	Schema marshaller.Node[oascore.JSONSchema] `key:"schema"`

	// For non-body parameters
	Type             marshaller.Node[*string]                         `key:"type"`
	Format           marshaller.Node[*string]                         `key:"format"`
	AllowEmptyValue  marshaller.Node[*bool]                           `key:"allowEmptyValue"`
	Items            marshaller.Node[*Items]                          `key:"items"`
	CollectionFormat marshaller.Node[*string]                         `key:"collectionFormat"`
	Default          marshaller.Node[values.Value]                    `key:"default"`
	Maximum          marshaller.Node[*float64]                        `key:"maximum"`
	ExclusiveMaximum marshaller.Node[*bool]                           `key:"exclusiveMaximum"`
	Minimum          marshaller.Node[*float64]                        `key:"minimum"`
	ExclusiveMinimum marshaller.Node[*bool]                           `key:"exclusiveMinimum"`
	MaxLength        marshaller.Node[*int64]                          `key:"maxLength"`
	MinLength        marshaller.Node[*int64]                          `key:"minLength"`
	Pattern          marshaller.Node[*string]                         `key:"pattern"`
	MaxItems         marshaller.Node[*int64]                          `key:"maxItems"`
	MinItems         marshaller.Node[*int64]                          `key:"minItems"`
	UniqueItems      marshaller.Node[*bool]                           `key:"uniqueItems"`
	Enum             marshaller.Node[[]marshaller.Node[values.Value]] `key:"enum"`
	MultipleOf       marshaller.Node[*float64]                        `key:"multipleOf"`

	Extensions core.Extensions `key:"extensions"`
}

// Items is a limited subset of JSON-Schema's items object for array parameters.
type Items struct {
	marshaller.CoreModel `model:"items"`

	Type             marshaller.Node[string]                          `key:"type"`
	Format           marshaller.Node[*string]                         `key:"format"`
	Items            marshaller.Node[*Items]                          `key:"items"`
	CollectionFormat marshaller.Node[*string]                         `key:"collectionFormat"`
	Default          marshaller.Node[values.Value]                    `key:"default"`
	Maximum          marshaller.Node[*float64]                        `key:"maximum"`
	ExclusiveMaximum marshaller.Node[*bool]                           `key:"exclusiveMaximum"`
	Minimum          marshaller.Node[*float64]                        `key:"minimum"`
	ExclusiveMinimum marshaller.Node[*bool]                           `key:"exclusiveMinimum"`
	MaxLength        marshaller.Node[*int64]                          `key:"maxLength"`
	MinLength        marshaller.Node[*int64]                          `key:"minLength"`
	Pattern          marshaller.Node[*string]                         `key:"pattern"`
	MaxItems         marshaller.Node[*int64]                          `key:"maxItems"`
	MinItems         marshaller.Node[*int64]                          `key:"minItems"`
	UniqueItems      marshaller.Node[*bool]                           `key:"uniqueItems"`
	Enum             marshaller.Node[[]marshaller.Node[values.Value]] `key:"enum"`
	MultipleOf       marshaller.Node[*float64]                        `key:"multipleOf"`

	Extensions core.Extensions `key:"extensions"`
}
