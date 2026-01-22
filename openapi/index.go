package openapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/validation"
	"gopkg.in/yaml.v3"
)

// CircularClassification represents the classification of a circular reference.
type CircularClassification int

const (
	// CircularUnclassified means the circular reference has not been classified yet.
	CircularUnclassified CircularClassification = iota
	// CircularValid means the circular reference is valid (has a termination point).
	CircularValid
	// CircularInvalid means the circular reference is invalid (no termination point).
	CircularInvalid
	// CircularPending means the circular reference is part of polymorphic and needs post-processing.
	CircularPending
)

// CircularPathSegment represents a segment of the path through the schema tree.
// It captures constraint information needed to determine if a circular reference can terminate.
type CircularPathSegment struct {
	Field         string // e.g., "properties", "items", "allOf", "oneOf", "anyOf", "additionalProperties"
	PropertyName  string // Set if Field == "properties"
	IsRequired    bool   // Set if this property is in parent's Required array
	ArrayMinItems int64  // Parent's MinItems value (0 means empty array terminates)
	MinProperties int64  // Parent's MinProperties value (0 means empty object terminates)
	BranchIndex   int    // Index in oneOf/anyOf/allOf array
	IsNullable    bool   // True if this schema allows null (termination point)
}

// SchemaVisitInfo tracks the visitation state of a schema during indexing.
type SchemaVisitInfo struct {
	Location      Locations              // Location where first seen
	InCurrentPath bool                   // True while actively walking this schema's children
	CircularType  CircularClassification // Classification result
}

// PolymorphicCircularRef tracks a polymorphic schema with recursive branches.
// Used for post-processing to determine if all branches recurse.
type PolymorphicCircularRef struct {
	ParentSchema   *oas3.JSONSchemaReferenceable  // The parent with oneOf/anyOf/allOf
	ParentLocation Locations                      // Location of the parent
	Field          string                         // "oneOf", "anyOf", or "allOf"
	BranchResults  map[int]CircularClassification // Index -> classification per branch
	TotalBranches  int                            // Total number of branches
}

// referenceStackEntry tracks a schema in the active reference resolution chain.
// Uses JSON pointer strings for identity to handle type differences.
type referenceStackEntry struct {
	refTarget string    // The $ref target (JSON pointer or URI)
	location  Locations // Where this reference was encountered
}

type Descriptioner interface {
	GetDescription() string
}

type Summarizer interface {
	GetSummary() string
}

type DescriptionAndSummary interface {
	GetDescription() string
	GetSummary() string
}

func (i *Index) currentDocumentPath() string {
	if i == nil {
		return ""
	}
	if len(i.currentDocumentStack) == 0 {
		return ""
	}
	return i.currentDocumentStack[len(i.currentDocumentStack)-1]
}

// Index represents a pre-computed index of an OpenAPI document.
// It provides efficient access to document elements without repeated full traversals.
type Index struct {
	Doc *OpenAPI

	ExternalDocumentation []*IndexNode[*oas3.ExternalDocumentation] // All external documentation nodes

	Tags []*IndexNode[*Tag] // All tags defined in the document

	Servers         []*IndexNode[*Server]         // All servers defined in the document
	ServerVariables []*IndexNode[*ServerVariable] // All server variables from all servers

	BooleanSchemas   []*IndexNode[*oas3.JSONSchemaReferenceable] // Boolean schema values (true/false)
	InlineSchemas    []*IndexNode[*oas3.JSONSchemaReferenceable] // Schemas defined inline (properties, items, etc.)
	ComponentSchemas []*IndexNode[*oas3.JSONSchemaReferenceable] // Schemas in /components/schemas/ of main document
	ExternalSchemas  []*IndexNode[*oas3.JSONSchemaReferenceable] // Top-level schemas in external documents
	SchemaReferences []*IndexNode[*oas3.JSONSchemaReferenceable] // All $ref pointers

	InlinePathItems    []*IndexNode[*ReferencedPathItem] // PathItems defined inline (in paths map)
	ComponentPathItems []*IndexNode[*ReferencedPathItem] // PathItems in /components/pathItems/
	ExternalPathItems  []*IndexNode[*ReferencedPathItem] // Top-level PathItems in external documents
	PathItemReferences []*IndexNode[*ReferencedPathItem] // All PathItem $ref pointers

	Operations []*IndexNode[*Operation] // All operations (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, etc.)

	InlineParameters    []*IndexNode[*ReferencedParameter] // Parameters defined inline in operations/path items
	ComponentParameters []*IndexNode[*ReferencedParameter] // Parameters in /components/parameters/
	ParameterReferences []*IndexNode[*ReferencedParameter] // All Parameter $ref pointers

	Responses []*IndexNode[*Responses] // All Responses containers (operation.responses)

	InlineResponses    []*IndexNode[*ReferencedResponse] // Responses defined inline in operations
	ComponentResponses []*IndexNode[*ReferencedResponse] // Responses in /components/responses/
	ResponseReferences []*IndexNode[*ReferencedResponse] // All Response $ref pointers

	InlineRequestBodies    []*IndexNode[*ReferencedRequestBody] // RequestBodies defined inline in operations
	ComponentRequestBodies []*IndexNode[*ReferencedRequestBody] // RequestBodies in /components/requestBodies/
	RequestBodyReferences  []*IndexNode[*ReferencedRequestBody] // All RequestBody $ref pointers

	InlineHeaders    []*IndexNode[*ReferencedHeader] // Headers defined inline
	ComponentHeaders []*IndexNode[*ReferencedHeader] // Headers in /components/headers/
	HeaderReferences []*IndexNode[*ReferencedHeader] // All Header $ref pointers

	InlineExamples    []*IndexNode[*ReferencedExample] // Examples defined inline
	ComponentExamples []*IndexNode[*ReferencedExample] // Examples in /components/examples/
	ExampleReferences []*IndexNode[*ReferencedExample] // All Example $ref pointers

	InlineLinks    []*IndexNode[*ReferencedLink] // Links defined inline in responses
	ComponentLinks []*IndexNode[*ReferencedLink] // Links in /components/links/
	LinkReferences []*IndexNode[*ReferencedLink] // All Link $ref pointers

	InlineCallbacks    []*IndexNode[*ReferencedCallback] // Callbacks defined inline in operations
	ComponentCallbacks []*IndexNode[*ReferencedCallback] // Callbacks in /components/callbacks/
	CallbackReferences []*IndexNode[*ReferencedCallback] // All Callback $ref pointers

	ComponentSecuritySchemes []*IndexNode[*ReferencedSecurityScheme] // SecuritySchemes in /components/securitySchemes/
	SecuritySchemeReferences []*IndexNode[*ReferencedSecurityScheme] // All SecurityScheme $ref pointers
	SecurityRequirements     []*IndexNode[*SecurityRequirement]      // All security requirement objects

	Discriminators []*IndexNode[*oas3.Discriminator] // All discriminator objects in schemas
	XMLs           []*IndexNode[*oas3.XML]           // All XML metadata in schemas
	MediaTypes     []*IndexNode[*MediaType]          // All media types in request/response bodies
	Encodings      []*IndexNode[*Encoding]           // All encoding objects in media types
	OAuthFlows     []*IndexNode[*OAuthFlows]         // All OAuth flows containers
	OAuthFlowItems []*IndexNode[*OAuthFlow]          // Individual OAuth flow objects (implicit, password, clientCredentials, authorizationCode)

	DescriptionNodes         []*IndexNode[Descriptioner]         // All nodes that have a Description field
	SummaryNodes             []*IndexNode[Summarizer]            // All nodes that have a Summary field
	DescriptionAndSummaryNodes []*IndexNode[DescriptionAndSummary] // All nodes that have both Description and Summary fields

	validationErrs []error
	resolutionErrs []error
	circularErrs   []error

	resolveOpts references.ResolveOptions

	// Circular reference tracking (internal)
	indexedSchemas       map[*oas3.JSONSchemaReferenceable]bool // Tracks which schemas have been fully indexed
	referenceStack       []referenceStackEntry                  // Active reference resolution chain (by ref target)
	polymorphicRefs      []*PolymorphicCircularRef              // Pending polymorphic circulars
	visitedRefs          map[string]bool                        // Tracks visited ref targets to avoid duplicates
	currentDocumentStack []string                               // Stack of document paths being walked (for determining external vs main)
}

