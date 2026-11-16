package criterion

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/arazzo/internal/specversion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
	"go.yaml.in/yaml/v3"
)

// CriterionType represents the type of criterion.
type CriterionType string

const (
	// CriterionTypeSimple indicates that the criterion represents a simple condition to be evaluated.
	CriterionTypeSimple CriterionType = "simple"
	// CriterionTypeRegex indicates that the criterion represents a regular expression to be evaluated.
	CriterionTypeRegex CriterionType = "regex"
	// CriterionTypeJsonPath indicates that the criterion represents a JSONPath expression to be evaluated.
	CriterionTypeJsonPath CriterionType = "jsonpath"
	// CriterionTypeXPath indicates that the criterion represents an XPath expression to be evaluated.
	CriterionTypeXPath CriterionType = "xpath"
	// CriterionTypeJsonPointer indicates a JSON Pointer (RFC 6901) selector. Only valid for Selector Objects
	// and Payload Replacement target selector types, not for criteria. Arazzo 1.1+ only.
	CriterionTypeJsonPointer CriterionType = "jsonpointer"
)

// CriterionTypeVersion represents the version of the criterion type.
type CriterionTypeVersion string

const (
	CriterionTypeVersionNone                            CriterionTypeVersion = ""
	CriterionTypeVersionDraftGoessnerDispatchJsonPath00 CriterionTypeVersion = "draft-goessner-dispatch-jsonpath-00"
	// CriterionTypeVersionRFC9535 is the RFC 9535 JSONPath version (the default JSONPath version in Arazzo 1.1+).
	CriterionTypeVersionRFC9535 CriterionTypeVersion = "rfc9535"
	// CriterionTypeVersionXPath31 is the XPath 3.1 version (the default XPath version in Arazzo 1.1+).
	CriterionTypeVersionXPath31 CriterionTypeVersion = "xpath-31"
	CriterionTypeVersionXPath30 CriterionTypeVersion = "xpath-30"
	CriterionTypeVersionXPath20 CriterionTypeVersion = "xpath-20"
	CriterionTypeVersionXPath10 CriterionTypeVersion = "xpath-10"
	// CriterionTypeVersionRFC6901 is the RFC 6901 JSON Pointer version. Arazzo 1.1+ only.
	CriterionTypeVersionRFC6901 CriterionTypeVersion = "rfc6901"
)

// allowedVersions returns the allowed versions for the expression type given the document version.
func allowedVersions(typ CriterionType, arazzo11 bool) []CriterionTypeVersion {
	switch typ {
	case CriterionTypeJsonPath:
		if arazzo11 {
			return []CriterionTypeVersion{CriterionTypeVersionRFC9535, CriterionTypeVersionDraftGoessnerDispatchJsonPath00}
		}
		return []CriterionTypeVersion{CriterionTypeVersionDraftGoessnerDispatchJsonPath00}
	case CriterionTypeXPath:
		if arazzo11 {
			return []CriterionTypeVersion{CriterionTypeVersionXPath31, CriterionTypeVersionXPath30, CriterionTypeVersionXPath20, CriterionTypeVersionXPath10}
		}
		return []CriterionTypeVersion{CriterionTypeVersionXPath30, CriterionTypeVersionXPath20, CriterionTypeVersionXPath10}
	case CriterionTypeJsonPointer:
		return []CriterionTypeVersion{CriterionTypeVersionRFC6901}
	default:
		return nil
	}
}

// CriterionExpressionType represents the type of expression used to evaluate the criterion.
// In Arazzo 1.1 this is the Expression Type Object, which is also used by Selector Objects and
// Payload Replacement Objects to describe the selector type and version.
type CriterionExpressionType struct {
	marshaller.Model[core.CriterionExpressionType]

	// Type is the type of criterion.
	Type CriterionType
	// Version is the version of the criterion type.
	Version CriterionTypeVersion
}

// Validate will validate the criterion expression type object against the Arazzo specification
// in the context of a Criterion Object (types `jsonpath` and `xpath`).
func (c *CriterionExpressionType) Validate(opts ...validation.Option) []error {
	return c.validate([]CriterionType{CriterionTypeJsonPath, CriterionTypeXPath}, opts...)
}

