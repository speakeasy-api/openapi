package openapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// OperationIdentifier uniquely identifies an operation either by operationId or by path and HTTP method.
// Either OperationID should be set, OR both Path and Method should be set.
type OperationIdentifier struct {
	// OperationID identifies the operation by its unique operationId
	OperationID string
	// Path is the endpoint path (e.g., "/users/{id}")
	Path string
	// Method is the HTTP method (e.g., "GET", "POST", "DELETE")
	Method string
}

// Snip removes specified operations from an OpenAPI document and cleans up unused components.
//
// This function removes the specified operations from the document's paths and then automatically
// runs Clean() to remove any components that are no longer referenced after the operations are removed.
//
// Why use Snip?
//
//   - **Reduce API surface**: Remove deprecated or unwanted operations from specifications
//   - **Create filtered specs**: Generate subsets of your API for specific use cases
//   - **Clean up documentation**: Remove internal-only endpoints before publishing
//   - **Prepare for migration**: Remove old endpoints when planning API changes
//   - **Generate client SDKs**: Create focused specifications for specific client needs
//
// What Snip does:
//
//  1. Removes each specified operation from its path item
//  2. Removes path items that become empty after operation removal
//  3. Automatically runs Clean() to remove unused components
//
// The operations to remove are specified by path and HTTP method. If all operations
// are removed from a path, the entire path item is removed from the document.
//
// Example usage:
//
//	// Define operations to remove by path and method
//	operationsToRemove := []OperationIdentifier{
//		{Path: "/users/{id}", Method: "DELETE"},
//		{Path: "/admin/debug", Method: "GET"},
//	}
//
//	// Or by operationId
//	operationsToRemove := []OperationIdentifier{
//		{OperationID: "deleteUser"},
//		{OperationID: "getDebugInfo"},
//	}
//
//	// Remove operations and clean up (modifies doc in place)
//	removed, err := Snip(ctx, doc, operationsToRemove)
//	if err != nil {
//		return fmt.Errorf("failed to snip operations: %w", err)
//	}
//
//	fmt.Printf("Removed %d operations\n", removed)
//	// doc now has the specified operations removed and unused components cleaned up
//
// Parameters:
//   - ctx: Context for the operation
//   - doc: The OpenAPI document to modify (modified in place)
//   - operations: Slice of OperationIdentifier specifying which operations to remove
//
// Returns:
//   - int: Number of operations actually removed
//   - error: Any error that occurred during the operation
func Snip(ctx context.Context, doc *OpenAPI, operations []OperationIdentifier) (int, error) {
	if doc == nil {
		return 0, errors.New("document cannot be nil")
	}

	if doc.Paths == nil || doc.Paths.Len() == 0 {
		return 0, nil // Nothing to remove
	}

	if len(operations) == 0 {
		return 0, nil // Nothing to remove
	}

	removedCount := 0

	// Remove each specified operation
	for _, op := range operations {
		// If OperationID is specified, find by ID first
		if op.OperationID != "" {
			if removed := removeOperationByID(doc, op.OperationID); removed {
				removedCount++
			}
		} else if op.Path != "" && op.Method != "" {
			// Otherwise use path and method
			if removed := removeOperation(doc, op.Path, op.Method); removed {
				removedCount++
			}
		}
	}

	// Clean up unused components after removing operations
	if err := Clean(ctx, doc); err != nil {
		return removedCount, fmt.Errorf("failed to clean unused components: %w", err)
	}

	return removedCount, nil
}

// removeOperationByID removes an operation by its operationId
// Returns true if the operation was found and removed, false otherwise
func removeOperationByID(doc *OpenAPI, operationID string) bool {
	if doc.Paths == nil {
		return false
	}

	// Search through all paths and operations to find matching operationId
	for path, pathItem := range doc.Paths.All() {
		if pathItem == nil || pathItem.Object == nil {
			continue
		}

		// Check each HTTP method in this path
		for method, operation := range pathItem.Object.All() {
			if operation != nil && operation.GetOperationID() == operationID {
				// Found it - remove this operation
				return removeOperation(doc, path, string(method))
			}
		}
	}

	return false
}

// removeOperation removes a single operation from the document by path and method
// Returns true if the operation was found and removed, false otherwise
func removeOperation(doc *OpenAPI, path, method string) bool {
	if doc.Paths == nil {
		return false
	}

	// Get the path item
	pathItem, exists := doc.Paths.Get(path)
	if !exists || pathItem == nil || pathItem.Object == nil {
		return false
	}

	// Convert method string to HTTPMethod type (lowercase to match constants)
	httpMethod := HTTPMethod(strings.ToLower(method))

	// Check if the operation exists
	operation := pathItem.Object.GetOperation(httpMethod)
	if operation == nil {
		return false
	}

	// Remove the operation from the embedded map
	// We need to access the Map field directly since PathItem has its own Delete() method
	pathItem.Object.Map.Delete(httpMethod)

	// If the path item has no more operations, remove the entire path
	if pathItem.Object.Len() == 0 {
		doc.Paths.Delete(path)
	}

	return true
}