// IndexNode wraps a node with its location in the document.
type IndexNode[T any] struct {
	Node T

	Location Locations
}

// BuildIndex creates a new Index by walking the entire OpenAPI document.
// It resolves references and detects circular reference patterns.
// Requires resolveOpts to have RootDocument, TargetDocument, and TargetLocation set.
func BuildIndex(ctx context.Context, doc *OpenAPI, resolveOpts references.ResolveOptions) *Index {
	if resolveOpts.RootDocument == nil {
		panic("BuildIndex: resolveOpts.RootDocument is required")
	}
	if resolveOpts.TargetDocument == nil {
		panic("BuildIndex: resolveOpts.TargetDocument is required")
	}
	if resolveOpts.TargetLocation == "" {
		panic("BuildIndex: resolveOpts.TargetLocation is required")
	}

	idx := &Index{
		Doc:                  doc,
		resolveOpts:          resolveOpts,
		indexedSchemas:       make(map[*oas3.JSONSchemaReferenceable]bool),
		referenceStack:       make([]referenceStackEntry, 0),
		polymorphicRefs:      make([]*PolymorphicCircularRef, 0),
		visitedRefs:          make(map[string]bool),
		currentDocumentStack: []string{resolveOpts.TargetLocation}, // Start with main document
	}

	// Phase 1: Walk and index everything
	_ = buildIndex(ctx, idx, doc)

	// Phase 2: Post-process polymorphic circular refs
	idx.finalizePolymorphicCirculars()

	return idx
}

// GetAllSchemas returns all schemas in the index (boolean, inline, component, external, and references).
func (i *Index) GetAllSchemas() []*IndexNode[*oas3.JSONSchemaReferenceable] {
	if i == nil {
		return nil
	}

	allSchemas := make([]*IndexNode[*oas3.JSONSchemaReferenceable], 0, len(i.BooleanSchemas)+
		len(i.InlineSchemas)+
		len(i.ComponentSchemas)+
		len(i.ExternalSchemas)+
		len(i.SchemaReferences),
	)
	allSchemas = append(allSchemas, i.BooleanSchemas...)
	allSchemas = append(allSchemas, i.InlineSchemas...)
	allSchemas = append(allSchemas, i.ComponentSchemas...)
	allSchemas = append(allSchemas, i.ExternalSchemas...)
	allSchemas = append(allSchemas, i.SchemaReferences...)
	return allSchemas
}

// GetAllPathItems returns all path items in the index (inline, component, and references).
func (i *Index) GetAllPathItems() []*IndexNode[*ReferencedPathItem] {
	if i == nil {
		return nil
	}

	allPathItems := make([]*IndexNode[*ReferencedPathItem], 0, len(i.InlinePathItems)+
		len(i.ComponentPathItems)+
		len(i.ExternalPathItems)+
		len(i.PathItemReferences),
	)
	allPathItems = append(allPathItems, i.InlinePathItems...)
	allPathItems = append(allPathItems, i.ComponentPathItems...)
	allPathItems = append(allPathItems, i.ExternalPathItems...)
	allPathItems = append(allPathItems, i.PathItemReferences...)
	return allPathItems
}

// GetValidationErrors returns validation errors from resolution operations.
func (i *Index) GetValidationErrors() []error {
	if i == nil {
		return nil
	}
	return i.validationErrs
}

// GetResolutionErrors returns errors from failed reference resolution.
func (i *Index) GetResolutionErrors() []error {
	if i == nil {
		return nil
	}
	return i.resolutionErrs
}

// GetCircularReferenceErrors returns invalid (non-terminating) circular reference errors.
func (i *Index) GetCircularReferenceErrors() []error {
	if i == nil {
		return nil
	}
	return i.circularErrs
}

// GetAllErrors returns all errors collected during indexing.
func (i *Index) GetAllErrors() []error {
	if i == nil {
		return nil
	}
	all := make([]error, 0, len(i.validationErrs)+len(i.resolutionErrs)+len(i.circularErrs))
	all = append(all, i.validationErrs...)
	all = append(all, i.resolutionErrs...)
	all = append(all, i.circularErrs...)
	return all
}

// HasErrors returns true if any errors were collected during indexing.
func (i *Index) HasErrors() bool {
	if i == nil {
		return false
	}
	return len(i.validationErrs) > 0 || len(i.resolutionErrs) > 0 || len(i.circularErrs) > 0
}