// ValidateAsSelectorType will validate the expression type object in the context of a Selector Object or
// Payload Replacement target selector type (types `jsonpath`, `xpath` and `jsonpointer`). Arazzo 1.1+ only.
func (c *CriterionExpressionType) ValidateAsSelectorType(opts ...validation.Option) []error {
	return c.validate([]CriterionType{CriterionTypeJsonPath, CriterionTypeXPath, CriterionTypeJsonPointer}, opts...)
}

func (c *CriterionExpressionType) validate(allowedTypes []CriterionType, opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	arazzo11 := specversion.IsAtLeast11(validation.NewOptions(opts...))

	if !slices.Contains(allowedTypes, c.Type) {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("type must be one of [`%s`]", joinTypes(allowedTypes)), core, core.Type))
	} else {
		versions := allowedVersions(c.Type, arazzo11)
		if !slices.Contains(versions, c.Version) {
			strVersions := make([]string, 0, len(versions))
			for _, v := range versions {
				strVersions = append(strVersions, string(v))
			}
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("version must be one of [`%s`]", strings.Join(strVersions, ", ")), core, core.Version))
		}
	}

	if len(errs) == 0 {
		c.Valid = true
	}

	return errs
}

func joinTypes(types []CriterionType) string {
	strTypes := make([]string, 0, len(types))
	for _, t := range types {
		strTypes = append(strTypes, string(t))
	}
	return strings.Join(strTypes, ", ")
}

// IsTypeProvided will return true if the criterion expression type has a type set.
func (c *CriterionExpressionType) IsTypeProvided() bool {
	if c == nil {
		return false
	}

	return string(c.Type) != ""
}

// CriterionTypeUnion represents the union of a criterion type and criterion expression type.
type CriterionTypeUnion struct {
	// Type is the type of criterion.
	Type *CriterionType
	// ExpressionType is the type of the criterion and any version.
	ExpressionType *CriterionExpressionType

	core core.CriterionTypeUnion
}

// GetCore will return the low level representation of the criterion type union object.
// Useful for accessing line and column numbers for various nodes in the backing yaml/json document.
func (c *CriterionTypeUnion) GetCore() *core.CriterionTypeUnion {
	return &c.core
}

// IsTypeProvided will return true if the criterion type union has a type set.
func (c *CriterionTypeUnion) IsTypeProvided() bool {
	if c == nil {
		return false
	}

	return c.ExpressionType.IsTypeProvided() || (c.Type != nil && *c.Type != "")
}

// GetType will return the type of the criterion.
func (c CriterionTypeUnion) GetType() CriterionType {
	if c.Type == nil && c.ExpressionType == nil {
		return CriterionTypeSimple
	}

	if c.Type != nil {
		return *c.Type
	} else {
		return c.ExpressionType.Type
	}
}

// GetVersion will return the version of the criterion type.
func (c CriterionTypeUnion) GetVersion() CriterionTypeVersion {
	if c.ExpressionType == nil {
		return CriterionTypeVersionNone
	}

	return c.ExpressionType.Version
}

// ValidateAsSelectorType will validate the type union as a selector type (`jsonpath`, `xpath` or `jsonpointer`,
// or an Expression Type Object with one of those types), as used by Selector Objects and Payload Replacement
// Objects in Arazzo 1.1+. The provided node is used for error locations when the type is a plain string.
func (c *CriterionTypeUnion) ValidateAsSelectorType(node *yaml.Node, opts ...validation.Option) []error {
	selectorTypes := []CriterionType{CriterionTypeJsonPath, CriterionTypeXPath, CriterionTypeJsonPointer}

	switch {
	case c.Type != nil:
		if !slices.Contains(selectorTypes, *c.Type) {
			return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("type must be one of [`%s`]", joinTypes(selectorTypes)), node)}
		}
	case c.ExpressionType != nil:
		return c.ExpressionType.ValidateAsSelectorType(opts...)
	}

	return nil
}

