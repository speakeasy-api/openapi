package oas3

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

var (
	// ErrInlineTimeout is returned when the inline operation times out due to context cancellation or exceeds the maximum number of cycles
	ErrInlineTimeout = errors.New("inline operation timed out")
)

// refInfo tracks information about a reference during inlining
type refInfo struct {
	preserve     bool                  // Whether to preserve this reference (don't inline)
	rewrittenRef string                // The rewritten reference (e.g., #/components/schemas/User -> #/$defs/components_schemas_User)
	schema       *JSONSchema[Concrete] // The resolved schema for this reference
	isCircular   bool                  // Whether this reference is part of a circular chain
}

// cycleCounter tracks the number of recursive calls to prevent infinite loops
type cycleCounter struct {
	cycleCount int64
	maxCycles  int64
}

// increment increments the appropriate counter and checks limits
func (c *cycleCounter) increment() error {
	c.cycleCount++

	if c.cycleCount > c.maxCycles {
		return fmt.Errorf("%w: %d cycles exceeded limit of %d", ErrInlineTimeout, c.cycleCount, c.maxCycles)
	}

	return nil
}

// InlineOptions represents the options available when inlining a JSON Schema.
type InlineOptions struct {
	// ResolveOptions are the options to use when resolving references during inlining.
	ResolveOptions ResolveOptions
	// RemoveUnusedDefs determines whether to remove $defs that are no longer referenced after inlining.
	RemoveUnusedDefs bool
	// MaxCycles sets the maximum number of analyzeReferences and inlineRecursive calls combined.
	// If 0, defaults to 5000000. Set to a higher value for complex schemas with many references.
	MaxCycles int64
}

