package openapi

import "fmt"

// SerializationStyle represents the serialization style of a parameter.
type SerializationStyle string

var _ fmt.Stringer = (*SerializationStyle)(nil)

func (s SerializationStyle) String() string {
	return string(s)
}

const (
	// SerializationStyleSimple represents simple serialization as defined by RFC 6570. Valid for path, header parameters.
	SerializationStyleSimple SerializationStyle = "simple"
	// SerializationStyleForm represents form serialization as defined by RFC 6570. Valid for query, cookie parameters.
	SerializationStyleForm SerializationStyle = "form"
	// SerializationStyleLabel represents label serialization as defined by RFC 6570. Valid for path parameters.
	SerializationStyleLabel SerializationStyle = "label"
	// SerializationStyleMatrix represents matrix serialization as defined by RFC 6570. Valid for path parameters.
	SerializationStyleMatrix SerializationStyle = "matrix"
	// SerializationStyleSpaceDelimited represents space-delimited serialization. Valid for query parameters.
	SerializationStyleSpaceDelimited SerializationStyle = "spaceDelimited"
	// SerializationStylePipeDelimited represents pipe-delimited serialization. Valid for query parameters.
	SerializationStylePipeDelimited SerializationStyle = "pipeDelimited"
	// SerializationStyleDeepObject represents deep object serialization for rendering nested objects using form parameters. Valid for query parameters.
	SerializationStyleDeepObject SerializationStyle = "deepObject"
)
