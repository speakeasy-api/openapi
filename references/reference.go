package references

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/jsonpointer"
)

// componentNameRegex matches valid OpenAPI component names according to the spec.
// Component names must match: ^[a-zA-Z0-9\.\-_]+$
var componentNameRegex = regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)

type Reference string

var _ fmt.Stringer = (*Reference)(nil)

func (r Reference) GetURI() string {
	parts := strings.Split(string(r), "#")
	if len(parts) < 1 {
		return ""
	}

	return strings.TrimSpace(parts[0])
}

func (r Reference) HasJSONPointer() bool {
	return len(strings.Split(string(r), "#")) > 1
}

func (r Reference) GetJSONPointer() jsonpointer.JSONPointer {
	parts := strings.Split(string(r), "#")
	if len(parts) < 2 {
		return ""
	}

	pointer := strings.TrimSpace(parts[1])

	// URL decode the JSON pointer to handle percent-encoded characters
	// like %25 (which represents %)
	if decoded, err := url.QueryUnescape(pointer); err == nil {
		pointer = decoded
	}

	return jsonpointer.JSONPointer(pointer)
}

func (r Reference) Validate() error {
	if r == "" {
		return nil // TODO do we want to treat empty references as valid?
	}

	uri := r.GetURI()

	if uri != "" {
		if _, err := url.Parse(uri); err != nil {
			return fmt.Errorf("invalid reference URI: %w", err)
		}
	}

	if r.HasJSONPointer() {
		jp := r.GetJSONPointer()
		if jp == "" {
			return errors.New("invalid reference JSON pointer: empty")
		}

		if err := jp.Validate(); err != nil {
			return fmt.Errorf("invalid reference JSON pointer: %w", err)
		}

		// Validate OpenAPI component references have valid component names
		if err := r.validateComponentReference(jp); err != nil {
			return err
		}
	}

	return nil
}

// validateComponentReference validates that component references have valid component names.
// According to the OpenAPI spec, component names must match: ^[a-zA-Z0-9\.\-_]+$
func (r Reference) validateComponentReference(jp jsonpointer.JSONPointer) error {
	jpStr := string(jp)

	// Check if this is a component reference
	if !strings.HasPrefix(jpStr, "/components/") {
		return nil
	}

	// Split the pointer into parts
	parts := strings.Split(strings.TrimPrefix(jpStr, "/"), "/")

	// parts[0] is "components", parts[1] is the component type (schemas, parameters, etc.)
	// parts[2] should be the component name
	if len(parts) < 3 {
		// Reference ends at component type (e.g., #/components/schemas/)
		return errors.New("invalid reference: component name cannot be empty")
	}

	componentName := parts[2]
	if componentName == "" {
		return errors.New("invalid reference: component name cannot be empty")
	}

	// Unescape the component name before validating (JSON pointer escaping: ~0 = ~, ~1 = /)
	unescapedName := strings.ReplaceAll(componentName, "~1", "/")
	unescapedName = strings.ReplaceAll(unescapedName, "~0", "~")

	// Validate component name matches the required pattern
	if !componentNameRegex.MatchString(unescapedName) {
		return fmt.Errorf("invalid reference: component name %q must match pattern ^[a-zA-Z0-9.\\-_]+$", unescapedName)
	}

	return nil
}

func (r Reference) String() string {
	return string(r)
}
