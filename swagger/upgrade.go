package swagger

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
)

// Upgrade converts a Swagger 2.0 document into an OpenAPI 3.0 document.
//
// The conversion performs the following major transformations:
// - swagger: "2.0" -> openapi: "3.0.0"
// - host/basePath/schemes -> servers
// - definitions -> components.schemas
// - parameters (global non-body) -> components.parameters
// - parameters (global body) -> components.requestBodies
// - responses (global) -> components.responses
// - securityDefinitions -> components.securitySchemes
// - operation parameters:
//   - in: body -> requestBody with content and schema
//   - in: formData -> requestBody with x-www-form-urlencoded or multipart/form-data schema
//   - other parameters carried over with schema and style/explode derived from collectionFormat
//
// - responses: schema/examples -> content[mediaType].schema/example
// - Rewrites JSON Schema $ref targets from "#/definitions/..." to "#/components/schemas/..."
func Upgrade(ctx context.Context, src *Swagger) (*openapi.OpenAPI, error) {
	if src == nil {
		return nil, nil
	}

	dst := &openapi.OpenAPI{
		OpenAPI: "3.0.0",
		Info:    convertInfo(src.Info),
		Tags:    convertTags(src.Tags),
	}

	// Servers
	dst.Servers = buildServers(src)

	// Paths
	dst.Paths = convertPaths(src)

	// Components (only set when any sub-section is non-nil)
	compSchemas := convertDefinitions(src.Definitions)
	compParams := convertGlobalParameters(src)
	compReqBodies := convertGlobalRequestBodies(src)
	compResponses := convertGlobalResponses(src)
	compSecSchemes := convertSecuritySchemes(src)

	if compSchemas != nil || compParams != nil || compReqBodies != nil || compResponses != nil || compSecSchemes != nil {
		dst.Components = &openapi.Components{
			Schemas:         compSchemas,
			Parameters:      compParams,
			RequestBodies:   compReqBodies,
			Responses:       compResponses,
			SecuritySchemes: compSecSchemes,
		}
	}

	// Security requirements
	dst.Security = convertSecurityRequirements(src)

	// External docs (root)
	dst.ExternalDocs = convertExternalDocs(src.ExternalDocs)

	// Rewrite schema $refs from "#/definitions/" -> "#/components/schemas/"
	rewriteRefTargets(ctx, dst)

	return dst, nil
}

func convertInfo(src Info) openapi.Info {
	return openapi.Info{
		Title:       src.Title,
		Version:     src.Version,
		Description: src.Description,
		TermsOfService: func() *string {
			if src.TermsOfService == nil || *src.TermsOfService == "" {
				return nil
			}
			return src.TermsOfService
		}(),
		Contact: convertInfoContact(src.Contact),
		License: convertInfoLicense(src.License),
		Extensions: func() *extensions.Extensions {
			if src.Extensions == nil {
				return nil
			}
			ext := extensions.New()
			_ = ext.Populate(src.Extensions)
			return ext
		}(),
	}
}

func convertInfoContact(src *Contact) *openapi.Contact {
	if src == nil {
		return nil
	}
	return &openapi.Contact{
		Name:       src.Name,
		URL:        src.URL,
		Email:      src.Email,
		Extensions: copyExtensions(src.Extensions),
	}
}

func convertInfoLicense(src *License) *openapi.License {
	if src == nil {
		return nil
	}
	return &openapi.License{
		Name:       src.Name,
		URL:        src.URL,
		Extensions: copyExtensions(src.Extensions),
	}
}

func copyExtensions(src *extensions.Extensions) *extensions.Extensions {
	if src == nil {
		return nil
	}
	dst := extensions.New()
	_ = dst.Populate(src)
	return dst
}

func convertExternalDocs(src *ExternalDocumentation) *oas3.ExternalDocumentation {
	if src == nil {
		return nil
	}
	return &oas3.ExternalDocumentation{
		Description: src.Description,
		URL:         src.URL,
		Extensions:  copyExtensions(src.Extensions),
	}
}