// Inline transforms a JSON Schema by replacing all $ref references with their actual schema content,
// creating a self-contained schema that doesn't depend on external definitions.
//
// Why use Inline?
//
//   - **Simplify schema distribution**: Create standalone schemas that can be shared without worrying
//     about missing referenced files or definitions
//   - **AI and MCP integration**: Provide complete, self-contained schemas to AI systems and
//     Model Context Protocol (MCP) servers that work better with fully expanded schemas
//   - **Improve tooling compatibility**: Some tools work better with fully expanded schemas rather
//     than ones with references
//   - **Generate documentation**: Create complete schema representations for API documentation
//     where all types are visible inline
//   - **Optimize for specific use cases**: Eliminate the need for reference resolution in
//     performance-critical applications
//   - **Debug schema issues**: See the full expanded schema to understand how references resolve
//
// What you'll get:
//
// Before inlining:
//
//	{
//	  "type": "object",
//	  "properties": {
//	    "user": {"$ref": "#/$defs/User"},
//	    "address": {"$ref": "#/$defs/Address"}
//	  },
//	  "$defs": {
//	    "User": {"type": "object", "properties": {"name": {"type": "string"}}},
//	    "Address": {"type": "object", "properties": {"street": {"type": "string"}}}
//	  }
//	}
//
// After inlining:
//
//	{
//	  "type": "object",
//	  "properties": {
//	    "user": {"type": "object", "properties": {"name": {"type": "string"}}},
//	    "address": {"type": "object", "properties": {"street": {"type": "string"}}}
//	  }
//	}
//
// Handling Circular References:
//
// The function intelligently handles circular references (schemas that reference themselves)
// by preserving them when they're safe to use. A circular reference is considered safe when
// there's an "escape route" that prevents infinite nesting:
//
// ✅ Safe circular reference (optional property):
//
//	{
//	  "type": "object",
//	  "properties": {
//	    "name": {"type": "string"},
//	    "parent": {"$ref": "#/$defs/Node"}  // Optional - can be omitted
//	  },
//	  "required": ["name"]  // parent not required = escape route
//	}
//
// ❌ Unsafe circular reference (required property):
//
//	{
//	  "type": "object",
//	  "properties": {
//	    "name": {"type": "string"},
//	    "parent": {"$ref": "#/$defs/Node"}  // Required - creates infinite nesting
//	  },
//	  "required": ["name", "parent"]  // No escape route!
//	}
//
// When circular references are detected, they're preserved in the $defs section and
// references are rewritten to point to the consolidated definitions.
//
// Example usage:
//
//	// Load a schema with references
//	schema := &JSONSchema[Referenceable]{...}
//
//	// Configure inlining
//	opts := InlineOptions{
//		ResolveOptions: ResolveOptions{
//			RootLocation: "schema.json",
//			RootDocument: schema,
//		},
//		RemoveUnusedDefs: true, // Clean up unused definitions
//	}
//
//	// Inline all references
//	result, err := Inline(ctx, schema, opts)
//	if err != nil {
//		return fmt.Errorf("failed to inline schema: %w", err)
//	}
//
//	// result is now a self-contained schema with all references expanded
//	// Safe circular references are preserved in $defs
//	// Unsafe circular references cause an error
//
// Parameters:
//   - ctx: Context for the operation
//   - schema: The schema to inline
//   - opts: Configuration options for inlining
//
// Returns:
//   - *JSONSchema[Referenceable]: The inlined schema (input schema left unmodified)
//   - error: Any error that occurred, including invalid circular reference errors
func Inline(ctx context.Context, schema *JSONSchema[Referenceable], opts InlineOptions) (*JSONSchema[Referenceable], error) {
	if schema == nil {
		return nil, nil
	}

	// If the working schema is not a reference, try to convert it to a referenced schema
	// This ensures consistent tracking during circular reference detection
	if !schema.IsReference() {
		// Try to get the JSON pointer for this schema within the root document
		if rootDoc, ok := opts.ResolveOptions.RootDocument.(GetRootNoder); ok {
			rootNode := rootDoc.GetRootNode()
			if rootNode != nil {
				jsonPtr := schema.GetCore().GetJSONPointer(rootNode)
				if jsonPtr != "" {
					// Create a referenced schema using the JSON pointer
					ref := references.Reference("#" + jsonPtr)
					schema = NewReferencedScheme(ctx, ref, (*JSONSchema[Concrete])(schema))
				}
			}
		}
	}

	refTracker := sequencedmap.New[string, *refInfo]() // Single source of truth for all reference info

	maxCycles := int64(5000000)
	if opts.MaxCycles > 0 {
		maxCycles = opts.MaxCycles
	}
	counter := &cycleCounter{
		maxCycles: maxCycles,
	}

	// First pass: analyze all references and make preservation decisions
	if err := analyzeReferences(ctx, schema, opts, refTracker, []*loopFrame{}, counter); err != nil {
		return nil, fmt.Errorf("failed to analyze references: %w", err)
	}

	// Second pass: perform actual inlining based on decisions
	workingSchema, err := inlineRecursive(ctx, schema, opts, refTracker, []string{}, counter)
	if err != nil {
		return nil, fmt.Errorf("failed to inline schema: %w", err)
	}

	// Add collected definitions to the top-level schema
	if err := consolidateDefinitions(workingSchema, refTracker); err != nil {
		return nil, fmt.Errorf("failed to consolidate definitions: %w", err)
	}

	// Remove unused $defs if requested
	if opts.RemoveUnusedDefs {
		removeUnusedDefs(ctx, workingSchema, refTracker)
	}

	return workingSchema, nil
}

type loopFrame struct {
	ref                 string
	detectedEscapeRoute bool
}

