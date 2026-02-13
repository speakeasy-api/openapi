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
	sb.Grow(len(l) * 20) // pre-allocate for typical pointer segments
	sb.WriteString("/")

	needsSep := false
	for _, location := range l {
		if location.ParentField != "" {
			if needsSep {
				sb.WriteString("/")
			}
			sb.WriteString(jsonpointer.EscapeString(location.ParentField))
			needsSep = true
		}

		if location.ParentKey != nil {
			sb.WriteString("/")
			sb.WriteString(jsonpointer.EscapeString(*location.ParentKey))
			needsSep = true
		} else if location.ParentIndex != nil {
			sb.WriteString("/")
			sb.WriteString(strconv.Itoa(*location.ParentIndex))
			needsSep = true
		}
	}

	return jsonpointer.JSONPointer(sb.String())
}

// IsParent checks if the immediate parent field matches the given field name.
// It handles both direct struct fields and map/slice items.
func (l Locations[T]) IsParent(field string) bool {
	if len(l) == 0 {
		return false
	}

	last := l[len(l)-1]
	if last.ParentKey != nil || last.ParentIndex != nil {
		if len(l) < 2 {
			return false
		}
		return l[len(l)-2].ParentField == field
	}

	return last.ParentField == field
}

// ParentKey returns the key of the current item if it is in a map.
// Returns empty string if not in a map or key is nil.
func (l Locations[T]) ParentKey() string {
	if len(l) == 0 {
		return ""
	}
	last := l[len(l)-1]
	if last.ParentKey != nil {
		return *last.ParentKey
	}
	return ""
}
