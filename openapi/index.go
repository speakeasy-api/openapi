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
	Field         string                        // e.g., "properties", "items", "allOf", "oneOf", "anyOf", "additionalProperties"
	PropertyName  string                        // Set if Field == "properties"
	IsRequired    bool                          // Set if this property is in parent's Required array
	ArrayMinItems int64                         // Parent's MinItems value (0 means empty array terminates)
	MinProperties int64                         // Parent's MinProperties value (0 means empty object terminates)
	BranchIndex   int                           // Index in oneOf/anyOf/allOf array
	IsNullable    bool                          // True if this schema allows null (termination point)
	ParentSchema  *oas3.JSONSchemaReferenceable // The parent schema (for polymorphic cases)
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
	ExternalParameters  []*IndexNode[*ReferencedParameter] // Top-level Parameters in external documents
	ParameterReferences []*IndexNode[*ReferencedParameter] // All Parameter $ref pointers

	Responses []*IndexNode[*Responses] // All Responses containers (operation.responses)

	InlineResponses    []*IndexNode[*ReferencedResponse] // Responses defined inline in operations
	ComponentResponses []*IndexNode[*ReferencedResponse] // Responses in /components/responses/
	ExternalResponses  []*IndexNode[*ReferencedResponse] // Top-level Responses in external documents
	ResponseReferences []*IndexNode[*ReferencedResponse] // All Response $ref pointers

	InlineRequestBodies    []*IndexNode[*ReferencedRequestBody] // RequestBodies defined inline in operations
	ComponentRequestBodies []*IndexNode[*ReferencedRequestBody] // RequestBodies in /components/requestBodies/
	ExternalRequestBodies  []*IndexNode[*ReferencedRequestBody] // Top-level RequestBodies in external documents
	RequestBodyReferences  []*IndexNode[*ReferencedRequestBody] // All RequestBody $ref pointers

	InlineHeaders    []*IndexNode[*ReferencedHeader] // Headers defined inline
	ComponentHeaders []*IndexNode[*ReferencedHeader] // Headers in /components/headers/
	ExternalHeaders  []*IndexNode[*ReferencedHeader] // Top-level Headers in external documents
	HeaderReferences []*IndexNode[*ReferencedHeader] // All Header $ref pointers

	InlineExamples    []*IndexNode[*ReferencedExample] // Examples defined inline
	ComponentExamples []*IndexNode[*ReferencedExample] // Examples in /components/examples/
	ExternalExamples  []*IndexNode[*ReferencedExample] // Top-level Examples in external documents
	ExampleReferences []*IndexNode[*ReferencedExample] // All Example $ref pointers

	InlineLinks    []*IndexNode[*ReferencedLink] // Links defined inline in responses
	ComponentLinks []*IndexNode[*ReferencedLink] // Links in /components/links/
	ExternalLinks  []*IndexNode[*ReferencedLink] // Top-level Links in external documents
	LinkReferences []*IndexNode[*ReferencedLink] // All Link $ref pointers

	InlineCallbacks    []*IndexNode[*ReferencedCallback] // Callbacks defined inline in operations
	ComponentCallbacks []*IndexNode[*ReferencedCallback] // Callbacks in /components/callbacks/
	ExternalCallbacks  []*IndexNode[*ReferencedCallback] // Top-level Callbacks in external documents
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

	DescriptionNodes           []*IndexNode[Descriptioner]         // All nodes that have a Description field
	SummaryNodes               []*IndexNode[Summarizer]            // All nodes that have a Summary field
	DescriptionAndSummaryNodes []*IndexNode[DescriptionAndSummary] // All nodes that have both Description and Summary fields

	validationErrs []error
	resolutionErrs []error
	circularErrs   []error

	validCircularRefs   int // Count of valid (terminating) circular references
	invalidCircularRefs int // Count of invalid (non-terminating) circular references

	resolveOpts references.ResolveOptions

	// Circular reference tracking (internal)
	indexedSchemas       map[*oas3.JSONSchemaReferenceable]bool // Tracks which schemas have been fully indexed
	indexedParameters    map[*Parameter]bool                    // Tracks which parameters have been fully indexed
	indexedResponses     map[*Response]bool                     // Tracks which responses have been fully indexed
	indexedRequestBodies map[*RequestBody]bool                  // Tracks which request bodies have been fully indexed
	indexedHeaders       map[*Header]bool                       // Tracks which headers have been fully indexed
	indexedExamples      map[*Example]bool                      // Tracks which examples have been fully indexed
	indexedLinks         map[*Link]bool                         // Tracks which links have been fully indexed
	indexedCallbacks     map[*Callback]bool                     // Tracks which callbacks have been fully indexed
	indexedPathItems     map[*PathItem]bool                     // Tracks which path items have been fully indexed
	referenceStack       []referenceStackEntry                  // Active reference resolution chain (by ref target)
	polymorphicRefs      []*PolymorphicCircularRef              // Pending polymorphic circulars
	visitedRefs          map[string]bool                        // Tracks visited ref targets to avoid duplicates
	indexedReferences    map[any]bool                           // Tracks indexed reference objects to ensure each $ref appears once
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
		indexedParameters:    make(map[*Parameter]bool),
		indexedResponses:     make(map[*Response]bool),
		indexedRequestBodies: make(map[*RequestBody]bool),
		indexedHeaders:       make(map[*Header]bool),
		indexedExamples:      make(map[*Example]bool),
		indexedLinks:         make(map[*Link]bool),
		indexedCallbacks:     make(map[*Callback]bool),
		indexedPathItems:     make(map[*PathItem]bool),
		referenceStack:       make([]referenceStackEntry, 0),
		polymorphicRefs:      make([]*PolymorphicCircularRef, 0),
		visitedRefs:          make(map[string]bool),
		indexedReferences:    make(map[any]bool),
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
		len(i.ExternalSchemas),
	)
	allSchemas = append(allSchemas, i.BooleanSchemas...)
	allSchemas = append(allSchemas, i.InlineSchemas...)
	allSchemas = append(allSchemas, i.ComponentSchemas...)
	allSchemas = append(allSchemas, i.ExternalSchemas...)
	return allSchemas
}

