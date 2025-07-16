package criterion

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/speakeasy-api/openapi/arazzo/core"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/validation"
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
)

// CriterionTypeVersion represents the version of the criterion type.
type CriterionTypeVersion string

const (
	CriterionTypeVersionNone                            CriterionTypeVersion = ""
	CriterionTypeVersionDraftGoessnerDispatchJsonPath00 CriterionTypeVersion = "draft-goessner-dispatch-jsonpath-00"
	CriterionTypeVersionXPath30                         CriterionTypeVersion = "xpath-30"
	CriterionTypeVersionXPath20                         CriterionTypeVersion = "xpath-20"
	CriterionTypeVersionXPath10                         CriterionTypeVersion = "xpath-10"
)

// CriterionExpressionType represents the type of expression used to evaluate the criterion.
type CriterionExpressionType struct {
	marshaller.Model[core.CriterionExpressionType]

	// Type is the type of criterion.
	Type CriterionType
	// Version is the version of the criterion type.
	Version CriterionTypeVersion
}

// Validate will validate the criterion expression type object against the Arazzo specification.
func (c *CriterionExpressionType) Validate(opts ...validation.Option) []error {
	core := c.GetCore()
	errs := []error{}

	switch c.Type {
	case CriterionTypeJsonPath:
		switch c.Version {
		case CriterionTypeVersionDraftGoessnerDispatchJsonPath00:
		default:
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("version must be one of [%s]", strings.Join([]string{string(CriterionTypeVersionDraftGoessnerDispatchJsonPath00)}, ", ")), core, core.Version))
		}
	case CriterionTypeXPath:
		switch c.Version {
		case CriterionTypeVersionXPath30:
		case CriterionTypeVersionXPath20:
		case CriterionTypeVersionXPath10:
		default:
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("version must be one of [%s]", strings.Join([]string{string(CriterionTypeVersionXPath30), string(CriterionTypeVersionXPath20), string(CriterionTypeVersionXPath10)}, ", ")), core, core.Version))
		}
	default:
		errs = append(errs, validation.NewValueError(validation.NewValueValidationError("type must be one of [%s]", strings.Join([]string{string(CriterionTypeJsonPath), string(CriterionTypeXPath)}, ", ")), core, core.Type))
	}

	if len(errs) == 0 {
		c.Valid = true
	}

	return errs
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

func (c *CriterionTypeUnion) Populate(source any) error {
	coreCriterionTypeUnion, ok := source.(core.CriterionTypeUnion)
	if !ok {
		return fmt.Errorf("expected core.CriterionTypeUnion, got %T", c)
	}

	if coreCriterionTypeUnion.Type != nil {
		typ := (CriterionType)(*coreCriterionTypeUnion.Type)
		c.Type = &typ
	} else if coreCriterionTypeUnion.ExpressionType != nil {
		c.ExpressionType = &CriterionExpressionType{}
		if err := marshaller.Populate(*coreCriterionTypeUnion.ExpressionType, c.ExpressionType); err != nil {
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
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("condition is required"), core, core.Condition))
	}

	if c.Type.Type != nil {
		switch *c.Type.Type {
		case CriterionTypeSimple:
		case CriterionTypeRegex:
		case CriterionTypeJsonPath:
		case CriterionTypeXPath:
		default:
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("type must be one of [%s]", strings.Join([]string{string(CriterionTypeSimple), string(CriterionTypeRegex), string(CriterionTypeJsonPath), string(CriterionTypeXPath)}, ", ")), core, core.Type))
		}
	} else if c.Type.ExpressionType != nil {
		errs = append(errs, c.Type.ExpressionType.Validate(opts...)...)
	}

	if c.Type.IsTypeProvided() && c.Context == nil {
		errs = append(errs, validation.NewValueError(validation.NewMissingValueError("context is required, if type is set"), core, core.Context))
	}

	if c.Context != nil {
		if err := c.Context.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError(err.Error()), core, core.Context))
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

	conditionLine := core.Condition.GetValueNodeOrRoot(core.RootNode).Line
	conditionColumn := core.Condition.GetValueNodeOrRoot(core.RootNode).Column

	switch c.Type.GetType() {
	case CriterionTypeSimple:
		cond, err := newCondition(c.Condition)
		if err != nil && c.Context == nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError(err.Error()), core, core.Condition))
		} else if cond != nil {
			errs = append(errs, cond.Validate(conditionLine, conditionColumn, opts...)...)
		}
	case CriterionTypeRegex:
		_, err := regexp.Compile(c.Condition)
		if err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("invalid regex expression: %s", err.Error()), core, core.Condition))
		}
	case CriterionTypeJsonPath:
		if _, err := jsonpath.NewPath(c.Condition); err != nil {
			errs = append(errs, validation.NewValueError(validation.NewValueValidationError("invalid jsonpath expression: %s", err), core, core.Condition))
		}
	case CriterionTypeXPath:
		// TODO validate xpath
	}

	return errs
}