func (c *CriterionTypeUnion) Populate(source any) error {
	var coreCriterionTypeUnion core.CriterionTypeUnion
	switch s := source.(type) {
	case core.CriterionTypeUnion:
		coreCriterionTypeUnion = s
	case *core.CriterionTypeUnion:
		if s == nil {
			return nil
		}
		coreCriterionTypeUnion = *s
	default:
		return fmt.Errorf("expected core.CriterionTypeUnion, got %T", source)
	}

	if coreCriterionTypeUnion.Type != nil {
		typ := (CriterionType)(*coreCriterionTypeUnion.Type)
		c.Type = &typ
	} else if coreCriterionTypeUnion.ExpressionType != nil {
		c.ExpressionType = &CriterionExpressionType{}
		if err := marshaller.PopulateWithContext(*coreCriterionTypeUnion.ExpressionType, c.ExpressionType, nil); err != nil {
			return err
		}
	}

	c.core = coreCriterionTypeUnion

	return nil
}

// Criterion represents a criterion that will be evaluated for a given step.
type Criterion struct {
	marshaller.Model[core.Criterion]

	// Context is the expression to the value to be evaluated.
	Context *expression.Expression
	// Condition is the condition to be evaluated.
	Condition string
	// Type is the type of criterion. Defaults to CriterionTypeSimple.
	Type CriterionTypeUnion
	// Extensions provides a list of extensions to the Criterion object.
	Extensions *extensions.Extensions
}

// Sync will sync any changes made to the Arazzo document models back to the core models.
func (c *Criterion) Sync(ctx context.Context) error {
	if _, err := marshaller.SyncValue(ctx, c, c.GetCore(), nil, false); err != nil {
		return err
	}
	return nil
}

// GetCondition will return the condition as a parsed condition object
func (c *Criterion) GetCondition() (*Condition, error) {
	return newCondition(c.Condition)
}

// Validate will validate the criterion object against the Arazzo specification.
func (c *Criterion) Validate(opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	if c.Condition == "" {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("condition is required"), core, core.Condition))
	}

	if c.Type.Type != nil {
		switch *c.Type.Type {
		case CriterionTypeSimple:
		case CriterionTypeRegex:
		case CriterionTypeJsonPath:
		case CriterionTypeXPath:
		default:
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationAllowedValues, fmt.Errorf("type must be one of [`%s`]", strings.Join([]string{string(CriterionTypeSimple), string(CriterionTypeRegex), string(CriterionTypeJsonPath), string(CriterionTypeXPath)}, ", ")), core, core.Type))
		}
	} else if c.Type.ExpressionType != nil {
		errs = append(errs, c.Type.ExpressionType.Validate(opts...)...)
	}

	if c.Type.IsTypeProvided() && c.Context == nil {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("context is required, if type is set"), core, core.Context))
	}

	if c.Context != nil {
		if err := c.Context.Validate(specversion.ExpressionOptions(validation.NewOptions(opts...))...); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("%s", err.Error()), core, core.Context))
		}
	}

	errs = append(errs, c.validateCondition(opts...)...)

	if len(errs) == 0 {
		c.Valid = true
	}

	return errs
}

func (c *Criterion) validateCondition(opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	valueNode := core.Condition.GetValueNodeOrRoot(core.RootNode)

	switch c.Type.GetType() {
	case CriterionTypeSimple:
		cond, err := newCondition(c.Condition)
		if err != nil && c.Context == nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("%s", err.Error()), core, core.Condition))
		} else if cond != nil {
			errs = append(errs, cond.Validate(valueNode, opts...)...)
		}
	case CriterionTypeRegex:
		_, err := regexp.Compile(c.Condition)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("invalid regex expression: %w", err), core, core.Condition))
		}
	case CriterionTypeJsonPath:
		if _, err := jsonpath.NewPath(c.Condition); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("invalid jsonpath expression: %w", err), core, core.Condition))
		}
	case CriterionTypeXPath:
		// TODO validate xpath
	case CriterionTypeJsonPointer:
		// Not a valid criterion type, reported when validating the type
	}

	return errs
}