func convertTags(src []*Tag) []*openapi.Tag {
	if len(src) == 0 {
		return nil
	}
	out := make([]*openapi.Tag, 0, len(src))
	for _, t := range src {
		if t == nil {
			continue
		}
		out = append(out, &openapi.Tag{
			Name:         t.Name,
			Description:  t.Description,
			ExternalDocs: convertExternalDocs(t.ExternalDocs),
			Extensions:   copyExtensions(t.Extensions),
		})
	}
	return out
}

func buildServers(src *Swagger) []*openapi.Server {
	host := src.GetHost()
	basePath := src.GetBasePath()
	schemes := src.GetSchemes()

	if host == "" {
		// No absolute server configured; rely on default "/" server (OpenAPI.GetServers fallback)
		return nil
	}

	pathsuffix := basePath
	if pathsuffix == "" {
		pathsuffix = "/"
	}
	pathsuffix = ensureLeadingSlash(pathsuffix)

	if len(schemes) == 0 {
		// Default to https if host present
		schemes = []string{"https"}
	}

	var servers []*openapi.Server
	seen := map[string]struct{}{}
	for _, sch := range schemes {
		url := fmt.Sprintf("%s://%s%s", sch, host, pathsuffix)
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		servers = append(servers, &openapi.Server{URL: url})
	}

	return servers
}

func ensureLeadingSlash(s string) string {
	if s == "" {
		return "/"
	}
	if !strings.HasPrefix(s, "/") {
		return "/" + s
	}
	return s
}

func convertDefinitions(defs *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Concrete]]) *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]] {
	if defs == nil || defs.Len() == 0 {
		return nil
	}
	out := sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]]()
	for name, schema := range defs.All() {
		if schema == nil {
			continue
		}
		out.Set(name, oas3.ConcreteToReferenceable(schema))
	}
	return out
}

func convertSecuritySchemes(src *Swagger) *sequencedmap.Map[string, *openapi.ReferencedSecurityScheme] {
	if src.SecurityDefinitions == nil || src.SecurityDefinitions.Len() == 0 {
		return nil
	}
	out := sequencedmap.New[string, *openapi.ReferencedSecurityScheme]()
	for name, s := range src.SecurityDefinitions.All() {
		if s == nil {
			continue
		}
		dst := &openapi.SecurityScheme{
			Extensions: copyExtensions(s.Extensions),
		}
		switch s.Type {
		case SecuritySchemeTypeBasic:
			dst.Type = openapi.SecuritySchemeTypeHTTP
			dst.Scheme = pointer.From("basic")
		case SecuritySchemeTypeAPIKey:
			dst.Type = openapi.SecuritySchemeTypeAPIKey
			dst.Name = s.Name
			if s.In != nil {
				switch *s.In {
				case SecuritySchemeInHeader:
					in := openapi.SecuritySchemeInHeader
					dst.In = &in
				case SecuritySchemeInQuery:
					in := openapi.SecuritySchemeInQuery
					dst.In = &in
				default:
					// Swagger 2.0 doesn't support cookie for apiKey
				}
			}
		case SecuritySchemeTypeOAuth2:
			dst.Type = openapi.SecuritySchemeTypeOAuth2
			dst.Flows = convertOAuth2Flows(s)
		default:
			// unsupported; copy as apiKey header by default to keep spec valid minimally
			dst.Type = openapi.SecuritySchemeTypeAPIKey
			n := pointer.From("Authorization")
			dst.Name = n
			in := openapi.SecuritySchemeInHeader
			dst.In = &in
		}
		out.Set(name, openapi.NewReferencedSecuritySchemeFromSecurityScheme(dst))
	}
	return out
}

