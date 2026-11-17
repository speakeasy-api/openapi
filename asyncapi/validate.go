package asyncapi

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"strings"

	"github.com/speakeasy-api/openapi/internal/asyncapiversion"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

var (
	// componentKeyRegex is the pattern all Components Object map keys must match.
	componentKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)
	// nameKeyRegex is the pattern Servers Object and Channel parameter keys must match.
	nameKeyRegex = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)
	// expressionVariableRegex matches `{name}` variables in server hosts, pathnames and channel addresses.
	expressionVariableRegex = regexp.MustCompile(`\{([^{}]+)\}`)
	// runtimeExpressionRegex matches the runtime expressions used by correlation IDs, parameters and reply addresses.
	runtimeExpressionRegex = regexp.MustCompile(`^\$message\.(header|payload)(#.*)?$`)
)

// documentVersionOption carries the parsed version of the document being validated through validation options.
type documentVersionOption struct {
	version *asyncapiversion.Version
}

func withDocumentVersion(v *asyncapiversion.Version) validation.Option {
	return validation.WithContextObject(&documentVersionOption{version: v})
}

// documentVersion returns the version of the document being validated, or nil if it is unknown.
func documentVersion(opts []validation.Option) *asyncapiversion.Version {
	if o := validation.GetContextObject[documentVersionOption](validation.NewOptions(opts...)); o != nil {
		return o.version
	}
	return nil
}

// coreValid reports whether a core model is free of unmarshalling errors. Models built in code rather than
// parsed have no root node and no unmarshalling errors, so they are treated as valid.
func coreValid(c interface {
	GetRootNode() *yaml.Node
	GetValid() bool
},
) bool {
	return c.GetRootNode() == nil || c.GetValid()
}

// noErrors reports whether none of the provided diagnostics is an error. Warnings and hints, such as unknown
// properties or content that is preserved but not validated, do not make a model invalid.
func noErrors(errs []error) bool {
	for _, err := range errs {
		var vErr *validation.Error
		if !errors.As(err, &vErr) || vErr.Severity == validation.SeverityError {
			return false
		}
	}
	return true
}

// unknownPropertyErrors returns a warning for each property of the object that is neither a defined field nor an extension.
// The properties are preserved, so documents still round-trip unchanged.
func unknownPropertyErrors(c marshaller.CoreModeler, object string) []error {
	props := c.GetUnknownProperties()
	if len(props) == 0 {
		return nil
	}

	errs := make([]error, 0, len(props))
	for _, prop := range props {
		node := c.GetRootNode()
		if keyNode, _, ok := yml.GetMapElementNodes(context.Background(), node, prop); ok {
			node = keyNode
		}
		errs = append(errs, validation.NewValidationError(validation.SeverityWarning, validation.RuleValidationUnknownProperties, fmt.Errorf("%s has unknown property `%s`", object, prop), node))
	}
	return errs
}

// validateRuntimeExpression validates an AsyncAPI runtime expression such as `$message.payload#/user/id`.
func validateRuntimeExpression(expr string) error {
	m := runtimeExpressionRegex.FindStringSubmatch(expr)
	if m == nil {
		return fmt.Errorf("`%s` is not a valid runtime expression, expected `$message.header` or `$message.payload` optionally followed by `#` and a JSON pointer", expr)
	}

	if fragment := strings.TrimPrefix(m[2], "#"); fragment != "" {
		if err := jsonpointer.JSONPointer(fragment).Validate(); err != nil {
			return fmt.Errorf("`%s` has an invalid JSON pointer fragment: %w", expr, err)
		}
	}

	return nil
}

// expressionVariables returns the names of the `{name}` variables used in the provided value, in order of appearance.
func expressionVariables(value string) []string {
	matches := expressionVariableRegex.FindAllStringSubmatch(value, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

func validateAbsoluteURL(value, field string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s is not a valid uri: %w", field, err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("%s `%s` must be an absolute uri", field, value)
	}
	return nil
}

func validateMediaType(value, field string) error {
	if _, _, err := mime.ParseMediaType(value); err != nil {
		return fmt.Errorf("%s `%s` is not a valid media type", field, value)
	}
	return nil
}

// validateTags validates a list of tags and checks their names are unique.
func validateTags(ctx context.Context, tags []*Tag, c validation.CoreModeler, node validation.SliceNodeGetter, field string, opts ...validation.Option) []error {
	errs := []error{}
	names := map[string]bool{}
	for idx, tag := range tags {
		if tag == nil {
			continue
		}
		errs = append(errs, tag.Validate(ctx, opts...)...)
		if tag.IsReference() || tag.Name == nil {
			continue
		}
		if names[*tag.Name] {
			errs = append(errs, validation.NewSliceError(validation.SeverityError, validation.RuleValidationDuplicateKey, fmt.Errorf("%s name `%s` is not unique", field, *tag.Name), c, node, idx))
		}
		names[*tag.Name] = true
	}
	return errs
}

// validateEnumDefault checks that a default value is one of the allowed enum values.
func validateEnumDefault(enum []string, def *string, c validation.CoreModeler, node validation.ValueNodeGetter, field string) []error {
	if def == nil || len(enum) == 0 {
		return nil
	}
	for _, e := range enum {
		if e == *def {
			return nil
		}
	}
	return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("%s.default `%s` must be one of the enum values [`%s`]", field, *def, strings.Join(enum, "`, `")), c, node)}
}

// requiredString reports a required string that is empty, or missing from a model built in code. Required fields
// missing from parsed documents are already reported by the unmarshaller.
func requiredString(value string, present bool, c validation.CoreModeler, node validation.ValueNodeGetter, field string) []error {
	if value == "" && (present || c.GetRootNode() == nil) {
		return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("`%s` is required", field), c, node)}
	}
	return nil
}
