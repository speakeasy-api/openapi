package oas3

type SchemaType string

const (
	SchemaTypeArray   SchemaType = "array"
	SchemaTypeBoolean SchemaType = "boolean"
	SchemaTypeInteger SchemaType = "integer"
	SchemaTypeNumber  SchemaType = "number"
	SchemaTypeNull    SchemaType = "null"
	SchemaTypeObject  SchemaType = "object"
	SchemaTypeString  SchemaType = "string"
)
