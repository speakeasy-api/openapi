package oas3

import (
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/values"
)

type (
	ExclusiveMaximum = *values.EitherValue[bool, bool, float64, float64]
	ExclusiveMinimum = *values.EitherValue[bool, bool, float64, float64]
	// Type represents the type of a schema either an array of types or a single type.
	Type = *values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string]
)

func NewExclusiveMaximumFromBool(value bool) ExclusiveMaximum {
	return &values.EitherValue[bool, bool, float64, float64]{
		Left: pointer.From(value),
	}
}

func NewExclusiveMaximumFromFloat64(value float64) ExclusiveMaximum {
	return &values.EitherValue[bool, bool, float64, float64]{
		Right: pointer.From(value),
	}
}

func NewExclusiveMinimumFromBool(value bool) ExclusiveMinimum {
	return &values.EitherValue[bool, bool, float64, float64]{
		Left: pointer.From(value),
	}
}

func NewExclusiveMinimumFromFloat64(value float64) ExclusiveMinimum {
	return &values.EitherValue[bool, bool, float64, float64]{
		Right: pointer.From(value),
	}
}

func NewTypeFromArray(value []SchemaType) Type {
	return &values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string]{
		Left:  pointer.From(value),
		Right: nil,
	}
}

func NewTypeFromString(value SchemaType) Type {
	return &values.EitherValue[[]SchemaType, []marshaller.Node[string], SchemaType, string]{
		Right: pointer.From(value),
	}
}
