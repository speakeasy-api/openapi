package openapi_test

import (
	"testing"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OperationInfo holds information about an operation found during walking
type OperationInfo struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
}

func TestWalk_CollectOperations_Success(t *testing.T) {
	t.Parallel()

	openAPIDoc, err := loadOpenAPIDocument(t.Context())
	require.NoError(t, err)

	var collectedOperations []OperationInfo

	// Walk the document and collect all operations
	for item := range openapi.Walk(t.Context(), openAPIDoc) {
		err := item.Match(openapi.Matcher{
			Operation: func(op *openapi.Operation) error {
				// Extract method and path from location context
				method, path := extractMethodAndPath(item.Location)

				if method == "" || path == "" {
					return nil
				}

				operationInfo := OperationInfo{
					OperationID: op.GetOperationID(),
					Method:      method,
					Path:        path,
					Summary:     op.GetSummary(),
				}

				collectedOperations = append(collectedOperations, operationInfo)

				return nil
			},
		})
		require.NoError(t, err)
	}

	// Define expected operations based on the test data (only operations under paths/{path}/{method})
	expectedOperations := []OperationInfo{
		{
			OperationID: "getUser",
			Method:      "get",
			Path:        "/users/{id}",
			Summary:     "Get user by ID",
		},
		{
			OperationID: "updateUser",
			Method:      "put",
			Path:        "/users/{id}",
			Summary:     "Update user",
		},
		{
			OperationID: "deleteUser",
			Method:      "delete",
			Path:        "/users/{id}",
			Summary:     "Delete user",
		},
		{
			OperationID: "listUsers",
			Method:      "get",
			Path:        "/users",
			Summary:     "List users",
		},
		{
			OperationID: "createUser",
			Method:      "post",
			Path:        "/users",
			Summary:     "Create user",
		},
		{
			OperationID: "getPet",
			Method:      "get",
			Path:        "/pets/{petId}",
			Summary:     "Get pet by ID",
		},
		{
			OperationID: "updatePetPartial",
			Method:      "patch",
			Path:        "/pets/{petId}",
			Summary:     "Partially update pet",
		},
		{
			OperationID: "healthCheck",
			Method:      "get",
			Path:        "/health",
			Summary:     "Health check",
		},
	}

	// Assert we found all expected operations
	assert.ElementsMatch(t, expectedOperations, collectedOperations, "should find all expected operations with correct details")
}

// extractMethodAndPath extracts HTTP method and path from location context
func extractMethodAndPath(locations openapi.Locations) (string, string) {
	if len(locations) == 0 {
		return "", ""
	}

	var method, path string

	for i := len(locations) - 1; i >= 0; i-- {
		switch getParentType(locations[i]) {
		case "Paths":
			path = pointer.Value(locations[i].ParentKey)
		case "PathItem":
			method = pointer.Value(locations[i].ParentKey)
		case "OpenAPI":
		default:
			// Matched something unexpected so not likely an operation in paths
			return "", ""
		}
	}

	return method, path
}

func getParentType(location openapi.LocationContext) string {
	parentType := ""
	_ = location.Parent(openapi.Matcher{
		Any: func(a any) error {
			switch a.(type) {
			case *openapi.Paths:
				parentType = "Paths"
			case *openapi.ReferencedPathItem:
				parentType = "PathItem"
			case *openapi.OpenAPI:
				parentType = "OpenAPI"
			default:
				parentType = "Unknown"
			}
			return nil
		},
	})
	return parentType
}