func convertOAuth2Flows(s *SecurityScheme) *openapi.OAuthFlows {
	if s == nil {
		return nil
	}
	flows := &openapi.OAuthFlows{
		Extensions: copyExtensions(s.Extensions),
	}
	if s.Flow == nil {
		return flows
	}
	switch *s.Flow {
	case OAuth2FlowImplicit:
		flows.Implicit = &openapi.OAuthFlow{
			AuthorizationURL: s.AuthorizationURL,
			TokenURL:         nil,
			RefreshURL:       nil,
			Scopes:           cloneStringMap(s.Scopes),
		}
	case OAuth2FlowPassword:
		flows.Password = &openapi.OAuthFlow{
			TokenURL:   s.TokenURL,
			Scopes:     cloneStringMap(s.Scopes),
			Extensions: nil,
		}
	case OAuth2FlowApplication:
		flows.ClientCredentials = &openapi.OAuthFlow{
			TokenURL:   s.TokenURL,
			Scopes:     cloneStringMap(s.Scopes),
			Extensions: nil,
		}
	case OAuth2FlowAccessCode:
		flows.AuthorizationCode = &openapi.OAuthFlow{
			AuthorizationURL: s.AuthorizationURL,
			TokenURL:         s.TokenURL,
			Scopes:           cloneStringMap(s.Scopes),
		}
	}
	return flows
}

func cloneStringMap(m *sequencedmap.Map[string, string]) *sequencedmap.Map[string, string] {
	if m == nil || m.Len() == 0 {
		return sequencedmap.New[string, string]()
	}
	out := sequencedmap.New[string, string]()
	for k, v := range m.All() {
		out.Set(k, v)
	}
	return out
}

func convertSecurityRequirements(src *Swagger) []*openapi.SecurityRequirement {
	if len(src.Security) == 0 {
		return nil
	}
	var out []*openapi.SecurityRequirement
	for _, req := range src.Security {
		if req == nil {
			continue
		}
		dst := openapi.NewSecurityRequirement()
		for k, v := range req.All() {
			dst.Set(k, v)
		}
		out = append(out, dst)
	}
	return out
}

func convertPaths(src *Swagger) *openapi.Paths {
	if src.Paths == nil || src.Paths.Len() == 0 {
		return openapi.NewPaths()
	}

	dst := openapi.NewPaths()
	// Stable order for deterministic output
	paths := make([]string, 0, src.Paths.Len())
	for p := range src.Paths.Keys() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		pathItem, _ := src.Paths.Get(p)
		if pathItem == nil {
			continue
		}
		dst.Set(p, openapi.NewReferencedPathItemFromPathItem(convertPathItem(src, pathItem)))
	}

	return dst
}

func convertPathItem(root *Swagger, src *PathItem) *openapi.PathItem {
	dst := openapi.NewPathItem()
	// Path-level parameters (non-body only in OAS3)
	for _, rp := range src.Parameters {
		if rp == nil {
			continue
		}
		if rp.IsReference() {
			// Resolve reference name to decide if body or not
			name := localComponentName(rp.GetReference())
			if name == "" {
				continue
			}
			if root.Parameters != nil {
				if gp, ok := root.Parameters.Get(name); ok && gp != nil {
					if gp.In == ParameterInBody {
						// skip; cannot put body parameter at path level; OAS3 has no path-level requestBody
						continue
					}
				}
			}
			// Non-body parameter reference -> components.parameters
			ref := references.Reference("#/components/parameters/" + name)
			if dst.Parameters == nil {
				dst.Parameters = []*openapi.ReferencedParameter{}
			}
			dst.Parameters = append(dst.Parameters, openapi.NewReferencedParameterFromRef(ref))
			continue
		}
		// Inline
		if srcp := rp.GetObject(); srcp != nil && srcp.In != ParameterInBody {
			if dst.Parameters == nil {
				dst.Parameters = []*openapi.ReferencedParameter{}
			}
			dst.Parameters = append(dst.Parameters, openapi.NewReferencedParameterFromParameter(convertParameter(srcp)))
		}
	}

	// Operations
	for method, op := range src.All() {
		if op == nil {
			continue
		}
		dst.Set(openapi.HTTPMethod(strings.ToLower(string(method))), convertOperation(root, op))
	}

	return dst
}

