package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
)

const RuleOwaspNoNumericIDs = "owasp-no-numeric-ids"

type OwaspNoNumericIDsRule struct{}

func (r *OwaspNoNumericIDsRule) ID() string {
	return RuleOwaspNoNumericIDs
}
func (r *OwaspNoNumericIDsRule) Category() string {
	return CategorySecurity
}
func (r *OwaspNoNumericIDsRule) Description() string {
	return "Resource identifiers must use random values like UUIDs instead of sequential numeric IDs. Sequential IDs enable enumeration attacks where attackers can guess valid IDs and access unauthorized resources."
}
func (r *OwaspNoNumericIDsRule) Summary() string {
	return "Resource identifiers should not use sequential numeric IDs."
}
func (r *OwaspNoNumericIDsRule) HowToFix() string {
	return "Use non-sequential identifiers (e.g., UUIDs) for ID parameters."
}
func (r *OwaspNoNumericIDsRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#owasp-no-numeric-ids"
}
func (r *OwaspNoNumericIDsRule) DefaultSeverity() validation.Severity {
	return validation.SeverityError
}
func (r *OwaspNoNumericIDsRule) Versions() []string {
	return []string{"3.0", "3.1"} // OAS3 only
}

// isIDParameter checks if a parameter name is an ID field
func isIDParameter(name string) bool {
	lowerName := strings.ToLower(name)
	return lowerName == "id" ||
		strings.HasSuffix(lowerName, "_id") ||
		strings.HasSuffix(lowerName, "-id") ||
		strings.HasSuffix(lowerName, "id")
}

func (r *OwaspNoNumericIDsRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Check all parameters (inline, component, external, and references)
	for _, paramNode := range docInfo.Index.GetAllParameters() {
		param := paramNode.Node
		if param == nil {
			continue
		}

		paramObj := param.GetObject()
		if paramObj == nil {
			continue
		}

		name := paramObj.GetName()
		if !isIDParameter(name) {
			continue
		}

		// Check if schema type is integer
		jsonSchema := paramObj.GetSchema()
		if jsonSchema == nil {
			continue
		}

		schema := jsonSchema.GetSchema()
		if schema == nil {
			continue
		}

		types := schema.GetType()
		if len(types) == 0 {
			continue
		}

		// Check if type contains "integer"
		for _, typ := range types {
			if typ == "integer" {
				if rootNode := jsonSchema.GetRootNode(); rootNode != nil {
					errs = append(errs, validation.NewValidationError(
						config.GetSeverity(r.DefaultSeverity()),
						RuleOwaspNoNumericIDs,
						fmt.Errorf("parameter '%s' uses integer type for ID - use random IDs like UUIDs instead of numeric IDs", name),
						rootNode,
					))
				}
				break
			}
		}
	}

	return errs
}
