package oas3

import "github.com/speakeasy-api/openapi/references"

// ReferenceChainEntry represents a step in the reference resolution chain.
// Each entry contains the schema that holds the reference and the reference itself.
type ReferenceChainEntry struct {
	// Schema is the JSONSchema node that contains the $ref.
	// This is the schema that was resolved to get to the next step in the chain.
	Schema *JSONSchema[Referenceable]

	// Reference is the $ref value from the schema (e.g., "#/components/schemas/User").
	Reference references.Reference
}

// GetReferenceChain returns the chain of references that were followed to resolve this schema.
// The chain is ordered from the outermost reference (top-level parent) to the innermost (immediate parent).
// Returns nil if this schema was not resolved via references.
//
// Example: If a response schema references Schema1, which references SchemaShared,
// calling GetReferenceChain() on the resolved SchemaShared would return:
//   - [0]: response schema with reference "#/components/schemas/Schema1"
//   - [1]: Schema1 with reference "#/components/schemas/SchemaShared"
//
// This allows tracking which schemas first referenced nested schemas during iteration.
func (j *JSONSchema[T]) GetReferenceChain() []*ReferenceChainEntry {
	if j == nil || j.parent == nil {
		return nil
	}

	var chain []*ReferenceChainEntry
	visited := make(map[*JSONSchema[Referenceable]]bool)

	// Walk from the immediate parent up to the top-level
	current := j.parent
	for current != nil {
		// Detect circular reference in parent chain - stop if we've seen this schema before
		if visited[current] {
			break
		}
		visited[current] = true

		if current.IsReference() {
			entry := &ReferenceChainEntry{
				Schema:    current,
				Reference: current.GetRef(),
			}
			// Prepend to get topLevel first (outer -> inner order)
			chain = append([]*ReferenceChainEntry{entry}, chain...)
		}

		// Move to the parent of current
		current = current.GetParent()
	}

	return chain
}

// GetImmediateReference returns the immediate parent reference that resolved to this schema.
// Returns nil if this schema was not resolved via a reference.
//
// This is a convenience method equivalent to getting the last element of GetReferenceChain().
func (j *JSONSchema[T]) GetImmediateReference() *ReferenceChainEntry {
	if j == nil || j.parent == nil || !j.parent.IsReference() {
		return nil
	}

	return &ReferenceChainEntry{
		Schema:    j.parent,
		Reference: j.parent.GetRef(),
	}
}

// GetTopLevelReference returns the outermost (first) reference in the chain that led to this schema.
// Returns nil if this schema was not resolved via a reference.
//
// This is a convenience method equivalent to getting the first element of GetReferenceChain().
func (j *JSONSchema[T]) GetTopLevelReference() *ReferenceChainEntry {
	if j == nil || j.topLevelParent == nil || !j.topLevelParent.IsReference() {
		return nil
	}

	return &ReferenceChainEntry{
		Schema:    j.topLevelParent,
		Reference: j.topLevelParent.GetRef(),
	}
}