// GetAllPathItems returns all path items in the index (inline, component, and external).
func (i *Index) GetAllPathItems() []*IndexNode[*ReferencedPathItem] {
	if i == nil {
		return nil
	}

	allPathItems := make([]*IndexNode[*ReferencedPathItem], 0, len(i.InlinePathItems)+
		len(i.ComponentPathItems)+
		len(i.ExternalPathItems),
	)
	allPathItems = append(allPathItems, i.InlinePathItems...)
	allPathItems = append(allPathItems, i.ComponentPathItems...)
	allPathItems = append(allPathItems, i.ExternalPathItems...)
	return allPathItems
}

// GetAllParameters returns all parameters in the index (inline, component, and external).
func (i *Index) GetAllParameters() []*IndexNode[*ReferencedParameter] {
	if i == nil {
		return nil
	}

	allParameters := make([]*IndexNode[*ReferencedParameter], 0, len(i.InlineParameters)+
		len(i.ComponentParameters)+
		len(i.ExternalParameters),
	)
	allParameters = append(allParameters, i.InlineParameters...)
	allParameters = append(allParameters, i.ComponentParameters...)
	allParameters = append(allParameters, i.ExternalParameters...)
	return allParameters
}

// GetAllResponses returns all responses in the index (inline, component, and external).
func (i *Index) GetAllResponses() []*IndexNode[*ReferencedResponse] {
	if i == nil {
		return nil
	}

	allResponses := make([]*IndexNode[*ReferencedResponse], 0, len(i.InlineResponses)+
		len(i.ComponentResponses)+
		len(i.ExternalResponses),
	)
	allResponses = append(allResponses, i.InlineResponses...)
	allResponses = append(allResponses, i.ComponentResponses...)
	allResponses = append(allResponses, i.ExternalResponses...)
	return allResponses
}

// GetAllRequestBodies returns all request bodies in the index (inline, component, and external).
func (i *Index) GetAllRequestBodies() []*IndexNode[*ReferencedRequestBody] {
	if i == nil {
		return nil
	}

	allRequestBodies := make([]*IndexNode[*ReferencedRequestBody], 0, len(i.InlineRequestBodies)+
		len(i.ComponentRequestBodies)+
		len(i.ExternalRequestBodies),
	)
	allRequestBodies = append(allRequestBodies, i.InlineRequestBodies...)
	allRequestBodies = append(allRequestBodies, i.ComponentRequestBodies...)
	allRequestBodies = append(allRequestBodies, i.ExternalRequestBodies...)
	return allRequestBodies
}