func buildIndex[T any](ctx context.Context, index *Index, obj *T) error {
	for item := range Walk(ctx, obj) {
		if err := item.Match(Matcher{
			ExternalDocs: func(ed *oas3.ExternalDocumentation) error {
				index.indexExternalDocs(ctx, item.Location, ed)
				return nil
			},
			Tag:            func(t *Tag) error { index.indexTag(ctx, item.Location, t); return nil },
			Server:         func(s *Server) error { index.indexServer(ctx, item.Location, s); return nil },
			ServerVariable: func(sv *ServerVariable) error { index.indexServerVariable(ctx, item.Location, sv); return nil },
			ReferencedPathItem: func(rpi *ReferencedPathItem) error {
				index.indexReferencedPathItem(ctx, item.Location, rpi)
				return nil
			},
			ReferencedParameter: func(rp *ReferencedParameter) error {
				index.indexReferencedParameter(ctx, item.Location, rp)
				return nil
			},
			Schema: func(j *oas3.JSONSchemaReferenceable) error {
				return index.indexSchema(ctx, item.Location, j)
			},
			Discriminator: func(d *oas3.Discriminator) error {
				index.indexDiscriminator(ctx, item.Location, d)
				return nil
			},
			XML: func(x *oas3.XML) error {
				index.indexXML(ctx, item.Location, x)
				return nil
			},
			MediaType: func(mt *MediaType) error {
				index.indexMediaType(ctx, item.Location, mt)
				return nil
			},
			Encoding: func(enc *Encoding) error {
				index.indexEncoding(ctx, item.Location, enc)
				return nil
			},
			ReferencedHeader: func(rh *ReferencedHeader) error {
				index.indexReferencedHeader(ctx, item.Location, rh)
				return nil
			},
			ReferencedExample: func(re *ReferencedExample) error {
				index.indexReferencedExample(ctx, item.Location, re)
				return nil
			},
			Operation: func(op *Operation) error {
				index.indexOperation(ctx, item.Location, op)
				return nil
			},
			ReferencedRequestBody: func(rb *ReferencedRequestBody) error {
				index.indexReferencedRequestBody(ctx, item.Location, rb)
				return nil
			},
			Responses: func(r *Responses) error {
				index.indexResponses(ctx, item.Location, r)
				return nil
			},
			ReferencedResponse: func(rr *ReferencedResponse) error {
				index.indexReferencedResponse(ctx, item.Location, rr)
				return nil
			},
			ReferencedLink: func(rl *ReferencedLink) error {
				index.indexReferencedLink(ctx, item.Location, rl)
				return nil
			},
			ReferencedCallback: func(rc *ReferencedCallback) error {
				index.indexReferencedCallback(ctx, item.Location, rc)
				return nil
			},
			ReferencedSecurityScheme: func(rss *ReferencedSecurityScheme) error {
				index.indexReferencedSecurityScheme(ctx, item.Location, rss)
				return nil
			},
			Security: func(req *SecurityRequirement) error {
				index.indexSecurityRequirement(ctx, item.Location, req)
				return nil
			},
			OAuthFlows: func(of *OAuthFlows) error {
				index.indexOAuthFlows(ctx, item.Location, of)
				return nil
			},
			OAuthFlow: func(of *OAuthFlow) error {
				index.indexOAuthFlow(ctx, item.Location, of)
				return nil
			},
			Any: func(a any) error {
				if d, ok := a.(Descriptioner); ok {
					index.indexDescriptionNode(ctx, item.Location, d)
				}
				if s, ok := a.(Summarizer); ok {
					index.indexSummaryNode(ctx, item.Location, s)
				}
				if ds, ok := a.(DescriptionAndSummary); ok {
					index.indexDescriptionAndSummaryNode(ctx, item.Location, ds)
				}
				return nil
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (i *Index) indexSchema(ctx context.Context, loc Locations, schema *oas3.JSONSchemaReferenceable) error {
	// Resolve if needed (do this first to get the resolved schema for tracking)
	if !schema.IsResolved() {
		vErrs, err := schema.Resolve(ctx, i.resolveOpts)
		if err != nil {
			i.resolutionErrs = append(i.resolutionErrs, validation.NewValidationErrorWithDocumentLocation(
				validation.SeverityError,
				"resolution-json-schema",
				err,
				getSchemaErrorNode(schema),
				i.documentPathForSchema(schema),
			))
			return nil
		}
		i.validationErrs = append(i.validationErrs, i.applyDocumentLocation(vErrs, i.documentPathForSchema(schema))...)
		if resolved := schema.GetResolvedSchema(); resolved != nil && i.Doc != nil {
			opts := i.referenceValidationOptions()
			schemaErrs := resolved.Validate(ctx, opts...)
			i.validationErrs = append(i.validationErrs, i.applyDocumentLocation(schemaErrs, i.documentPathForSchema(schema))...)
		}
	}

	// Index the schema based on its type
	if schema.IsBool() {
		if !i.indexedSchemas[schema] {
			i.BooleanSchemas = append(i.BooleanSchemas, &IndexNode[*oas3.JSONSchemaReferenceable]{
				Node:     schema,
				Location: loc,
			})
			i.indexedSchemas[schema] = true
		}
		return nil
	}

	if schema.IsReference() {
		// Add to references list (allow duplicates at different locations)
		i.SchemaReferences = append(i.SchemaReferences, &IndexNode[*oas3.JSONSchemaReferenceable]{
			Node:     schema,
			Location: loc,
		})

		// Get the $ref target for tracking
		refTarget := getRefTarget(schema)
		if refTarget == "" {
			return nil // Can't track without a ref target
		}

		// IMPORTANT: Check circular reference BEFORE walking
		// A schema might be visited AND currently in the reference stack (circular case)
		for stackIdx, entry := range i.referenceStack {
			if entry.refTarget == refTarget {
				// CIRCULAR REFERENCE DETECTED - this is the SECOND+ encounter
				// Build path segments from first occurrence to current
				pathSegments := i.buildPathSegmentsFromStack(stackIdx, loc)
				externalDocumentPath := ""
				currentDocPath := i.currentDocumentPath()
				if currentDocPath != i.resolveOpts.TargetLocation {
					externalDocumentPath = currentDocPath
				}
				circularChain := i.buildCircularReferenceChain(stackIdx, refTarget)

				// Classify the circular reference
				classification, polymorphicInfo := i.classifyCircularPath(schema, pathSegments, loc)

				if classification == CircularInvalid {
					err := fmt.Errorf("non-terminating circular reference detected: %s", joinReferenceChainWithArrows(circularChain))
					i.circularErrs = append(i.circularErrs, validation.NewValidationErrorWithDocumentLocation(
						validation.SeverityError,
						"circular-reference-invalid",
						err,
						getSchemaErrorNode(schema),
						externalDocumentPath,
					))
				} else if classification == CircularPending && polymorphicInfo != nil {
					i.recordPolymorphicBranch(polymorphicInfo)
				}
				// CircularValid - no action needed

				// Stop processing this branch - don't walk the same schema again
				return nil
			}
		}

		// Get the document path for the resolved schema
		info := schema.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push ref target onto reference stack
		i.referenceStack = append(i.referenceStack, referenceStackEntry{
			refTarget: refTarget,
			location:  copyLocations(loc),
		})

		// Push document path onto document stack BEFORE walking
		// This allows nested resolved documents (including returning to main) to
		// attribute errors to the correct document.
		currentDoc := ""
		if len(i.currentDocumentStack) > 0 {
			currentDoc = i.currentDocumentStack[len(i.currentDocumentStack)-1]
		}
		if docPath != "" && docPath != currentDoc {
			i.currentDocumentStack = append(i.currentDocumentStack, docPath)
			defer func() {
				// Pop from document stack
				if len(i.currentDocumentStack) > 1 {
					i.currentDocumentStack = i.currentDocumentStack[:len(i.currentDocumentStack)-1]
				}
			}()
		}

		// Get the resolved schema and recursively walk it
		// Walk API doesn't walk resolved references automatically - we must walk them
		resolved := schema.GetResolvedSchema()
		if resolved != nil {
			// Convert Concrete to Referenceable for walking
			refableResolved := oas3.ConcreteToReferenceable(resolved)
			if err := buildIndex(ctx, i, refableResolved); err != nil {
				i.referenceStack = i.referenceStack[:len(i.referenceStack)-1]
				return err
			}
		}

		// Pop from reference stack
		i.referenceStack = i.referenceStack[:len(i.referenceStack)-1]

		return nil
	}

	// Non-reference schema (component, external, or inline)
	// Note: We don't use indexedSchemas check here because schemas can be referenced
	// from multiple paths and should be indexed for each occurrence

	// Check if this is a top-level component in the main document
	if isTopLevelComponent(loc, "schemas") {
		if !i.indexedSchemas[schema] {
			i.ComponentSchemas = append(i.ComponentSchemas, &IndexNode[*oas3.JSONSchemaReferenceable]{
				Node:     schema,
				Location: loc,
			})
			i.indexedSchemas[schema] = true
		}
		return nil
	}

	// Check if this is a top-level schema in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument(schema) && !i.indexedSchemas[schema] {
			i.ExternalSchemas = append(i.ExternalSchemas, &IndexNode[*oas3.JSONSchemaReferenceable]{
				Node:     schema,
				Location: loc,
			})
			i.indexedSchemas[schema] = true
		}
		return nil
	}

	// Everything else is an inline schema
	// Inline schemas can appear multiple times (e.g., same property type in different schemas)
	// but we only index each unique schema object once
	if !i.indexedSchemas[schema] {
		i.InlineSchemas = append(i.InlineSchemas, &IndexNode[*oas3.JSONSchemaReferenceable]{
			Node:     schema,
			Location: loc,
		})
		i.indexedSchemas[schema] = true
	}

	return nil
}

// isTopLevelExternalSchema checks if the location represents a top-level schema
// in an external document (i.e., at the root of an external document, not under /components/).
func isTopLevelExternalSchema(loc Locations) bool {
	// Top-level external schemas appear at location "/" (root of external doc)
	// They have 0 location contexts (empty Locations slice)
	if len(loc) == 0 {
		return true
	}

	// Single context with no ParentField (or empty ParentField) also indicates root
	if len(loc) == 1 && loc[0].ParentField == "" {
		return true
	}

	return false
}

// isFromMainDocument checks if we're currently walking the main document
// by checking the current document stack.
func (i *Index) isFromMainDocument(_ *oas3.JSONSchemaReferenceable) bool {
	if len(i.currentDocumentStack) == 0 {
		return true // Safety fallback - assume main document
	}

	currentDoc := i.currentDocumentStack[len(i.currentDocumentStack)-1]
	mainDoc := i.resolveOpts.TargetLocation

	return currentDoc == mainDoc
}

// buildPathSegmentsFromStack builds path segments from a point in the reference stack to current location.
func (i *Index) buildPathSegmentsFromStack(startStackIdx int, currentLoc Locations) []CircularPathSegment {
	// Collect all locations from the stack starting point plus current
	var segments []CircularPathSegment

	// Add segments from each stack entry after the circular start point
	for stackIdx := startStackIdx; stackIdx < len(i.referenceStack); stackIdx++ {
		entry := i.referenceStack[stackIdx]
		for _, locCtx := range entry.location {
			segments = append(segments, buildPathSegment(locCtx))
		}
	}

	// Add segments from current location
	for _, locCtx := range currentLoc {
		segments = append(segments, buildPathSegment(locCtx))
	}

	return segments
}

func (i *Index) buildCircularReferenceChain(startStackIdx int, refTarget string) []string {
	chain := make([]string, 0, len(i.referenceStack)-startStackIdx+1)
	for stackIdx := startStackIdx; stackIdx < len(i.referenceStack); stackIdx++ {
		chain = append(chain, i.referenceStack[stackIdx].refTarget)
	}
	chain = append(chain, refTarget)
	return chain
}

func (i *Index) indexExternalDocs(_ context.Context, loc Locations, ed *oas3.ExternalDocumentation) {
	i.ExternalDocumentation = append(i.ExternalDocumentation, &IndexNode[*oas3.ExternalDocumentation]{
		Node:     ed,
		Location: loc,
	})
}

func (i *Index) indexTag(_ context.Context, loc Locations, tag *Tag) {
	i.Tags = append(i.Tags, &IndexNode[*Tag]{
		Node:     tag,
		Location: loc,
	})
}

func (i *Index) indexServer(_ context.Context, loc Locations, server *Server) {
	i.Servers = append(i.Servers, &IndexNode[*Server]{
		Node:     server,
		Location: loc,
	})
}

func (i *Index) indexServerVariable(_ context.Context, loc Locations, serverVariable *ServerVariable) {
	i.ServerVariables = append(i.ServerVariables, &IndexNode[*ServerVariable]{
		Node:     serverVariable,
		Location: loc,
	})
}

func (i *Index) indexReferencedPathItem(ctx context.Context, loc Locations, pathItem *ReferencedPathItem) {
	if pathItem == nil {
		return
	}

	if pathItem.IsReference() && !pathItem.IsResolved() {
		resolveAndValidateReference(i, ctx, pathItem)
	}

	// Index description and summary if both are present
	// For PathItems wrapped in References, we need to get the underlying PathItem
	obj := pathItem.GetObject()
	if obj != nil {
		desc := obj.GetDescription()
		summary := obj.GetSummary()

		if desc != "" {
			i.indexDescriptionNode(ctx, loc, obj)
		}
		if summary != "" {
			i.indexSummaryNode(ctx, loc, obj)
		}
		if desc != "" && summary != "" {
			i.indexDescriptionAndSummaryNode(ctx, loc, obj)
		}
	}

	// Categorize path items similarly to schemas
	if pathItem.IsReference() {
		i.PathItemReferences = append(i.PathItemReferences, &IndexNode[*ReferencedPathItem]{
			Node:     pathItem,
			Location: loc,
		})
		return
	}

	// Check if this is a component path item
	if isTopLevelComponent(loc, "pathItems") {
		i.ComponentPathItems = append(i.ComponentPathItems, &IndexNode[*ReferencedPathItem]{
			Node:     pathItem,
			Location: loc,
		})
		return
	}

	// Check if this is a top-level path item in an external document
	// External path items appear at location "/" (root of external doc)
	if isTopLevelExternalSchema(loc) {
		i.ExternalPathItems = append(i.ExternalPathItems, &IndexNode[*ReferencedPathItem]{
			Node:     pathItem,
			Location: loc,
		})
		return
	}

	// Everything else is an inline path item
	i.InlinePathItems = append(i.InlinePathItems, &IndexNode[*ReferencedPathItem]{
		Node:     pathItem,
		Location: loc,
	})
}

func (i *Index) indexOperation(_ context.Context, loc Locations, operation *Operation) {
	if operation == nil {
		return
	}
	i.Operations = append(i.Operations, &IndexNode[*Operation]{
		Node:     operation,
		Location: loc,
	})
}

func (i *Index) indexReferencedParameter(ctx context.Context, loc Locations, param *ReferencedParameter) {
	if param == nil {
		return
	}

	if param.IsReference() && !param.IsResolved() {
		resolveAndValidateReference(i, ctx, param)
	}

	if param.IsReference() {
		i.ParameterReferences = append(i.ParameterReferences, &IndexNode[*ReferencedParameter]{
			Node:     param,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "parameters") {
		i.ComponentParameters = append(i.ComponentParameters, &IndexNode[*ReferencedParameter]{
			Node:     param,
			Location: loc,
		})
		return
	}

	i.InlineParameters = append(i.InlineParameters, &IndexNode[*ReferencedParameter]{
		Node:     param,
		Location: loc,
	})
}

func (i *Index) indexResponses(_ context.Context, loc Locations, responses *Responses) {
	if responses == nil {
		return
	}
	i.Responses = append(i.Responses, &IndexNode[*Responses]{
		Node:     responses,
		Location: loc,
	})
}

func (i *Index) indexReferencedResponse(ctx context.Context, loc Locations, resp *ReferencedResponse) {
	if resp == nil {
		return
	}

	if resp.IsReference() && !resp.IsResolved() {
		resolveAndValidateReference(i, ctx, resp)
	}

	if resp.IsReference() {
		i.ResponseReferences = append(i.ResponseReferences, &IndexNode[*ReferencedResponse]{
			Node:     resp,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "responses") {
		i.ComponentResponses = append(i.ComponentResponses, &IndexNode[*ReferencedResponse]{
			Node:     resp,
			Location: loc,
		})
		return
	}

	i.InlineResponses = append(i.InlineResponses, &IndexNode[*ReferencedResponse]{
		Node:     resp,
		Location: loc,
	})
}

func (i *Index) indexReferencedRequestBody(ctx context.Context, loc Locations, rb *ReferencedRequestBody) {
	if rb == nil {
		return
	}

	if rb.IsReference() && !rb.IsResolved() {
		resolveAndValidateReference(i, ctx, rb)
	}

	if rb.IsReference() {
		i.RequestBodyReferences = append(i.RequestBodyReferences, &IndexNode[*ReferencedRequestBody]{
			Node:     rb,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "requestBodies") {
		i.ComponentRequestBodies = append(i.ComponentRequestBodies, &IndexNode[*ReferencedRequestBody]{
			Node:     rb,
			Location: loc,
		})
		return
	}

	i.InlineRequestBodies = append(i.InlineRequestBodies, &IndexNode[*ReferencedRequestBody]{
		Node:     rb,
		Location: loc,
	})
}

func (i *Index) indexReferencedHeader(ctx context.Context, loc Locations, header *ReferencedHeader) {
	if header == nil {
		return
	}

	if header.IsReference() && !header.IsResolved() {
		resolveAndValidateReference(i, ctx, header)
	}

	if header.IsReference() {
		i.HeaderReferences = append(i.HeaderReferences, &IndexNode[*ReferencedHeader]{
			Node:     header,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "headers") {
		i.ComponentHeaders = append(i.ComponentHeaders, &IndexNode[*ReferencedHeader]{
			Node:     header,
			Location: loc,
		})
		return
	}

	i.InlineHeaders = append(i.InlineHeaders, &IndexNode[*ReferencedHeader]{
		Node:     header,
		Location: loc,
	})
}

func (i *Index) indexReferencedExample(ctx context.Context, loc Locations, example *ReferencedExample) {
	if example == nil {
		return
	}

	if example.IsReference() && !example.IsResolved() {
		resolveAndValidateReference(i, ctx, example)
	}

	if example.IsReference() {
		i.ExampleReferences = append(i.ExampleReferences, &IndexNode[*ReferencedExample]{
			Node:     example,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "examples") {
		i.ComponentExamples = append(i.ComponentExamples, &IndexNode[*ReferencedExample]{
			Node:     example,
			Location: loc,
		})
		return
	}

	i.InlineExamples = append(i.InlineExamples, &IndexNode[*ReferencedExample]{
		Node:     example,
		Location: loc,
	})
}

func (i *Index) indexReferencedLink(ctx context.Context, loc Locations, link *ReferencedLink) {
	if link == nil {
		return
	}

	if link.IsReference() && !link.IsResolved() {
		resolveAndValidateReference(i, ctx, link)
	}

	if link.IsReference() {
		i.LinkReferences = append(i.LinkReferences, &IndexNode[*ReferencedLink]{
			Node:     link,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "links") {
		i.ComponentLinks = append(i.ComponentLinks, &IndexNode[*ReferencedLink]{
			Node:     link,
			Location: loc,
		})
		return
	}

	i.InlineLinks = append(i.InlineLinks, &IndexNode[*ReferencedLink]{
		Node:     link,
		Location: loc,
	})
}

func (i *Index) indexReferencedCallback(ctx context.Context, loc Locations, callback *ReferencedCallback) {
	if callback == nil {
		return
	}

	if callback.IsReference() && !callback.IsResolved() {
		resolveAndValidateReference(i, ctx, callback)
	}

	if callback.IsReference() {
		i.CallbackReferences = append(i.CallbackReferences, &IndexNode[*ReferencedCallback]{
			Node:     callback,
			Location: loc,
		})
		return
	}

	if isTopLevelComponent(loc, "callbacks") {
		i.ComponentCallbacks = append(i.ComponentCallbacks, &IndexNode[*ReferencedCallback]{
			Node:     callback,
			Location: loc,
		})
		return
	}

	i.InlineCallbacks = append(i.InlineCallbacks, &IndexNode[*ReferencedCallback]{
		Node:     callback,
		Location: loc,
	})
}

func (i *Index) indexReferencedSecurityScheme(ctx context.Context, loc Locations, ss *ReferencedSecurityScheme) {
	if ss == nil {
		return
	}

	if ss.IsReference() && !ss.IsResolved() {
		resolveAndValidateReference(i, ctx, ss)
	}

	if ss.IsReference() {
		i.SecuritySchemeReferences = append(i.SecuritySchemeReferences, &IndexNode[*ReferencedSecurityScheme]{
			Node:     ss,
			Location: loc,
		})
		return
	}

	// SecuritySchemes are always components (no inline security schemes)
	i.ComponentSecuritySchemes = append(i.ComponentSecuritySchemes, &IndexNode[*ReferencedSecurityScheme]{
		Node:     ss,
		Location: loc,
	})
}

func (i *Index) indexSecurityRequirement(_ context.Context, loc Locations, req *SecurityRequirement) {
	if req == nil {
		return
	}

	i.SecurityRequirements = append(i.SecurityRequirements, &IndexNode[*SecurityRequirement]{
		Node:     req,
		Location: loc,
	})
}

func (i *Index) indexDiscriminator(_ context.Context, loc Locations, discriminator *oas3.Discriminator) {
	if discriminator == nil {
		return
	}
	i.Discriminators = append(i.Discriminators, &IndexNode[*oas3.Discriminator]{
		Node:     discriminator,
		Location: loc,
	})
}

func (i *Index) indexXML(_ context.Context, loc Locations, xml *oas3.XML) {
	if xml == nil {
		return
	}
	i.XMLs = append(i.XMLs, &IndexNode[*oas3.XML]{
		Node:     xml,
		Location: loc,
	})
}

func (i *Index) indexMediaType(_ context.Context, loc Locations, mediaType *MediaType) {
	if mediaType == nil {
		return
	}
	i.MediaTypes = append(i.MediaTypes, &IndexNode[*MediaType]{
		Node:     mediaType,
		Location: loc,
	})
}

func (i *Index) indexEncoding(_ context.Context, loc Locations, encoding *Encoding) {
	if encoding == nil {
		return
	}
	i.Encodings = append(i.Encodings, &IndexNode[*Encoding]{
		Node:     encoding,
		Location: loc,
	})
}

func (i *Index) indexOAuthFlows(_ context.Context, loc Locations, flows *OAuthFlows) {
	if flows == nil {
		return
	}
	i.OAuthFlows = append(i.OAuthFlows, &IndexNode[*OAuthFlows]{
		Node:     flows,
		Location: loc,
	})
}

func (i *Index) indexOAuthFlow(_ context.Context, loc Locations, flow *OAuthFlow) {
	if flow == nil {
		return
	}
	i.OAuthFlowItems = append(i.OAuthFlowItems, &IndexNode[*OAuthFlow]{
		Node:     flow,
		Location: loc,
	})
}

func (i *Index) indexDescriptionNode(_ context.Context, loc Locations, d Descriptioner) {
	if d == nil {
		return
	}
	i.DescriptionNodes = append(i.DescriptionNodes, &IndexNode[Descriptioner]{
		Node:     d,
		Location: loc,
	})
}

func (i *Index) indexSummaryNode(_ context.Context, loc Locations, s Summarizer) {
	if s == nil {
		return
	}
	i.SummaryNodes = append(i.SummaryNodes, &IndexNode[Summarizer]{
		Node:     s,
		Location: loc,
	})
}

func (i *Index) indexDescriptionAndSummaryNode(_ context.Context, loc Locations, ds DescriptionAndSummary) {
	if ds == nil {
		return
	}
	i.DescriptionAndSummaryNodes = append(i.DescriptionAndSummaryNodes, &IndexNode[DescriptionAndSummary]{
		Node:     ds,
		Location: loc,
	})
}

func (i *Index) documentPathForSchema(schema *oas3.JSONSchemaReferenceable) string {
	if i == nil || schema == nil {
		return ""
	}

	if info := schema.GetReferenceResolutionInfo(); info != nil {
		if info.AbsoluteDocumentPath != i.resolveOpts.TargetLocation {
			return info.AbsoluteDocumentPath
		}
		if len(i.currentDocumentStack) > 0 {
			current := i.currentDocumentStack[len(i.currentDocumentStack)-1]
			if current != i.resolveOpts.TargetLocation {
				return current
			}
		}
		return ""
	}

	if len(i.currentDocumentStack) > 0 {
		current := i.currentDocumentStack[len(i.currentDocumentStack)-1]
		if current != i.resolveOpts.TargetLocation {
			return current
		}
		return ""
	}

	return ""
}

func (i *Index) applyDocumentLocation(errs []error, documentPath string) []error {
	if len(errs) == 0 || documentPath == "" {
		return errs
	}

	updated := make([]error, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		var vErr *validation.Error
		if errors.As(err, &vErr) && vErr != nil {
			if vErr.DocumentLocation == "" {
				vErr.DocumentLocation = documentPath
			}
			updated = append(updated, vErr)
			continue
		}
		updated = append(updated, err)
	}

	return updated
}

func (i *Index) referenceValidationOptions() []validation.Option {
	if i == nil || i.Doc == nil {
		return nil
	}

	return []validation.Option{
		validation.WithContextObject(i.Doc),
		validation.WithContextObject(&oas3.ParentDocumentVersion{OpenAPI: pointer.From(i.Doc.OpenAPI)}),
	}
}

func documentPathForReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](i *Index, ref *Reference[T, V, C]) string {
	if i == nil || ref == nil {
		return ""
	}

	if info := ref.GetReferenceResolutionInfo(); info != nil {
		if info.AbsoluteDocumentPath != i.resolveOpts.TargetLocation {
			return info.AbsoluteDocumentPath
		}
		return ""
	}

	return ""
}

func resolveAndValidateReference[T any, V interfaces.Validator[T], C marshaller.CoreModeler](i *Index, ctx context.Context, ref *Reference[T, V, C]) {
	if i == nil || ref == nil {
		return
	}

	if _, err := ref.Resolve(ctx, i.resolveOpts); err != nil {
		i.resolutionErrs = append(i.resolutionErrs, validation.NewValidationErrorWithDocumentLocation(
			validation.SeverityError,
			"resolution-openapi-reference",
			err,
			nil,
			documentPathForReference(i, ref),
		))
		return
	}

	obj := ref.GetObject()
	if obj == nil || i.Doc == nil {
		return
	}

	var validator V
	if v, ok := any(obj).(V); ok {
		validator = v
		validationErrs := validator.Validate(ctx, i.referenceValidationOptions()...)
		i.validationErrs = append(i.validationErrs, i.applyDocumentLocation(validationErrs, documentPathForReference(i, ref))...)
	}
}

// isTopLevelComponent checks if the location represents a top-level component definition.
// A top-level component has the path: /components/{componentType}/{name}
func isTopLevelComponent(loc Locations, componentType string) bool {
	// Location should be exactly: /components/{componentType}/{name}
	// Length 2: [components context, {componentType}/{name} context]
	if len(loc) != 2 {
		return false
	}

	// First element: ParentField = "components"
	if loc[0].ParentField != "components" {
		return false
	}

	// Second element: ParentField = componentType, ParentKey = name
	if loc[1].ParentField != componentType || loc[1].ParentKey == nil {
		return false
	}

	return true
}

// getParentSchema extracts the parent schema from a LocationContext using the ParentMatchFunc.
func getParentSchema(loc LocationContext) *oas3.Schema {
	var parentSchema *oas3.Schema

	// Use the ParentMatchFunc to capture the parent node
	_ = loc.ParentMatchFunc(Matcher{
		Schema: func(s *oas3.JSONSchemaReferenceable) error {
			if s == nil {
				return nil
			}
			if !s.IsBool() && !s.IsReference() {
				parentSchema = s.GetSchema()
			} else if s.IsReference() {
				// For references, get the resolved schema
				if resolved := s.GetResolvedSchema(); resolved != nil && !resolved.IsBool() {
					parentSchema = resolved.GetSchema()
				}
			}
			return nil
		},
	})

	return parentSchema
}

// buildPathSegment creates a CircularPathSegment with constraint info from the parent schema.
func buildPathSegment(loc LocationContext) CircularPathSegment {
	segment := CircularPathSegment{
		Field: loc.ParentField,
	}

	if loc.ParentKey != nil {
		segment.PropertyName = *loc.ParentKey
	}
	if loc.ParentIndex != nil {
		segment.BranchIndex = *loc.ParentIndex
	}

	parent := getParentSchema(loc)
	if parent == nil {
		return segment
	}

	// Check if parent schema is nullable (termination point)
	segment.IsNullable = isNullable(parent)

	// Extract constraints based on field type
	switch loc.ParentField {
	case "properties":
		if loc.ParentKey != nil {
			// Check if property is required
			for _, req := range parent.GetRequired() {
				if req == *loc.ParentKey {
					segment.IsRequired = true
					break
				}
			}
		}
	case "items":
		segment.ArrayMinItems = parent.GetMinItems() // Returns 0 if nil (default)
	case "additionalProperties":
		if minProps := parent.GetMinProperties(); minProps != nil {
			segment.MinProperties = *minProps
		}
		// Default is 0 (empty object allowed)
	}

	return segment
}

// isNullable checks if a schema allows null values (termination point for circular refs).
func isNullable(schema *oas3.Schema) bool {
	if schema == nil {
		return false
	}

	// OAS 3.0 style: nullable: true
	if schema.GetNullable() {
		return true
	}

	// OAS 3.1 style: type includes "null"
	types := schema.GetType()
	for _, t := range types {
		if t == oas3.SchemaTypeNull {
			return true
		}
	}

	return false
}

// classifyCircularPath determines if the path allows termination.
// Returns (classification, polymorphicInfo) where polymorphicInfo is set if pending.
func (i *Index) classifyCircularPath(schema *oas3.JSONSchemaReferenceable, segments []CircularPathSegment, loc Locations) (CircularClassification, *PolymorphicCircularRef) {
	// Check if any segment allows termination
	for segIdx, segment := range segments {
		// Check nullable at any point in the path
		if segment.IsNullable {
			return CircularValid, nil
		}

		switch segment.Field {
		case "properties":
			// Optional property = valid termination
			if !segment.IsRequired {
				return CircularValid, nil
			}

		case "items":
			// Empty array terminates if minItems == 0 (default)
			if segment.ArrayMinItems == 0 {
				return CircularValid, nil
			}

		case "additionalProperties":
			// Empty object terminates if minProperties == 0 (default)
			if segment.MinProperties == 0 {
				return CircularValid, nil
			}

		case "oneOf", "anyOf":
			// Mark for post-processing - need to check ALL branches
			// Create polymorphic tracking info
			parentLocLen := len(loc) - len(segments) + segIdx
			if parentLocLen < 0 {
				parentLocLen = 0
			}
			parentLoc := copyLocations(loc[:parentLocLen])

			polymorphicInfo := &PolymorphicCircularRef{
				ParentSchema:   schema,
				ParentLocation: parentLoc,
				Field:          segment.Field,
				BranchResults:  make(map[int]CircularClassification),
				TotalBranches:  countPolymorphicBranches(schema, segment.Field),
			}
			// Record this branch as potentially invalid (recurses)
			polymorphicInfo.BranchResults[segment.BranchIndex] = CircularInvalid
			return CircularPending, polymorphicInfo

		case "allOf":
			// For allOf, if ANY branch has invalid circular ref, the whole thing is invalid
			// because ALL branches must be satisfied
			// Check if rest of path allows termination
			remaining := segments[segIdx+1:]
			if !pathAllowsTermination(remaining) {
				return CircularInvalid, nil
			}
		}
	}

	// No termination point found in non-polymorphic path
	return CircularInvalid, nil
}

// countPolymorphicBranches counts the number of branches in a oneOf/anyOf schema.
func countPolymorphicBranches(schema *oas3.JSONSchemaReferenceable, field string) int {
	if schema == nil || schema.IsBool() {
		return 0
	}

	innerSchema := schema.GetSchema()
	if innerSchema == nil {
		return 0
	}

	switch field {
	case "oneOf":
		if oneOf := innerSchema.GetOneOf(); oneOf != nil {
			return len(oneOf)
		}
	case "anyOf":
		if anyOf := innerSchema.GetAnyOf(); anyOf != nil {
			return len(anyOf)
		}
	case "allOf":
		if allOf := innerSchema.GetAllOf(); allOf != nil {
			return len(allOf)
		}
	}

	return 0
}

// pathAllowsTermination checks if any segment in the remaining path allows termination.
func pathAllowsTermination(segments []CircularPathSegment) bool {
	for _, seg := range segments {
		if seg.IsNullable {
			return true
		}

		switch seg.Field {
		case "properties":
			if !seg.IsRequired {
				return true
			}
		case "items":
			if seg.ArrayMinItems == 0 {
				return true
			}
		case "additionalProperties":
			if seg.MinProperties == 0 {
				return true
			}
		case "oneOf", "anyOf":
			// Assume polymorphic branches might provide termination
			return true
		}
	}
	return false
}

func joinReferenceChainWithArrows(chain []string) string {
	if len(chain) == 0 {
		return ""
	}
	if len(chain) == 1 {
		return chain[0]
	}

	var result strings.Builder
	result.WriteString(chain[0])
	for i := 1; i < len(chain); i++ {
		result.WriteString(" -> ")
		result.WriteString(chain[i])
	}
	return result.String()
}

// recordPolymorphicBranch records a polymorphic branch for post-processing.
func (i *Index) recordPolymorphicBranch(info *PolymorphicCircularRef) {
	if info == nil {
		return
	}
	i.polymorphicRefs = append(i.polymorphicRefs, info)
}

// finalizePolymorphicCirculars is called after all walking completes.
// It analyzes polymorphic schemas to determine if ALL branches recurse.
func (i *Index) finalizePolymorphicCirculars() {
	// Group by parent schema
	grouped := make(map[*oas3.JSONSchemaReferenceable]*PolymorphicCircularRef)

	for _, ref := range i.polymorphicRefs {
		existing, found := grouped[ref.ParentSchema]
		if found {
			// Merge branch results
			for idx, classification := range ref.BranchResults {
				existing.BranchResults[idx] = classification
			}
		} else {
			grouped[ref.ParentSchema] = ref
		}
	}

	// Analyze each polymorphic schema
	for _, ref := range grouped {
		switch ref.Field {
		case "oneOf", "anyOf":
			// Invalid only if ALL branches have invalid circular refs
			allInvalid := true
			for branchIdx := 0; branchIdx < ref.TotalBranches; branchIdx++ {
				classification, found := ref.BranchResults[branchIdx]
				if !found || classification != CircularInvalid {
					// This branch either doesn't recurse or has valid termination
					allInvalid = false
					break
				}
			}

			if allInvalid && ref.TotalBranches > 0 {
				i.circularErrs = append(i.circularErrs, validation.NewValidationErrorWithDocumentLocation(
					validation.SeverityError,
					"circular-reference-invalid",
					fmt.Errorf("non-terminating circular reference: all %s branches recurse with no base case", ref.Field),
					getSchemaErrorNode(ref.ParentSchema),
					i.documentPathForSchema(ref.ParentSchema),
				))
			}

		case "allOf":
			// Invalid if ANY branch has invalid circular ref (already handled inline in classifyCircularPath)
			// This case is here for completeness if we need cross-branch tracking
		}
	}
}

// copyLocations creates a copy of the Locations slice.
func copyLocations(loc Locations) Locations {
	if loc == nil {
		return nil
	}
	result := make(Locations, len(loc))
	copy(result, loc)
	return result
}

// getRefTarget extracts the absolute $ref target string from a schema reference.
// Uses the resolved AbsoluteReference from resolution cache for normalization.
func getRefTarget(schema *oas3.JSONSchemaReferenceable) string {
	if schema == nil || !schema.IsReference() {
		return ""
	}

	if !schema.IsResolved() {
		panic("getRefTarget called on unresolved schema reference")
	}

	info := schema.GetReferenceResolutionInfo()
	if info == nil {
		return ""
	}

	return info.AbsoluteReference.String()
}

// getSchemaErrorNode returns an appropriate YAML node for error reporting.
func getSchemaErrorNode(schema *oas3.JSONSchemaReferenceable) *yaml.Node {
	if schema == nil {
		return nil
	}
	if schema.IsBool() {
		return nil
	}
	innerSchema := schema.GetSchema()
	if innerSchema == nil {
		return nil
	}
	// Try to get the $ref node if it's a reference
	if core := innerSchema.GetCore(); core != nil && core.Ref.Present {
		return core.Ref.GetKeyNodeOrRoot(innerSchema.GetRootNode())
	}
	return innerSchema.GetRootNode()
}
