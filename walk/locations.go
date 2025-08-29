package walk

import (
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/speakeasy-api/openapi/jsonpointer"
)

const (
	// ErrTerminate is a sentinel error that can be returned from a MatchFunc to detect when to terminate the walk.
	// When used with the iterator API, users can check for this error and break out of the for loop.
	ErrTerminate = errors.Error("terminate")
)

// LocationContext represents the context of where an element is located within its parent.
// It uses generics to work with different MatchFunc types from different packages.
type LocationContext[T any] struct {
	ParentMatchFunc T
	ParentField     string
	ParentKey       *string
	ParentIndex     *int
}

// Locations represents a slice of location contexts that can be converted to a JSON pointer.
type Locations[T any] []LocationContext[T]

// ToJSONPointer converts the locations to a JSON pointer.
func (l Locations[T]) ToJSONPointer() jsonpointer.JSONPointer {
	var sb strings.Builder
	sb.WriteString("/")

	for _, location := range l {
		if location.ParentField != "" {
			if !strings.HasSuffix(sb.String(), "/") {
				sb.WriteString("/")
			}
			sb.WriteString(jsonpointer.EscapeString(location.ParentField))
		}

		if location.ParentKey != nil {
			sb.WriteString("/")
			sb.WriteString(jsonpointer.EscapeString(*location.ParentKey))
		} else if location.ParentIndex != nil {
			sb.WriteString("/")
			sb.WriteString(strconv.Itoa(*location.ParentIndex))
		}
	}

	return jsonpointer.JSONPointer(sb.String())
}
