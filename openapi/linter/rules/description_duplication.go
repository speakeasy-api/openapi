package rules

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

const RuleStyleDescriptionDuplication = "style-description-duplication"

type DescriptionDuplicationRule struct{}

func (r *DescriptionDuplicationRule) ID() string       { return RuleStyleDescriptionDuplication }
func (r *DescriptionDuplicationRule) Category() string { return CategoryStyle }
func (r *DescriptionDuplicationRule) Description() string {
	return "Description and summary fields should not contain identical text within the same node. These fields serve different purposes: summaries provide brief overviews while descriptions offer detailed explanations, so duplicating content provides no additional value to API consumers."
}
func (r *DescriptionDuplicationRule) Link() string {
	return "https://github.com/speakeasy-api/openapi/blob/main/openapi/linter/README.md#style-description-duplication"
}
func (r *DescriptionDuplicationRule) DefaultSeverity() validation.Severity {
	return validation.SeverityWarning
}
func (r *DescriptionDuplicationRule) Versions() []string {
	return nil // Applies to all OpenAPI versions
}

func (r *DescriptionDuplicationRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
	if docInfo == nil || docInfo.Document == nil || docInfo.Index == nil {
		return nil
	}

	var errs []error

	// Use the index to check all nodes that have both description and summary
	for _, node := range docInfo.Index.DescriptionAndSummaryNodes {
		desc := node.Node.GetDescription()
		summary := node.Node.GetSummary()

		// Skip if either is empty
		if desc == "" || summary == "" {
			continue
		}

		// Check if description and summary are identical
		if desc == summary {
			path := node.Location.ToJSONPointer().String()

			// Get the root node from the node (works for both regular types and Reference types)
			type nodeWithRootNode interface {
				GetRootNode() *yaml.Node
			}

			if nodeWithRoot, ok := node.Node.(nodeWithRootNode); ok {
				if rootNode := nodeWithRoot.GetRootNode(); rootNode != nil {
					_, summaryValueNode, found := yml.GetMapElementNodes(ctx, rootNode, "summary")
					if found && summaryValueNode != nil {
						errs = append(errs, validation.NewValidationError(
							config.GetSeverity(r.DefaultSeverity()),
							RuleStyleDescriptionDuplication,
							fmt.Errorf("summary is identical to description in %s", path),
							summaryValueNode,
						))
					}
				}
			}
		}
	}

	return errs
}
