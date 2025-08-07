package references

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/speakeasy-api/openapi/jsonpointer"
)

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
	return jsonpointer.JSONPointer(strings.TrimSpace(parts[1]))
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
			return fmt.Errorf("invalid reference JSON pointer: empty")
		}

		if err := jp.Validate(); err != nil {
			return fmt.Errorf("invalid reference JSON pointer: %w", err)
		}
	}

	return nil
}

func (r Reference) String() string {
	return string(r)
}