// GetAllHeaders returns all headers in the index (inline, component, and external).
func (i *Index) GetAllHeaders() []*IndexNode[*ReferencedHeader] {
	if i == nil {
		return nil
	}

	allHeaders := make([]*IndexNode[*ReferencedHeader], 0, len(i.InlineHeaders)+
		len(i.ComponentHeaders)+
		len(i.ExternalHeaders),
	)
	allHeaders = append(allHeaders, i.InlineHeaders...)
	allHeaders = append(allHeaders, i.ComponentHeaders...)
	allHeaders = append(allHeaders, i.ExternalHeaders...)
	return allHeaders
}

// GetAllExamples returns all examples in the index (inline, component, and external).
func (i *Index) GetAllExamples() []*IndexNode[*ReferencedExample] {
	if i == nil {
		return nil
	}

	allExamples := make([]*IndexNode[*ReferencedExample], 0, len(i.InlineExamples)+
		len(i.ComponentExamples)+
		len(i.ExternalExamples),
	)
	allExamples = append(allExamples, i.InlineExamples...)
	allExamples = append(allExamples, i.ComponentExamples...)
	allExamples = append(allExamples, i.ExternalExamples...)
	return allExamples
}

// GetAllLinks returns all links in the index (inline, component, and external).
func (i *Index) GetAllLinks() []*IndexNode[*ReferencedLink] {
	if i == nil {
		return nil
	}

	allLinks := make([]*IndexNode[*ReferencedLink], 0, len(i.InlineLinks)+
		len(i.ComponentLinks)+
		len(i.ExternalLinks),
	)
	allLinks = append(allLinks, i.InlineLinks...)
	allLinks = append(allLinks, i.ComponentLinks...)
	allLinks = append(allLinks, i.ExternalLinks...)
	return allLinks
}

// GetAllCallbacks returns all callbacks in the index (inline, component, and external).
func (i *Index) GetAllCallbacks() []*IndexNode[*ReferencedCallback] {
	if i == nil {
		return nil
	}

	allCallbacks := make([]*IndexNode[*ReferencedCallback], 0, len(i.InlineCallbacks)+
		len(i.ComponentCallbacks)+
		len(i.ExternalCallbacks),
	)
	allCallbacks = append(allCallbacks, i.InlineCallbacks...)
	allCallbacks = append(allCallbacks, i.ComponentCallbacks...)
	allCallbacks = append(allCallbacks, i.ExternalCallbacks...)
	return allCallbacks
}

// ReferenceNode represents any node that can be a reference in an OpenAPI document.
// This interface is satisfied by both Reference[T, V, C] types (PathItem, Parameter, Response, etc.)
// and JSONSchemaReferenceable.
type ReferenceNode interface {
	GetReference() references.Reference
	IsReference() bool
	GetRootNode() *yaml.Node
}