func convertOperation(root *Swagger, src *Operation) *openapi.Operation {
	dst := &openapi.Operation{
		OperationID: src.OperationID,
		Summary:     src.Summary,
		Description: src.Description,
		Deprecated:  src.Deprecated,
		Extensions:  copyExtensions(src.Extensions),
		Responses:   openapi.NewResponses(),
	}
	// Only set tags if present to avoid emitting empty arrays
	if len(src.Tags) > 0 {
		dst.Tags = append([]string{}, src.Tags...)
	}

	// Security requirements
	if len(src.Security) > 0 {
		dst.Security = make([]*openapi.SecurityRequirement, 0, len(src.Security))
		for _, req := range src.Security {
			if req == nil {
				continue
			}
			secReq := openapi.NewSecurityRequirement()
			for k, v := range req.All() {
				secReq.Set(k, v)
			}
			dst.Security = append(dst.Security, secReq)
		}
	}

	// Determine consumes/produces for this operation
	consumes := src.Consumes
	if len(consumes) == 0 {
		consumes = root.Consumes
	}
	produces := src.Produces
	if len(produces) == 0 {
		produces = root.Produces
	}
	if len(produces) == 0 {
		produces = []string{"application/json"}
	}
	if len(consumes) == 0 {
		consumes = []string{"application/json"}
	}

	// Parameters -> Parameters + RequestBody
	formParams := []*Parameter{}
	var bodyParam *Parameter

	for _, rp := range src.Parameters {
		if rp == nil {
			continue
		}
		if rp.IsReference() {
			// Reference to global parameter
			name := localComponentName(rp.GetReference())
			if name == "" {
				continue
			}
			if root.Parameters != nil {
				if gp, ok := root.Parameters.Get(name); ok && gp != nil {
					switch gp.In {
					case ParameterInBody:
						// Use requestBodies reference
						dst.RequestBody = openapi.NewReferencedRequestBodyFromRef(references.Reference("#/components/requestBodies/" + name))
					case ParameterInFormData:
						formParams = append(formParams, gp)
					default:
						// Carry as parameter reference
						if dst.Parameters == nil {
							dst.Parameters = []*openapi.ReferencedParameter{}
						}
						dst.Parameters = append(dst.Parameters, openapi.NewReferencedParameterFromRef(references.Reference("#/components/parameters/"+name)))
					}
					continue
				}
			}
			// Fallback: treat as parameter ref
			if dst.Parameters == nil {
				dst.Parameters = []*openapi.ReferencedParameter{}
			}
			dst.Parameters = append(dst.Parameters, openapi.NewReferencedParameterFromRef(references.Reference("#/components/parameters/"+name)))
			continue
		}

		// Inline parameter
		p := rp.GetObject()
		if p == nil {
			continue
		}
		switch p.In {
		case ParameterInBody:
			bodyParam = p
		case ParameterInFormData:
			formParams = append(formParams, p)
		default:
			if dst.Parameters == nil {
				dst.Parameters = []*openapi.ReferencedParameter{}
			}
			dst.Parameters = append(dst.Parameters, openapi.NewReferencedParameterFromParameter(convertParameter(p)))
		}
	}

	// Build requestBody from body parameter if present
	if dst.RequestBody == nil && bodyParam != nil {
		rb := &openapi.RequestBody{
			Description: bodyParam.Description,
			Required:    bodyParam.Required,
			Content:     sequencedmap.New[string, *openapi.MediaType](),
			Extensions:  nil,
		}
		// Create media types from consumes
		for _, mt := range consumes {
			mt = strings.TrimSpace(mt)
			if mt == "" {
				continue
			}
			rb.Content.Set(mt, &openapi.MediaType{
				Schema: bodyParam.Schema,
			})
		}
		dst.RequestBody = openapi.NewReferencedRequestBodyFromRequestBody(rb)
	}

	// Build requestBody from formData if any
	if dst.RequestBody == nil && len(formParams) > 0 {
		mediaType := "application/x-www-form-urlencoded"
		for _, fp := range formParams {
			if fp.Type != nil && *fp.Type == "file" {
				mediaType = "multipart/form-data"
				break
			}
		}

		obj := &oas3.Schema{
			Type:       oas3.NewTypeFromString(oas3.SchemaType("object")),
			Properties: sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]](),
		}
		// required list is optional; omitted for minimal conversion
		for _, fp := range formParams {
			propSchema := schemaForSwaggerParamType(fp, true)
			obj.Properties.Set(fp.Name, propSchema)
		}

		rb := &openapi.RequestBody{
			Required: pointer.From(anyRequired(formParams)),
			Content:  sequencedmap.New(sequencedmap.NewElem(mediaType, &openapi.MediaType{Schema: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](obj)})),
		}
		dst.RequestBody = openapi.NewReferencedRequestBodyFromRequestBody(rb)
	}

	// Responses
	if src.Responses != nil {
		// Default
		if src.Responses.Default != nil {
			dst.Responses.Default = convertReferencedResponse(src.Responses.Default, produces)
		}
		// Codes
		for code, rr := range src.Responses.All() {
			dst.Responses.Set(code, convertReferencedResponse(rr, produces))
		}
	}

	return dst
}