// analyzeReferences performs the first pass to collect reference usage information
func analyzeReferences(ctx context.Context, schema *JSONSchema[Referenceable], opts InlineOptions, refTracker *sequencedmap.Map[string, *refInfo], visited []*loopFrame, counter *cycleCounter) error {
	if schema == nil {
		return nil
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrInlineTimeout, ctx.Err())
	default:
		// Increment cycle counter and check limits
		if err := counter.increment(); err != nil {
			return err
		}
	}

	// Ensure the schema is resolved before proceeding
	_, err := schema.Resolve(ctx, opts.ResolveOptions)
	if err != nil {
		return fmt.Errorf("failed to resolve schema %s: %w", schema.GetAbsRef(), err)
	}
	resolved := schema.GetResolvedSchema()

	if schema.IsReference() {
		absRef := getAbsRef(schema, opts)

		// Track reference usage using the absolute reference
		info, exists := refTracker.Get(absRef)
		if !exists {
			info = &refInfo{}
			refTracker.Set(absRef, info)
		}

		previousIdx := slices.IndexFunc(visited, func(frame *loopFrame) bool {
			return frame.ref == absRef
		})

		if previousIdx != -1 {
			detectedEscapeRoute := false

			for _, frame := range visited[previousIdx:] {
				if frame.detectedEscapeRoute {
					detectedEscapeRoute = true
					break
				}
			}

			// If we found an escape route, this is a valid circular reference
			if detectedEscapeRoute {
				info.isCircular = true
				info.preserve = true
				// Determine the rewritten reference but don't modify the schema yet
				if info.rewrittenRef == "" {
					info.rewrittenRef = rewriteExternalReference(schema, refTracker)
				}
			} else {
				// Invalid circular reference
				return fmt.Errorf("invalid circular reference %s: %w", absRef, err)
			}
			// Don't continue analyzing circular references
			return nil
		}

		visited = append(visited, &loopFrame{
			ref: absRef,
		})

		// Continue analyzing the resolved schema
		// Important: Use ConcreteToReferenceable to maintain resolution context
		return analyzeReferences(ctx, ConcreteToReferenceable(resolved), opts, refTracker, visited, counter)
	}

	if resolved.IsBool() {
		return nil // Boolean schemas don't have references to analyze
	}

	currentFrame := &loopFrame{}
	if len(visited) > 0 {
		currentFrame = visited[len(visited)-1]
	}

	js := resolved.GetSchema()

	// Analyze all nested schemas
	for _, schema := range js.AllOf {
		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	for _, schema := range js.OneOf {
		currentFrame.detectedEscapeRoute = len(js.OneOf) > 1
		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	for _, schema := range js.AnyOf {
		currentFrame.detectedEscapeRoute = len(js.AnyOf) > 1
		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	for _, schema := range js.PrefixItems {
		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	if err := analyzeReferences(ctx, js.Contains, opts, refTracker, visited, counter); err != nil {
		return err
	}

	if err := analyzeReferences(ctx, js.If, opts, refTracker, visited, counter); err != nil {
		return err
	}

	if err := analyzeReferences(ctx, js.Then, opts, refTracker, visited, counter); err != nil {
		return err
	}

	if err := analyzeReferences(ctx, js.Else, opts, refTracker, visited, counter); err != nil {
		return err
	}

	for _, schema := range js.DependentSchemas.All() {
		currentFrame.detectedEscapeRoute = true
		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	for _, schema := range js.PatternProperties.All() {
		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	if err := analyzeReferences(ctx, js.PropertyNames, opts, refTracker, visited, counter); err != nil {
		return err
	}

	if err := analyzeReferences(ctx, js.UnevaluatedItems, opts, refTracker, visited, counter); err != nil {
		return err
	}

	if err := analyzeReferences(ctx, js.UnevaluatedProperties, opts, refTracker, visited, counter); err != nil {
		return err
	}

	if js.Items != nil {
		currentFrame.detectedEscapeRoute = js.GetMinItems() == 0
		if err := analyzeReferences(ctx, js.Items, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	if err := analyzeReferences(ctx, js.Not, opts, refTracker, visited, counter); err != nil {
		return err
	}

	for property, schema := range js.Properties.All() {
		currentFrame.detectedEscapeRoute = !slices.Contains(js.GetRequired(), property)

		if err := analyzeReferences(ctx, schema, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	if js.AdditionalProperties != nil {
		currentFrame.detectedEscapeRoute = true
		if err := analyzeReferences(ctx, js.AdditionalProperties, opts, refTracker, visited, counter); err != nil {
			return err
		}
	}

	return nil
}

func inlineRecursive(ctx context.Context, schema *JSONSchema[Referenceable], opts InlineOptions, refTracker *sequencedmap.Map[string, *refInfo], visited []string, counter *cycleCounter) (*JSONSchema[Referenceable], error) {
	if schema == nil {
		return nil, nil
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %w", ErrInlineTimeout, ctx.Err())
	default:
		// Increment cycle counter and check limits
		if err := counter.increment(); err != nil {
			return nil, err
		}
	}

	schema = schema.ShallowCopy()
	resolved := ReferenceableToConcrete(schema)

	// Handle references based on pre-computed decisions
	if schema.IsReference() {
		resolved = resolved.MustGetResolvedSchema().ShallowCopy()

		absRef := getAbsRef(schema, opts)

		// Get the pre-computed decision for this reference using the absolute reference
		info, exists := refTracker.Get(absRef)
		if !exists {
			return nil, fmt.Errorf("reference %s not found in analysis phase", absRef)
		}
		if info.schema == nil {
			info.schema = resolved
		}

		// If this reference should be preserved, we still need to process its contents once
		// to inline any non-circular references within it
		if info.preserve {
			previousIdx := slices.Index(visited, absRef)

			// Check if this is a circular reference
			if previousIdx != -1 {
				// This is the second+ occurrence of a circular reference
				// Rewrite the reference if needed, then don't recurse
				if info.rewrittenRef != "" {
					schema.GetSchema().Ref = pointer.From(references.Reference(info.rewrittenRef))
					rewrittenAbsRef := references.Reference(opts.ResolveOptions.TargetLocation + info.rewrittenRef)
					// Add reverse lookup for the rewritten reference
					if !refTracker.Has(rewrittenAbsRef.String()) {
						refTracker.Set(rewrittenAbsRef.String(), info)
					}
				}
				return schema, nil
			}
			// This is the first occurrence - process its contents but don't inline the reference itself
			visited = append(visited, absRef)
		}
		// If not preserve, this reference should be inlined - we'll process its content below
	}
	if resolved.IsBool() {
		inlineSchemaInPlace(schema, resolved)
		return schema, nil
	}

	js := resolved.GetSchema()

	// Walk through allOf schemas
	for i, s := range js.AllOf {
		s, err := inlineRecursive(ctx, s, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.AllOf[i] = s
	}

	// Walk through oneOf schemas
	for i, s := range js.OneOf {
		s, err := inlineRecursive(ctx, s, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.OneOf[i] = s
	}

	// Walk through anyOf schemas
	for i, s := range js.AnyOf {
		s, err := inlineRecursive(ctx, s, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.AnyOf[i] = s
	}

	// Walk through prefixItems schemas
	for i, s := range js.PrefixItems {
		s, err := inlineRecursive(ctx, s, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.PrefixItems[i] = s
	}

	// Visit contains schema
	s, err := inlineRecursive(ctx, js.Contains, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.Contains = s

	// Visit if schema
	s, err = inlineRecursive(ctx, js.If, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.If = s

	// Visit then schema
	s, err = inlineRecursive(ctx, js.Then, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.Then = s

	// Visit else schema
	s, err = inlineRecursive(ctx, js.Else, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.Else = s

	// Walk through dependentSchemas schemas
	for key, schema := range js.DependentSchemas.All() {
		s, err := inlineRecursive(ctx, schema, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.DependentSchemas.Set(key, s)
	}

	// Walk through patternProperties schemas
	for key, schema := range js.PatternProperties.All() {
		s, err := inlineRecursive(ctx, schema, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.PatternProperties.Set(key, s)
	}

	// Visit propertyNames schema
	s, err = inlineRecursive(ctx, js.PropertyNames, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.PropertyNames = s

	// Visit unevaluatedItems schema
	s, err = inlineRecursive(ctx, js.UnevaluatedItems, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.UnevaluatedItems = s

	// Visit unevaluatedProperties schema
	s, err = inlineRecursive(ctx, js.UnevaluatedProperties, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.UnevaluatedProperties = s

	// Visit items schema
	s, err = inlineRecursive(ctx, js.Items, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.Items = s

	// Visit not schema
	s, err = inlineRecursive(ctx, js.Not, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.Not = s

	// Walk through properties schemas
	for key, s := range js.Properties.All() {
		s, err := inlineRecursive(ctx, s, opts, refTracker, visited, counter)
		if err != nil {
			return nil, err
		}
		js.Properties.Set(key, s)
	}

	// Visit additionalProperties schema
	s, err = inlineRecursive(ctx, js.AdditionalProperties, opts, refTracker, visited, counter)
	if err != nil {
		return nil, err
	}
	js.AdditionalProperties = s

	// Handle reference inlining at the end
	if schema.IsReference() {
		absRef := getAbsRef(schema, opts)

		info, exists := refTracker.Get(absRef)
		if !exists {
			return nil, fmt.Errorf("reference %s not found in analysis phase", absRef)
		}

		// If we reach here, this reference should be inlined (preserve=false)
		if !info.preserve {
			inlineSchemaInPlace(schema, resolved)
		} else if info.rewrittenRef != "" {
			// This is a preserved reference - rewrite it to point to the new $defs location
			schema.GetSchema().Ref = pointer.From(references.Reference(info.rewrittenRef))
			rewrittenAbsRef := references.Reference(opts.ResolveOptions.TargetLocation + info.rewrittenRef)
			// Add reverse lookup for the rewritten reference
			if !refTracker.Has(rewrittenAbsRef.String()) {
				refTracker.Set(rewrittenAbsRef.String(), info)
			}
		}
	}

	return schema, nil
}

func getAbsRef(schema *JSONSchema[Referenceable], opts InlineOptions) string {
	absRef := schema.GetAbsRef()
	if absRef.GetURI() == "" {
		absRef = references.Reference(opts.ResolveOptions.TargetLocation + absRef.String())
	}

	// Use absRefStr for consistent tracking across external and internal references
	absRefStr := absRef.String()
	return absRefStr
}

// inlineSchemaInPlace replaces a reference schema with its resolved content in place.
// It includes circular reference detection to prevent infinite recursion.
func inlineSchemaInPlace(schema *JSONSchema[Referenceable], resolved *JSONSchema[Concrete]) {
	if !schema.IsReference() {
		// Not a reference, nothing to inline
		return
	}

	ref := string(schema.GetRef())
	if ref == "" {
		return
	}

	// Replace the current schema's EitherValue with the resolved schema's content
	schema.EitherValue = resolved.EitherValue

	// Clear the reference resolution cache and related fields since we've inlined the content
	schema.referenceResolutionCache = nil
	schema.resolvedSchemaCache = nil
	schema.circularErrorFound = false
	schema.parent = nil
	schema.topLevelParent = nil
}

// removeUnusedDefs removes $defs that are no longer referenced after inlining
func removeUnusedDefs(_ context.Context, schema *JSONSchema[Referenceable], refTracker *sequencedmap.Map[string, *refInfo]) {
	if schema == nil || !schema.IsSchema() {
		return
	}

	schemaObj := schema.GetSchema()
	if schemaObj == nil || schemaObj.Defs == nil || schemaObj.Defs.Len() == 0 {
		return
	}

	// Remove unused definitions
	defsToRemove := make([]string, 0)
	for defName := range schemaObj.Defs.All() {
		defRef := "#/$defs/" + defName

		found := false
		for _, info := range refTracker.All() {
			if info.rewrittenRef == defRef {
				found = true
				break
			}
		}

		if !found {
			defsToRemove = append(defsToRemove, defName)
		}
	}

	for _, defName := range defsToRemove {
		schemaObj.Defs.Delete(defName)
	}

	// If no defs remain, set Defs to nil
	if schemaObj.Defs.Len() == 0 {
		schemaObj.Defs = nil
	}
}

// generateUniqueDefName generates a unique name for a definition to avoid conflicts
func generateUniqueDefName(baseName string, existingDefs map[string]bool) string {
	if _, exists := existingDefs[baseName]; !exists {
		return baseName
	}

	counter := 1
	for {
		uniqueName := fmt.Sprintf("%s_%d", baseName, counter)
		if _, exists := existingDefs[uniqueName]; !exists {
			return uniqueName
		}
		counter++
	}
}

// rewriteExternalReference rewrites external references to top-level $defs and returns the new reference
func rewriteExternalReference(schema *JSONSchema[Referenceable], refTracker *sequencedmap.Map[string, *refInfo]) string {
	if schema == nil || !schema.IsReference() {
		return ""
	}

	ref := schema.GetRef()

	// Check if this is already a $defs reference - if so, no rewriting needed
	if ref.HasJSONPointer() {
		jsonPointer := ref.GetJSONPointer()
		if strings.HasPrefix(string(jsonPointer), "/$defs/") {
			return ref.String()
		}
	}

	// This is an external reference that needs to be rewritten
	var newDefName string

	switch {
	case ref.GetURI() != "":
		// External document reference - use URI + JSON pointer as name
		uri := ref.GetURI()
		// Clean up URI to make it a valid definition name
		newDefName = strings.ReplaceAll(uri, "/", "_")
		newDefName = strings.ReplaceAll(newDefName, ":", "_")
		newDefName = strings.ReplaceAll(newDefName, ".", "_")
		newDefName = strings.ReplaceAll(newDefName, "-", "_")

		if ref.HasJSONPointer() {
			jsonPointer := string(ref.GetJSONPointer())
			// Append JSON pointer to make it unique
			pointerName := strings.ReplaceAll(jsonPointer, "/", "_")
			pointerName = strings.ReplaceAll(pointerName, "~0", "_tilde_")
			pointerName = strings.ReplaceAll(pointerName, "~1", "_slash_")
			newDefName += pointerName
		}

		if newDefName == "" {
			newDefName = "ExternalRef"
		}
	case ref.HasJSONPointer():
		// Internal JSON pointer reference (not $defs)
		jsonPointer := string(ref.GetJSONPointer())

		// Special handling for OpenAPI component references
		if strings.HasPrefix(jsonPointer, "/components/schemas/") {
			// Extract just the schema name
			newDefName = strings.TrimPrefix(jsonPointer, "/components/schemas/")
		} else {
			// Convert JSON pointer to a valid definition name
			newDefName = strings.TrimPrefix(jsonPointer, "/")
			newDefName = strings.ReplaceAll(newDefName, "/", "_")
			newDefName = strings.ReplaceAll(newDefName, "~0", "_tilde_")
			newDefName = strings.ReplaceAll(newDefName, "~1", "_slash_")
		}

		if newDefName == "" {
			newDefName = "InternalRef"
		}
	default:
		// Edge case - reference with no URI and no JSON pointer
		newDefName = "UnknownRef"
	}

	// Generate a unique name to avoid conflicts
	existingDefs := make(map[string]bool)
	for _, info := range refTracker.All() {
		if info.rewrittenRef != "" && strings.HasPrefix(info.rewrittenRef, "#/$defs/") {
			defName := strings.TrimPrefix(info.rewrittenRef, "#/$defs/")
			existingDefs[defName] = true
		}
	}

	uniqueName := generateUniqueDefName(newDefName, existingDefs)
	newRefStr := "#/$defs/" + uniqueName

	return newRefStr
}

// consolidateDefinitions adds all collected definitions to the top-level schema's $defs
func consolidateDefinitions(schema *JSONSchema[Referenceable], refTracker *sequencedmap.Map[string, *refInfo]) error {
	if schema == nil || refTracker.Len() == 0 {
		return nil
	}

	// Ensure we have a schema object (not a boolean schema)
	if schema.IsBool() {
		return errors.New("cannot add definitions to a boolean schema")
	}

	js := schema.GetSchema()
	if js == nil {
		return errors.New("schema object is nil")
	}

	// Count how many definitions we actually need to add
	defsToAdd := sequencedmap.New[string, *JSONSchema[Referenceable]]()

	for originalRef, info := range refTracker.All() {
		if info.preserve {
			// This reference needs to be preserved, so we need its target schema in $defs
			var defName string
			var targetSchema *JSONSchema[Referenceable]

			if info.rewrittenRef != "" {
				// Use the rewritten reference
				if strings.HasPrefix(info.rewrittenRef, "#/$defs/") {
					defName = strings.TrimPrefix(info.rewrittenRef, "#/$defs/")
					targetSchema = ConcreteToReferenceable(info.schema)
				}
			} else if strings.HasPrefix(originalRef, "#/$defs/") {
				// Already a $defs reference, use as-is
				defName = strings.TrimPrefix(originalRef, "#/$defs/")
				targetSchema = ConcreteToReferenceable(info.schema)
			}

			if defName != "" && targetSchema != nil {
				defsToAdd.Set(defName, targetSchema)
			}
		}
	}

	// Only initialize $defs if we have definitions to add
	if defsToAdd.Len() > 0 {
		if js.Defs == nil {
			js.Defs = sequencedmap.New[string, *JSONSchema[Referenceable]]()
		}

		// Add all collected definitions
		for defName, defSchema := range defsToAdd.All() {
			js.Defs.Set(defName, defSchema)
		}
	}

	return nil
}