// GetAllReferences returns all reference nodes in the index across all reference types.
// This includes SchemaReferences, PathItemReferences, ParameterReferences, ResponseReferences,
// RequestBodyReferences, HeaderReferences, ExampleReferences, LinkReferences, CallbackReferences,
// and SecuritySchemeReferences.
func (i *Index) GetAllReferences() []*IndexNode[ReferenceNode] {
	if i == nil {
		return nil
	}

	totalCount := len(i.SchemaReferences) +
		len(i.PathItemReferences) +
		len(i.ParameterReferences) +
		len(i.ResponseReferences) +
		len(i.RequestBodyReferences) +
		len(i.HeaderReferences) +
		len(i.ExampleReferences) +
		len(i.LinkReferences) +
		len(i.CallbackReferences) +
		len(i.SecuritySchemeReferences)

	allReferences := make([]*IndexNode[ReferenceNode], 0, totalCount)

	// Add schema references
	for _, ref := range i.SchemaReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add path item references
	for _, ref := range i.PathItemReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add parameter references
	for _, ref := range i.ParameterReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add response references
	for _, ref := range i.ResponseReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add request body references
	for _, ref := range i.RequestBodyReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add header references
	for _, ref := range i.HeaderReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add example references
	for _, ref := range i.ExampleReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add link references
	for _, ref := range i.LinkReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add callback references
	for _, ref := range i.CallbackReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	// Add security scheme references
	for _, ref := range i.SecuritySchemeReferences {
		allReferences = append(allReferences, &IndexNode[ReferenceNode]{
			Node:     ref.Node,
			Location: ref.Location,
		})
	}

	return allReferences
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

// GetValidCircularRefCount returns the count of valid (terminating) circular references found during indexing.
func (i *Index) GetValidCircularRefCount() int {
	if i == nil {
		return 0
	}
	return i.validCircularRefs
}

// GetInvalidCircularRefCount returns the count of invalid (non-terminating) circular references found during indexing.
func (i *Index) GetInvalidCircularRefCount() int {
	if i == nil {
		return 0
	}
	return i.invalidCircularRefs
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
				// Check for unknown properties on any model with a core
				if coreAccessor, ok := a.(interface{ GetCoreAny() any }); ok {
					if core := coreAccessor.GetCoreAny(); core != nil {
						if coreModeler, ok := core.(marshaller.CoreModeler); ok {
							index.checkUnknownProperties(ctx, coreModeler)
						}
					}
				}

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
		// Add to references list only if this exact schema object hasn't been indexed yet
		// This ensures each $ref in the source document is indexed exactly once
		if !i.indexedSchemas[schema] {
			i.SchemaReferences = append(i.SchemaReferences, &IndexNode[*oas3.JSONSchemaReferenceable]{
				Node:     schema,
				Location: loc,
			})
			i.indexedSchemas[schema] = true
		}

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

				switch classification {
				case CircularInvalid:
					i.invalidCircularRefs++
					err := fmt.Errorf("non-terminating circular reference detected: %s", joinReferenceChainWithArrows(circularChain))
					i.circularErrs = append(i.circularErrs, validation.NewValidationErrorWithDocumentLocation(
						validation.SeverityError,
						"circular-reference-invalid",
						err,
						getSchemaErrorNode(schema),
						externalDocumentPath,
					))
				case CircularPending:
					if polymorphicInfo != nil {
						i.recordPolymorphicBranch(polymorphicInfo)
					}
				case CircularValid:
					i.validCircularRefs++
				case CircularUnclassified:
					// No action needed for unclassified circulars
				}

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
		if !i.isFromMainDocument() && !i.indexedSchemas[schema] {
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
func (i *Index) isFromMainDocument() bool {
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

// checkUnknownProperties checks for unknown properties in a core model and adds warnings.
func (i *Index) checkUnknownProperties(_ context.Context, core marshaller.CoreModeler) {
	if core == nil {
		return
	}

	unknownProps := core.GetUnknownProperties()
	if len(unknownProps) == 0 {
		return
	}

	docPath := ""
	if len(i.currentDocumentStack) > 0 {
		currentDoc := i.currentDocumentStack[len(i.currentDocumentStack)-1]
		if currentDoc != i.resolveOpts.TargetLocation {
			docPath = currentDoc
		}
	}

	for _, prop := range unknownProps {
		err := fmt.Errorf("unknown property '%s' found", prop)
		i.validationErrs = append(i.validationErrs, validation.NewValidationErrorWithDocumentLocation(
			validation.SeverityWarning,
			"validation-unknown-properties",
			err,
			core.GetRootNode(),
			docPath,
		))
	}
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
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[pathItem] {
			i.PathItemReferences = append(i.PathItemReferences, &IndexNode[*ReferencedPathItem]{
				Node:     pathItem,
				Location: loc,
			})
			i.indexedReferences[pathItem] = true
		}

		// Get the document path for the resolved path item
		info := pathItem.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content (similar to how schemas are handled)
		resolved := pathItem.GetObject()
		if resolved != nil {
			// Wrap the resolved PathItem back into a ReferencedPathItem for walking
			wrapped := &ReferencedPathItem{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	if obj == nil {
		return
	}

	// Check if this is a component path item
	if isTopLevelComponent(loc, "pathItems") {
		if !i.indexedPathItems[obj] {
			i.ComponentPathItems = append(i.ComponentPathItems, &IndexNode[*ReferencedPathItem]{
				Node:     pathItem,
				Location: loc,
			})
			i.indexedPathItems[obj] = true
		}
		return
	}

	// Check if this is a top-level path item in an external document
	// External path items appear at location "/" (root of external doc)
	if isTopLevelExternalSchema(loc) {
		if !i.indexedPathItems[obj] {
			i.ExternalPathItems = append(i.ExternalPathItems, &IndexNode[*ReferencedPathItem]{
				Node:     pathItem,
				Location: loc,
			})
			i.indexedPathItems[obj] = true
		}
		return
	}

	// Everything else is an inline path item
	if !i.indexedPathItems[obj] {
		i.InlinePathItems = append(i.InlinePathItems, &IndexNode[*ReferencedPathItem]{
			Node:     pathItem,
			Location: loc,
		})
		i.indexedPathItems[obj] = true
	}
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
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[param] {
			i.ParameterReferences = append(i.ParameterReferences, &IndexNode[*ReferencedParameter]{
				Node:     param,
				Location: loc,
			})
			i.indexedReferences[param] = true
		}

		// Get the document path for the resolved parameter
		info := param.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := param.GetObject()
		if resolved != nil {
			wrapped := &ReferencedParameter{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := param.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "parameters") {
		if !i.indexedParameters[obj] {
			i.ComponentParameters = append(i.ComponentParameters, &IndexNode[*ReferencedParameter]{
				Node:     param,
				Location: loc,
			})
			i.indexedParameters[obj] = true
		}
		return
	}

	// Check if this is a top-level parameter in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedParameters[obj] {
			i.ExternalParameters = append(i.ExternalParameters, &IndexNode[*ReferencedParameter]{
				Node:     param,
				Location: loc,
			})
			i.indexedParameters[obj] = true
		}
		return
	}

	// Everything else is an inline parameter
	if !i.indexedParameters[obj] {
		i.InlineParameters = append(i.InlineParameters, &IndexNode[*ReferencedParameter]{
			Node:     param,
			Location: loc,
		})
		i.indexedParameters[obj] = true
	}
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
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[resp] {
			i.ResponseReferences = append(i.ResponseReferences, &IndexNode[*ReferencedResponse]{
				Node:     resp,
				Location: loc,
			})
			i.indexedReferences[resp] = true
		}

		// Get the document path for the resolved response
		info := resp.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := resp.GetObject()
		if resolved != nil {
			wrapped := &ReferencedResponse{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := resp.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "responses") {
		if !i.indexedResponses[obj] {
			i.ComponentResponses = append(i.ComponentResponses, &IndexNode[*ReferencedResponse]{
				Node:     resp,
				Location: loc,
			})
			i.indexedResponses[obj] = true
		}
		return
	}

	// Check if this is a top-level response in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedResponses[obj] {
			i.ExternalResponses = append(i.ExternalResponses, &IndexNode[*ReferencedResponse]{
				Node:     resp,
				Location: loc,
			})
			i.indexedResponses[obj] = true
		}
		return
	}

	// Everything else is an inline response
	if !i.indexedResponses[obj] {
		i.InlineResponses = append(i.InlineResponses, &IndexNode[*ReferencedResponse]{
			Node:     resp,
			Location: loc,
		})
		i.indexedResponses[obj] = true
	}
}

func (i *Index) indexReferencedRequestBody(ctx context.Context, loc Locations, rb *ReferencedRequestBody) {
	if rb == nil {
		return
	}

	if rb.IsReference() && !rb.IsResolved() {
		resolveAndValidateReference(i, ctx, rb)
	}

	if rb.IsReference() {
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[rb] {
			i.RequestBodyReferences = append(i.RequestBodyReferences, &IndexNode[*ReferencedRequestBody]{
				Node:     rb,
				Location: loc,
			})
			i.indexedReferences[rb] = true
		}

		// Get the document path for the resolved request body
		info := rb.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := rb.GetObject()
		if resolved != nil {
			wrapped := &ReferencedRequestBody{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := rb.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "requestBodies") {
		if !i.indexedRequestBodies[obj] {
			i.ComponentRequestBodies = append(i.ComponentRequestBodies, &IndexNode[*ReferencedRequestBody]{
				Node:     rb,
				Location: loc,
			})
			i.indexedRequestBodies[obj] = true
		}
		return
	}

	// Check if this is a top-level request body in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedRequestBodies[obj] {
			i.ExternalRequestBodies = append(i.ExternalRequestBodies, &IndexNode[*ReferencedRequestBody]{
				Node:     rb,
				Location: loc,
			})
			i.indexedRequestBodies[obj] = true
		}
		return
	}

	// Everything else is an inline request body
	if !i.indexedRequestBodies[obj] {
		i.InlineRequestBodies = append(i.InlineRequestBodies, &IndexNode[*ReferencedRequestBody]{
			Node:     rb,
			Location: loc,
		})
		i.indexedRequestBodies[obj] = true
	}
}

func (i *Index) indexReferencedHeader(ctx context.Context, loc Locations, header *ReferencedHeader) {
	if header == nil {
		return
	}

	if header.IsReference() && !header.IsResolved() {
		resolveAndValidateReference(i, ctx, header)
	}

	if header.IsReference() {
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[header] {
			i.HeaderReferences = append(i.HeaderReferences, &IndexNode[*ReferencedHeader]{
				Node:     header,
				Location: loc,
			})
			i.indexedReferences[header] = true
		}

		// Get the document path for the resolved header
		info := header.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := header.GetObject()
		if resolved != nil {
			wrapped := &ReferencedHeader{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := header.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "headers") {
		if !i.indexedHeaders[obj] {
			i.ComponentHeaders = append(i.ComponentHeaders, &IndexNode[*ReferencedHeader]{
				Node:     header,
				Location: loc,
			})
			i.indexedHeaders[obj] = true
		}
		return
	}

	// Check if this is a top-level header in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedHeaders[obj] {
			i.ExternalHeaders = append(i.ExternalHeaders, &IndexNode[*ReferencedHeader]{
				Node:     header,
				Location: loc,
			})
			i.indexedHeaders[obj] = true
		}
		return
	}

	// Everything else is an inline header
	if !i.indexedHeaders[obj] {
		i.InlineHeaders = append(i.InlineHeaders, &IndexNode[*ReferencedHeader]{
			Node:     header,
			Location: loc,
		})
		i.indexedHeaders[obj] = true
	}
}

func (i *Index) indexReferencedExample(ctx context.Context, loc Locations, example *ReferencedExample) {
	if example == nil {
		return
	}

	if example.IsReference() && !example.IsResolved() {
		resolveAndValidateReference(i, ctx, example)
	}

	if example.IsReference() {
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[example] {
			i.ExampleReferences = append(i.ExampleReferences, &IndexNode[*ReferencedExample]{
				Node:     example,
				Location: loc,
			})
			i.indexedReferences[example] = true
		}

		// Get the document path for the resolved example
		info := example.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := example.GetObject()
		if resolved != nil {
			wrapped := &ReferencedExample{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := example.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "examples") {
		if !i.indexedExamples[obj] {
			i.ComponentExamples = append(i.ComponentExamples, &IndexNode[*ReferencedExample]{
				Node:     example,
				Location: loc,
			})
			i.indexedExamples[obj] = true
		}
		return
	}

	// Check if this is a top-level example in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedExamples[obj] {
			i.ExternalExamples = append(i.ExternalExamples, &IndexNode[*ReferencedExample]{
				Node:     example,
				Location: loc,
			})
			i.indexedExamples[obj] = true
		}
		return
	}

	// Everything else is an inline example
	if !i.indexedExamples[obj] {
		i.InlineExamples = append(i.InlineExamples, &IndexNode[*ReferencedExample]{
			Node:     example,
			Location: loc,
		})
		i.indexedExamples[obj] = true
	}
}

func (i *Index) indexReferencedLink(ctx context.Context, loc Locations, link *ReferencedLink) {
	if link == nil {
		return
	}

	if link.IsReference() && !link.IsResolved() {
		resolveAndValidateReference(i, ctx, link)
	}

	if link.IsReference() {
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[link] {
			i.LinkReferences = append(i.LinkReferences, &IndexNode[*ReferencedLink]{
				Node:     link,
				Location: loc,
			})
			i.indexedReferences[link] = true
		}

		// Get the document path for the resolved link
		info := link.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := link.GetObject()
		if resolved != nil {
			wrapped := &ReferencedLink{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := link.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "links") {
		if !i.indexedLinks[obj] {
			i.ComponentLinks = append(i.ComponentLinks, &IndexNode[*ReferencedLink]{
				Node:     link,
				Location: loc,
			})
			i.indexedLinks[obj] = true
		}
		return
	}

	// Check if this is a top-level link in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedLinks[obj] {
			i.ExternalLinks = append(i.ExternalLinks, &IndexNode[*ReferencedLink]{
				Node:     link,
				Location: loc,
			})
			i.indexedLinks[obj] = true
		}
		return
	}

	// Everything else is an inline link
	if !i.indexedLinks[obj] {
		i.InlineLinks = append(i.InlineLinks, &IndexNode[*ReferencedLink]{
			Node:     link,
			Location: loc,
		})
		i.indexedLinks[obj] = true
	}
}

func (i *Index) indexReferencedCallback(ctx context.Context, loc Locations, callback *ReferencedCallback) {
	if callback == nil {
		return
	}

	if callback.IsReference() && !callback.IsResolved() {
		resolveAndValidateReference(i, ctx, callback)
	}

	if callback.IsReference() {
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[callback] {
			i.CallbackReferences = append(i.CallbackReferences, &IndexNode[*ReferencedCallback]{
				Node:     callback,
				Location: loc,
			})
			i.indexedReferences[callback] = true
		}

		// Get the document path for the resolved callback
		info := callback.GetReferenceResolutionInfo()
		var docPath string
		if info != nil {
			docPath = info.AbsoluteDocumentPath
		}

		// Push document path onto document stack BEFORE walking
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

		// If resolved, explicitly walk the resolved content
		resolved := callback.GetObject()
		if resolved != nil {
			wrapped := &ReferencedCallback{Object: resolved}
			_ = buildIndex(ctx, i, wrapped)
		}
		return
	}

	obj := callback.GetObject()
	if obj == nil {
		return
	}

	if isTopLevelComponent(loc, "callbacks") {
		if !i.indexedCallbacks[obj] {
			i.ComponentCallbacks = append(i.ComponentCallbacks, &IndexNode[*ReferencedCallback]{
				Node:     callback,
				Location: loc,
			})
			i.indexedCallbacks[obj] = true
		}
		return
	}

	// Check if this is a top-level callback in an external document
	// Important: Only mark as external if it's NOT from the main document
	if isTopLevelExternalSchema(loc) {
		if !i.isFromMainDocument() && !i.indexedCallbacks[obj] {
			i.ExternalCallbacks = append(i.ExternalCallbacks, &IndexNode[*ReferencedCallback]{
				Node:     callback,
				Location: loc,
			})
			i.indexedCallbacks[obj] = true
		}
		return
	}

	// Everything else is an inline callback
	if !i.indexedCallbacks[obj] {
		i.InlineCallbacks = append(i.InlineCallbacks, &IndexNode[*ReferencedCallback]{
			Node:     callback,
			Location: loc,
		})
		i.indexedCallbacks[obj] = true
	}
}

func (i *Index) indexReferencedSecurityScheme(ctx context.Context, loc Locations, ss *ReferencedSecurityScheme) {
	if ss == nil {
		return
	}

	if ss.IsReference() && !ss.IsResolved() {
		resolveAndValidateReference(i, ctx, ss)
	}

	if ss.IsReference() {
		// Add to references list only if this exact reference object hasn't been indexed
		if !i.indexedReferences[ss] {
			i.SecuritySchemeReferences = append(i.SecuritySchemeReferences, &IndexNode[*ReferencedSecurityScheme]{
				Node:     ss,
				Location: loc,
			})
			i.indexedReferences[ss] = true
		}
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

	// Get the parent schema for this segment
	var parentSchemaRef *oas3.JSONSchemaReferenceable
	_ = loc.ParentMatchFunc(Matcher{
		Schema: func(s *oas3.JSONSchemaReferenceable) error {
			parentSchemaRef = s
			return nil
		},
	})
	segment.ParentSchema = parentSchemaRef

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

			// Use the ParentSchema from the segment (which has the oneOf/anyOf)
			// instead of the schema parameter (which is the $ref)
			parentSchema := segment.ParentSchema
			if parentSchema == nil {
				parentSchema = schema // Fallback to old behavior if ParentSchema not set
			}

			totalBranches := countPolymorphicBranches(parentSchema, segment.Field)
			polymorphicInfo := &PolymorphicCircularRef{
				ParentSchema:   parentSchema,
				ParentLocation: parentLoc,
				Field:          segment.Field,
				BranchResults:  make(map[int]CircularClassification),
				TotalBranches:  totalBranches,
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
				i.invalidCircularRefs++
				i.circularErrs = append(i.circularErrs, validation.NewValidationErrorWithDocumentLocation(
					validation.SeverityError,
					"circular-reference-invalid",
					fmt.Errorf("non-terminating circular reference: all %s branches recurse with no base case", ref.Field),
					getSchemaErrorNode(ref.ParentSchema),
					i.documentPathForSchema(ref.ParentSchema),
				))
			} else if !allInvalid && ref.TotalBranches > 0 {
				// At least one branch allows termination - this is a valid circular ref
				i.validCircularRefs++
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