func anyRequired(params []*Parameter) bool {
	for _, p := range params {
		if p.GetRequired() {
			return true
		}
	}
	return false
}

func schemaForSwaggerParamType(p *Parameter, copyDescription bool) *oas3.JSONSchema[oas3.Referenceable] {
	if p == nil {
		return nil
	}
	switch {
	case p.Type != nil && *p.Type == "array":
		items := &oas3.Schema{Type: oas3.NewTypeFromString(oas3.SchemaType("string"))}
		if p.Items != nil {
			if p.Items.Type != "" {
				items.Type = oas3.NewTypeFromString(oas3.SchemaType(strings.ToLower(p.Items.Type)))
			}
			// Preserve enum and default from items
			if len(p.Items.Enum) > 0 {
				items.Enum = make([]values.Value, len(p.Items.Enum))
				copy(items.Enum, p.Items.Enum)
			}
			if p.Items.Default != nil {
				items.Default = p.Items.Default
			}
		}
		return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type:  oas3.NewTypeFromString(oas3.SchemaType("array")),
			Items: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](items),
		})
	case p.Type != nil && *p.Type == "file":
		schema := &oas3.Schema{
			Type:   oas3.NewTypeFromString(oas3.SchemaType("string")),
			Format: pointer.From("binary"),
		}
		if p.Description != nil && copyDescription {
			schema.Description = p.Description
		}
		return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)
	case p.Type != nil && *p.Type != "":
		schema := &oas3.Schema{
			Type: oas3.NewTypeFromString(oas3.SchemaType(strings.ToLower(*p.Type))),
		}
		if p.Description != nil && copyDescription {
			schema.Description = p.Description
		}
		if p.Format != nil {
			schema.Format = p.Format
		}
		if p.Minimum != nil {
			schema.Minimum = p.Minimum
		}
		if p.Maximum != nil {
			schema.Maximum = p.Maximum
		}
		if len(p.Enum) > 0 {
			schema.Enum = make([]values.Value, len(p.Enum))
			copy(schema.Enum, p.Enum)
		}
		if p.Default != nil {
			schema.Default = p.Default
		}
		return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)
	default:
		// Body parameter case should not call this; fall back to string
		return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type: oas3.NewTypeFromString(oas3.SchemaType("string")),
		})
	}
}

