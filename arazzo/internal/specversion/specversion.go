// Package specversion provides helpers for version-aware validation of Arazzo documents.
//
// The Arazzo document version is threaded through validation via an oas3.ParentDocumentVersion
// context object, which allows nested objects (including those in the criterion package) to
// apply version specific rules without importing the root arazzo package.
package specversion

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/internal/version"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/validation"
)

// Parse parses an Arazzo version string, ignoring any prerelease suffix (e.g. `1.1.0-rc1`).
func Parse(v string) (*version.Version, error) {
	base, _, _ := strings.Cut(v, "-")
	return version.Parse(base)
}

// FromOptions returns the Arazzo document version from the validation options, or nil if it is
// not available or cannot be parsed.
func FromOptions(o *validation.Options) *version.Version {
	if o == nil {
		return nil
	}

	pdv := validation.GetContextObject[oas3.ParentDocumentVersion](o)
	if pdv == nil || pdv.Arazzo == nil {
		return nil
	}

	v, err := Parse(*pdv.Arazzo)
	if err != nil {
		return nil
	}

	return v
}

// IsAtLeast11 returns true if the Arazzo document being validated declares version 1.1 or later.
func IsAtLeast11(o *validation.Options) bool {
	v := FromOptions(o)
	if v == nil {
		return false
	}

	return v.Major > 1 || (v.Major == 1 && v.Minor >= 1)
}

// ExpressionOptions returns the runtime expression validation options appropriate for the Arazzo
// document version being validated.
func ExpressionOptions(o *validation.Options) []expression.ValidateOption {
	if IsAtLeast11(o) {
		return []expression.ValidateOption{expression.WithArazzo11Grammar()}
	}
	return nil
}

// Requires11Error returns the error used when an Arazzo 1.1 feature is used in a document that
// declares an earlier version.
func Requires11Error(feature string) error {
	return fmt.Errorf("%s requires Arazzo 1.1.0 or later, update the `arazzo` version (e.g. with `openapi arazzo upgrade`)", feature)
}
