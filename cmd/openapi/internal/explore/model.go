package explore

import "github.com/speakeasy-api/openapi/openapi"

// OperationInfo represents a single API operation with its metadata
type OperationInfo struct {
	// Path is the endpoint path (e.g., "/users/{id}")
	Path string
	// Method is the HTTP method (e.g., "GET", "POST")
	Method string
	// OperationID is the unique identifier for the operation
	OperationID string
	// Summary is a short summary of what the operation does
	Summary string
	// Description is a verbose explanation of the operation behavior
	Description string
	// Tags is a list of tags for API documentation control
	Tags []string
	// Deprecated indicates if the operation is deprecated
	Deprecated bool

	// Operation is the full operation object for detailed inspection
	Operation *openapi.Operation

	// Folded tracks whether details are hidden in the UI
	Folded bool
}

// GetDisplaySummary returns a display-friendly summary
// Returns the summary if available, otherwise a truncated description
func (o *OperationInfo) GetDisplaySummary() string {
	if o.Summary != "" {
		return o.Summary
	}
	if o.Description != "" && len(o.Description) > 60 {
		return o.Description[:57] + "..."
	}
	return o.Description
}

// HasDetails returns true if the operation has additional details to display
func (o *OperationInfo) HasDetails() bool {
	return o.Summary != "" ||
		o.Description != "" ||
		len(o.Operation.GetParameters()) > 0 ||
		o.Operation.GetRequestBody() != nil ||
		o.Operation.GetResponses() != nil
}