func convertParameter(p *Parameter) *openapi.Parameter {
	if p == nil {
		return nil
	}
	dst := &openapi.Parameter{
		Name:        p.Name,
		Description: p.Description,
		Required:    p.Required,
		Deprecated:  nil, // Swagger 2.0 parameter doesn't have deprecated
		Schema:      nil,
		Content:     nil,
		Extensions:  copyExtensions(p.Extensions),
	}

	// in
	switch p.In {
	case ParameterInQuery:
		dst.In = openapi.ParameterInQuery
	case ParameterInHeader:
		dst.In = openapi.ParameterInHeader
	case ParameterInPath:
		dst.In = openapi.ParameterInPath
	default:
		// Cookie not in Swagger 2.0
		dst.In = openapi.ParameterInQuery
	}

	// schema from type/format/items (non-body only)
	if p.In != ParameterInBody {
		dst.Schema = schemaForSwaggerParamType(p, false)
		// collectionFormat -> style/explode
		if p.CollectionFormat != nil {
			switch *p.CollectionFormat {
			case CollectionFormatMulti:
				// style=form explode=true (default for query)
				style := openapi.SerializationStyleForm
				dst.Style = &style
				dst.Explode = pointer.From(true)
			case CollectionFormatCSV:
				style := openapi.SerializationStyleForm
				dst.Style = &style
				dst.Explode = pointer.From(false)
			case CollectionFormatSSV:
				style := openapi.SerializationStyleSpaceDelimited
				dst.Style = &style
				dst.Explode = pointer.From(false)
			case CollectionFormatPipes:
				style := openapi.SerializationStylePipeDelimited
				dst.Style = &style
				dst.Explode = pointer.From(false)
			default:
				// tsv or unknown -> default form + explode=false
				style := openapi.SerializationStyleForm
				dst.Style = &style
				dst.Explode = pointer.From(false)
			}
		}
	}

	return dst
}

func convertReferencedResponse(rr *ReferencedResponse, produces []string) *openapi.ReferencedResponse {
	if rr == nil {
		return nil
	}
	if rr.IsReference() {
		// Global response reference -> components.responses
		name := localComponentName(rr.GetReference())
		return openapi.NewReferencedResponseFromRef(references.Reference("#/components/responses/" + name))
	}
	src := rr.GetObject()
	if src == nil {
		return nil
	}

	dst := &openapi.Response{
		Description: src.Description,
		Headers:     convertResponseHeaders(src.Headers),
		// Content will be created only when there is a schema to describe
		Content:    nil,
		Extensions: copyExtensions(src.Extensions),
	}

	if src.Schema != nil {
		// Build content entries for each produces
		if len(produces) == 0 {
			produces = []string{"application/json"}
		}
		// Initialize content map before setting entries
		if dst.Content == nil {
			dst.Content = sequencedmap.New[string, *openapi.MediaType]()
		}
		for _, mt := range produces {
			mt = strings.TrimSpace(mt)
			if mt == "" {
				continue
			}
			dst.Content.Set(mt, &openapi.MediaType{
				Schema:  src.Schema,
				Example: exampleForMediaType(mt, src),
			})
		}
	}

	return openapi.NewReferencedResponseFromResponse(dst)
}

func exampleForMediaType(mt string, src *Response) values.Value {
	if src == nil || src.Examples == nil || src.Examples.Len() == 0 {
		return nil
	}
	// Match exact or wildcard examples
	if v, ok := src.Examples.Get(mt); ok {
		return v
	}
	// Common defaults
	for _, cand := range []string{"application/json", "application/xml", "text/plain"} {
		if v, ok := src.Examples.Get(cand); ok {
			return v
		}
	}
	return nil
}

