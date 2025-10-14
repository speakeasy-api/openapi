package explore

import (
	"context"
	"sort"
	"strings"

	"github.com/speakeasy-api/openapi/openapi"
)

// CollectOperations walks the OpenAPI document and collects all operations
func CollectOperations(ctx context.Context, doc *openapi.OpenAPI) ([]OperationInfo, error) {
	var operations []OperationInfo

	for item := range openapi.Walk(ctx, doc) {
		err := item.Match(openapi.Matcher{
			Operation: func(op *openapi.Operation) error {
				method, path := openapi.ExtractMethodAndPath(item.Location)
				if method == "" || path == "" {
					return nil
				}

				operations = append(operations, OperationInfo{
					Path:        path,
					Method:      strings.ToUpper(method), // Uppercase for display
					OperationID: op.GetOperationID(),
					Summary:     op.GetSummary(),
					Description: op.GetDescription(),
					Tags:        op.GetTags(),
					Deprecated:  op.GetDeprecated(),
					Operation:   op,
					Folded:      true, // Start with details folded
				})
				return nil
			},
		})
		if err != nil {
			return nil, err
		}
	}

	// Sort operations for stable, predictable display
	// First by path, then by method
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Path != operations[j].Path {
			return operations[i].Path < operations[j].Path
		}
		return operations[i].Method < operations[j].Method
	})

	return operations, nil
}
