package oas31

import (
	"github.com/speakeasy-api/openapi/jsonschema/oas31/core"
	"github.com/speakeasy-api/openapi/pointer"
	"gopkg.in/yaml.v3"
)

type (
	ExclusiveMaximum = *EitherValue[bool, bool, float64, float64]
	ExclusiveMinimum = *EitherValue[bool, bool, float64, float64]
	Type             = *EitherValue[[]string, []string, string, string]
)

func NewExclusiveMaximumFromBool(value bool) ExclusiveMaximum {
	return &EitherValue[bool, bool, float64, float64]{
		Left: pointer.From(value),
	}
}

func NewExclusiveMaximumFromFloat64(value float64) ExclusiveMaximum {
	return &EitherValue[bool, bool, float64, float64]{
		Right: pointer.From(value),
	}
}

func NewExclusiveMinimumFromBool(value bool) ExclusiveMinimum {
	return &EitherValue[bool, bool, float64, float64]{
		Left: pointer.From(value),
	}
}

func NewExclusiveMinimumFromFloat64(value float64) ExclusiveMinimum {
	return &EitherValue[bool, bool, float64, float64]{
		Right: pointer.From(value),
	}
}

func NewTypeFromArray(value []string) Type {
	return &EitherValue[[]string, []string, string, string]{
		Left:  pointer.From(value),
		Right: nil,
	}
}

func NewTypeFromString(value string) Type {
	return &EitherValue[[]string, []string, string, string]{
		Right: pointer.From(value),
	}
}

func NewJSONSchemaOrBoolFromJSONSchema(value Schema) JSONSchema {
	return &EitherValue[Schema, core.Schema, bool, bool]{
		Left:  pointer.From(value),
		Right: nil,
	}
}

func NewJSONSchemaOrBoolFromBool(value bool) JSONSchema {
	return &EitherValue[Schema, core.Schema, bool, bool]{
		Left:  nil,
		Right: pointer.From(value),
	}
}

type EitherValue[L any, LCore any, R any, RCore any] struct {
	Left  *L
	Right *R

	core core.EitherValue[LCore, RCore]
}

func (e *EitherValue[L, LCore, R, RCore]) GetCore() *core.EitherValue[LCore, RCore] {
	return &e.core
}

func (e *EitherValue[L, LCore, R, RCore]) IsLeft() bool {
	return e.Left != nil
}

func (e *EitherValue[L, LCore, R, RCore]) GetLeft() L {
	return *e.Left
}

func (e *EitherValue[L, LCore, R, RCore]) IsRight() bool {
	return e.Right != nil
}

func (e *EitherValue[L, LCore, R, RCore]) GetRight() R {
	return *e.Right
}

type Value = *yaml.Node