func convertResponseHeaders(hdrs *sequencedmap.Map[string, *Header]) *sequencedmap.Map[string, *openapi.ReferencedHeader] {
	if hdrs == nil || hdrs.Len() == 0 {
		return nil
	}
	out := sequencedmap.New[string, *openapi.ReferencedHeader]()
	for name, h := range hdrs.All() {
		if h == nil {
			continue
		}
		dst := &openapi.Header{
			Description: h.Description,
			Schema: func() *oas3.JSONSchema[oas3.Referenceable] {
				// Convert simple header types
				if h.Type == "" {
					return nil
				}
				switch strings.ToLower(h.Type) {
				case "array":
					items := &oas3.Schema{Type: oas3.NewTypeFromString(oas3.SchemaType("string"))}
					if h.Items != nil && h.Items.Type != "" {
						items.Type = oas3.NewTypeFromString(oas3.SchemaType(strings.ToLower(h.Items.Type)))
					}
					return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
						Type:  oas3.NewTypeFromString(oas3.SchemaType("array")),
						Items: oas3.NewJSONSchemaFromSchema[oas3.Referenceable](items),
					})
				default:
					return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
						Type:   oas3.NewTypeFromString(oas3.SchemaType(strings.ToLower(h.Type))),
						Format: h.Format,
					})
				}
			}(),
			Extensions: copyExtensions(h.Extensions),
		}
		out.Set(name, openapi.NewReferencedHeaderFromHeader(dst))
	}
	return out
}

func convertGlobalParameters(src *Swagger) *sequencedmap.Map[string, *openapi.ReferencedParameter] {
	if src.Parameters == nil || src.Parameters.Len() == 0 {
		return nil
	}
	out := sequencedmap.New[string, *openapi.ReferencedParameter]()
	for name, p := range src.Parameters.All() {
		if p == nil {
			continue
		}
		if p.In == ParameterInBody {
			// Skip; handled as requestBodies
			continue
		}
		out.Set(name, openapi.NewReferencedParameterFromParameter(convertParameter(p)))
	}
	return out
}

func convertGlobalRequestBodies(src *Swagger) *sequencedmap.Map[string, *openapi.ReferencedRequestBody] {
	if src.Parameters == nil || src.Parameters.Len() == 0 {
		return nil
	}
	var count int
	out := sequencedmap.New[string, *openapi.ReferencedRequestBody]()
	for name, p := range src.Parameters.All() {
		if p == nil || p.In != ParameterInBody {
			continue
		}
		count++
		rb := &openapi.RequestBody{
			Description: p.Description,
			Required:    p.Required,
			Content:     sequencedmap.New[string, *openapi.MediaType](),
		}
		// Use global consumes or default
		consumes := src.Consumes
		if len(consumes) == 0 {
			consumes = []string{"application/json"}
		}
		for _, mt := range consumes {
			mt = strings.TrimSpace(mt)
			if mt == "" {
				continue
			}
			rb.Content.Set(mt, &openapi.MediaType{
				Schema: p.Schema,
			})
		}
		out.Set(name, openapi.NewReferencedRequestBodyFromRequestBody(rb))
	}
	if count == 0 {
		return nil
	}
	return out
}

func convertGlobalResponses(src *Swagger) *sequencedmap.Map[string, *openapi.ReferencedResponse] {
	if src.Responses == nil || src.Responses.Len() == 0 {
		return nil
	}
	out := sequencedmap.New[string, *openapi.ReferencedResponse]()
	// Determine fallback produces
	produces := src.Produces
	if len(produces) == 0 {
		produces = []string{"application/json"}
	}
	for name, r := range src.Responses.All() {
		out.Set(name, convertReferencedResponse(&ReferencedResponse{Object: r}, produces))
	}
	return out
}

func localComponentName(ref references.Reference) string {
	s := string(ref)
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func rewriteRefTargets(ctx context.Context, doc *openapi.OpenAPI) {
	if doc == nil {
		return
	}
	for item := range openapi.Walk(ctx, doc) {
		_ = item.Match(openapi.Matcher{
			Schema: func(js *oas3.JSONSchema[oas3.Referenceable]) error {
				if js == nil || !js.IsReference() {
					return nil
				}
				ref := string(js.GetReference())
				if strings.HasPrefix(ref, "#/definitions/") {
					newRef := references.Reference(strings.Replace(ref, "#/definitions/", "#/components/schemas/", 1))
					// Update only the reference, preserving all other metadata (XML, etc.)
					schema := js.GetSchema()
					if schema != nil {
						schema.Ref = pointer.From(newRef)
					}
				}
				return nil
			},
		})
	}
}
