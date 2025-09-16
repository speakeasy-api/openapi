package openapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/speakeasy-api/openapi/cmd/openapi/internal/tui/navigator"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
)

// BuildTree creates a navigation tree from an OpenAPI document
func BuildTree(doc *openapi.OpenAPI) navigator.TreeNode {
	// Get OpenAPI version for the root title
	version := doc.GetOpenAPI()
	if version == "" {
		version = openapi.Version // Default fallback
	}

	root := &navigator.BaseNode{
		ID:          "root",
		Content:     fmt.Sprintf("OpenAPI %s", version),
		Description: getDocumentTitle(doc),
		Type:        navigator.NodeTypeRoot,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	// Add document info
	if info := doc.GetInfo(); info != nil {
		root.AddChild(buildInfoNode(info))
	}

	// Add servers
	if servers := doc.GetServers(); len(servers) > 0 {
		root.AddChild(buildServersNode(servers))
	}

	// Add paths
	if paths := doc.Paths; paths != nil && paths.Len() > 0 {
		root.AddChild(buildPathsNode(paths))
	}

	// Add components
	if components := doc.Components; components != nil {
		root.AddChild(buildComponentsNode(components))
	}

	// Add security
	if security := doc.Security; len(security) > 0 {
		root.AddChild(buildSecurityNode(security))
	}

	// Add tags
	if tags := doc.GetTags(); len(tags) > 0 {
		root.AddChild(buildTagsNode(tags))
	}

	return root
}

// getDocumentTitle extracts a meaningful title from the OpenAPI document
func getDocumentTitle(doc *openapi.OpenAPI) string {
	if info := doc.GetInfo(); info != nil {
		if title := info.GetTitle(); title != "" {
			version := info.GetVersion()
			if version != "" {
				return fmt.Sprintf("%s v%s", title, version)
			}
			return title
		}
	}
	return "Untitled API"
}

// buildInfoNode creates a node for the info section
func buildInfoNode(info *openapi.Info) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "info",
		Content:     "Info",
		Description: info.GetTitle(),
		Type:        navigator.NodeTypeInfo,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	// Add info details as children
	if title := info.GetTitle(); title != "" {
		node.AddChild(&navigator.BaseNode{
			ID:       "info.title",
			Content:  fmt.Sprintf("Title: %s", title),
			Type:     navigator.NodeTypeInfo,
			Children: []navigator.TreeNode{},
		})
	}

	if version := info.GetVersion(); version != "" {
		node.AddChild(&navigator.BaseNode{
			ID:       "info.version",
			Content:  fmt.Sprintf("Version: %s", version),
			Type:     navigator.NodeTypeInfo,
			Children: []navigator.TreeNode{},
		})
	}

	if description := info.GetDescription(); description != "" {
		node.AddChild(&navigator.BaseNode{
			ID:       "info.description",
			Content:  fmt.Sprintf("Description: %s", description),
			Type:     navigator.NodeTypeInfo,
			Children: []navigator.TreeNode{},
		})
	}

	return node
}

// buildServersNode creates a node for the servers section
func buildServersNode(servers []*openapi.Server) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "servers",
		Content:     "Servers",
		Description: fmt.Sprintf("%d server(s)", len(servers)),
		Type:        navigator.NodeTypeServers,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	for i, server := range servers {
		serverNode := &navigator.BaseNode{
			ID:           fmt.Sprintf("server.%d", i),
			Content:      fmt.Sprintf("- url: %s", server.GetURL()),
			DisplayTitle: server.GetURL(), // Clean display for top-level
			Type:         navigator.NodeTypeServer,
			Children:     []navigator.TreeNode{},
			Details:      make(map[string]interface{}),
		}

		// Add server properties as children
		if description := server.GetDescription(); description != "" {
			serverNode.AddChild(&navigator.BaseNode{
				ID:      fmt.Sprintf("server.%d.description", i),
				Content: fmt.Sprintf("  description: %s", description),
				Type:    navigator.NodeTypeServer,
			})
		}

		// Add variables if they exist
		if variables := server.Variables; variables != nil && variables.Len() > 0 {
			variablesNode := &navigator.BaseNode{
				ID:       fmt.Sprintf("server.%d.variables", i),
				Content:  "  variables: (enter to view)",
				Type:     navigator.NodeTypeServer,
				Children: []navigator.TreeNode{},
			}

			// Add each variable as a child
			for varName := range variables.Keys() {
				variable, _ := variables.Get(varName)
				if variable != nil {
					varNode := &navigator.BaseNode{
						ID:       fmt.Sprintf("server.%d.var.%s", i, varName),
						Content:  fmt.Sprintf("    %s:", varName),
						Type:     navigator.NodeTypeServer,
						Children: []navigator.TreeNode{},
					}

					// Add variable properties
					if defaultVal := variable.GetDefault(); defaultVal != "" {
						varNode.AddChild(&navigator.BaseNode{
							ID:      fmt.Sprintf("server.%d.var.%s.default", i, varName),
							Content: fmt.Sprintf("      default: %s", defaultVal),
							Type:    navigator.NodeTypeServer,
						})
					}

					if enum := variable.GetEnum(); len(enum) > 0 {
						enumNode := &navigator.BaseNode{
							ID:       fmt.Sprintf("server.%d.var.%s.enum", i, varName),
							Content:  "      enum:",
							Type:     navigator.NodeTypeServer,
							Children: []navigator.TreeNode{},
						}
						for j, enumVal := range enum {
							enumNode.AddChild(&navigator.BaseNode{
								ID:      fmt.Sprintf("server.%d.var.%s.enum.%d", i, varName, j),
								Content: fmt.Sprintf("        - %s", enumVal),
								Type:    navigator.NodeTypeServer,
							})
						}
						varNode.AddChild(enumNode)
					}

					variablesNode.AddChild(varNode)
				}
			}

			serverNode.AddChild(variablesNode)
		}

		node.AddChild(serverNode)
	}

	return node
}

// buildPathsNode creates a node for the paths section
func buildPathsNode(paths *openapi.Paths) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "paths",
		Content:     "Paths",
		Description: fmt.Sprintf("%d path(s)", paths.Len()),
		Type:        navigator.NodeTypePaths,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	// Get all paths and sort them
	var pathKeys []string
	for key := range paths.Keys() {
		pathKeys = append(pathKeys, key)
	}
	sort.Strings(pathKeys)

	// Build path nodes
	for _, pathKey := range pathKeys {
		pathItem, _ := paths.Get(pathKey)
		if pathItem != nil {
			pathNode := buildPathNode(pathKey, pathItem)
			node.AddChild(pathNode)
		}
	}

	return node
}

// buildPathNode creates a node for a single path
func buildPathNode(path string, pathItem *openapi.ReferencedPathItem) navigator.TreeNode {
	operations := getOperationsFromPath(pathItem, path)

	node := &navigator.BaseNode{
		ID:          fmt.Sprintf("path.%s", path),
		Content:     path,
		Description: fmt.Sprintf("%d operation(s)", len(operations)),
		Type:        navigator.NodeTypePath,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	// Add operations as children
	for _, op := range operations {
		node.AddChild(op)
	}

	return node
}

// getOperationsFromPath extracts all operations from a path item
func getOperationsFromPath(pathItem *openapi.ReferencedPathItem, path string) []navigator.TreeNode {
	var operations []navigator.TreeNode

	if pathItem == nil {
		return operations
	}

	pathItemObj := pathItem.GetObject()
	if pathItemObj == nil {
		return operations
	}

	// Use All() to iterate through all operations
	for method, operation := range pathItemObj.All() {
		if operation != nil {
			opNode := buildOperationNode(method, operation, path)
			operations = append(operations, opNode)
		}
	}

	return operations
}

// buildOperationNode creates a node for an operation
func buildOperationNode(method openapi.HTTPMethod, operation *openapi.Operation, path string) navigator.TreeNode {
	summary := operation.GetSummary()
	if summary == "" {
		summary = "No summary"
	}

	node := &navigator.BaseNode{
		ID:          fmt.Sprintf("operation.%s", string(method)),
		Content:     fmt.Sprintf("%s %s", string(method), summary),
		Description: path, // Use the path as description for proper display
		Type:        navigator.NodeTypeOperation,
		Children:    []navigator.TreeNode{},
		Details: map[string]interface{}{
			"method":      string(method),
			"summary":     summary,
			"path":        path,
			"description": operation.GetDescription(),
			"operationId": operation.GetOperationID(),
			"deprecated":  operation.GetDeprecated(),
		},
	}

	// Add operation details as children
	if params := operation.Parameters; len(params) > 0 {
		node.AddChild(buildParametersNode(params))
	}

	if reqBody := operation.RequestBody; reqBody != nil {
		node.AddChild(buildRequestBodyNode(reqBody))
	}

	if responses := operation.Responses; responses != nil {
		node.AddChild(buildResponsesNode(responses))
	}

	if security := operation.Security; len(security) > 0 {
		node.AddChild(buildOperationSecurityNode(security))
	}

	return node
}

// buildParametersNode creates a node for parameters
func buildParametersNode(parameters []*openapi.ReferencedParameter) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "parameters",
		Content:     "Parameters",
		Description: fmt.Sprintf("%d parameter(s)", len(parameters)),
		Type:        navigator.NodeTypeParameters,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	for i, param := range parameters {
		if param != nil {
			var paramNode *navigator.BaseNode

			if param.IsReference() {
				// Show reference as a node that can be navigated into
				paramNode = &navigator.BaseNode{
					ID:          fmt.Sprintf("parameter.%d", i),
					Content:     fmt.Sprintf("$ref: %s", param.GetReference()),
					Description: "Reference to external parameter",
					Type:        navigator.NodeTypeParameter,
					Children:    []navigator.TreeNode{},
					Details: map[string]interface{}{
						"reference":   param.GetReference().String(),
						"isReference": true,
					},
				}
			} else {
				// Show inline parameter
				paramObj := param.GetObject()
				if paramObj != nil {
					paramNode = &navigator.BaseNode{
						ID:          fmt.Sprintf("parameter.%d", i),
						Content:     fmt.Sprintf("%s (%s)", paramObj.GetName(), paramObj.GetIn()),
						Description: paramObj.GetDescription(),
						Type:        navigator.NodeTypeParameter,
						Children:    []navigator.TreeNode{},
						Details: map[string]interface{}{
							"name":        paramObj.GetName(),
							"in":          paramObj.GetIn(),
							"required":    paramObj.GetRequired(),
							"description": paramObj.GetDescription(),
						},
					}
				}
			}

			if paramNode != nil {
				node.AddChild(paramNode)
			}
		}
	}

	return node
}

// buildRequestBodyNode creates a node for request body
func buildRequestBodyNode(requestBody *openapi.ReferencedRequestBody) navigator.TreeNode {
	if requestBody.IsReference() {
		// Show reference as a node that can be navigated into
		return &navigator.BaseNode{
			ID:          "requestBody",
			Content:     fmt.Sprintf("Request Body ($ref: %s)", requestBody.GetReference()),
			Description: "Reference to external request body",
			Type:        navigator.NodeTypeRequestBody,
			Children:    []navigator.TreeNode{},
			Details: map[string]interface{}{
				"reference":   requestBody.GetReference().String(),
				"isReference": true,
			},
		}
	}

	// Show inline request body
	description := "Request Body"
	if reqBodyObj := requestBody.GetObject(); reqBodyObj != nil {
		if desc := reqBodyObj.GetDescription(); desc != "" {
			description = desc
		}
	}

	node := &navigator.BaseNode{
		ID:          "requestBody",
		Content:     "Request Body",
		Description: description,
		Type:        navigator.NodeTypeRequestBody,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	return node
}

// buildResponsesNode creates a node for responses
func buildResponsesNode(responses *openapi.Responses) navigator.TreeNode {
	count := 0
	if responses != nil {
		count = responses.Len()
	}

	node := &navigator.BaseNode{
		ID:          "responses",
		Content:     "Responses",
		Description: fmt.Sprintf("%d response(s)", count),
		Type:        navigator.NodeTypeResponses,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	if responses != nil {
		// Get all response codes and sort them
		var codes []string
		for code := range responses.Keys() {
			codes = append(codes, code)
		}
		sort.Strings(codes)

		for _, code := range codes {
			response, _ := responses.Get(code)
			if response != nil {
				var responseNode *navigator.BaseNode

				if response.IsReference() {
					// Show reference as a node that can be navigated into
					responseNode = &navigator.BaseNode{
						ID:          fmt.Sprintf("response.%s", code),
						Content:     fmt.Sprintf("%s: $ref: %s", code, response.GetReference()),
						Description: "Reference to external response",
						Type:        navigator.NodeTypeResponse,
						Children:    []navigator.TreeNode{},
						Details: map[string]interface{}{
							"code":        code,
							"reference":   response.GetReference().String(),
							"isReference": true,
						},
					}
				} else {
					// Show inline response
					responseNode = &navigator.BaseNode{
						ID:       fmt.Sprintf("response.%s", code),
						Content:  fmt.Sprintf("%s: %s", code, getResponseDescription(response)),
						Type:     navigator.NodeTypeResponse,
						Children: []navigator.TreeNode{},
						Details: map[string]interface{}{
							"code":        code,
							"description": getResponseDescription(response),
						},
					}
				}

				node.AddChild(responseNode)
			}
		}
	}

	return node
}

// buildComponentsNode creates a node for components
func buildComponentsNode(components *openapi.Components) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "components",
		Content:     "Components",
		Description: "Reusable components",
		Type:        navigator.NodeTypeComponents,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	// Add schemas
	if schemas := components.Schemas; schemas != nil && schemas.Len() > 0 {
		schemasNode := &navigator.BaseNode{
			ID:          "components.schemas",
			Content:     "Schemas",
			Description: fmt.Sprintf("%d schema(s)", schemas.Len()),
			Type:        navigator.NodeTypeSchemas,
			Children:    []navigator.TreeNode{},
			Details:     make(map[string]interface{}),
		}

		// Get all schema names and sort them
		var schemaNames []string
		for name := range schemas.Keys() {
			schemaNames = append(schemaNames, name)
		}
		sort.Strings(schemaNames)

		for _, name := range schemaNames {
			schema, _ := schemas.Get(name)
			if schema != nil {
				schemaNode := &navigator.BaseNode{
					ID:          fmt.Sprintf("schema.%s", name),
					Content:     name,
					Description: getSchemaDescription(schema),
					Type:        navigator.NodeTypeSchema,
					Children:    []navigator.TreeNode{},
					Details: map[string]interface{}{
						"name": name,
					},
				}
				schemasNode.AddChild(schemaNode)
			}
		}

		node.AddChild(schemasNode)
	}

	return node
}

// buildSecurityNode creates a node for security requirements
func buildSecurityNode(security []*openapi.SecurityRequirement) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "security",
		Content:     "Security",
		Description: fmt.Sprintf("%d requirement(s)", len(security)),
		Type:        navigator.NodeTypeSecurity,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	return node
}

// buildOperationSecurityNode creates a node for operation-level security
func buildOperationSecurityNode(security []*openapi.SecurityRequirement) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "operation.security",
		Content:     "Security",
		Description: fmt.Sprintf("%d requirement(s)", len(security)),
		Type:        navigator.NodeTypeSecurity,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	return node
}

// buildTagsNode creates a node for tags
func buildTagsNode(tags []*openapi.Tag) navigator.TreeNode {
	node := &navigator.BaseNode{
		ID:          "tags",
		Content:     "Tags",
		Description: fmt.Sprintf("%d tag(s)", len(tags)),
		Type:        navigator.NodeTypeTags,
		Children:    []navigator.TreeNode{},
		Details:     make(map[string]interface{}),
	}

	for i, tag := range tags {
		tagNode := &navigator.BaseNode{
			ID:          fmt.Sprintf("tag.%d", i),
			Content:     tag.GetName(),
			Description: tag.GetDescription(),
			Type:        navigator.NodeTypeTag,
			Children:    []navigator.TreeNode{},
			Details: map[string]interface{}{
				"name":        tag.GetName(),
				"description": tag.GetDescription(),
			},
		}
		node.AddChild(tagNode)
	}

	return node
}

// Helper functions

func getResponseDescription(response *openapi.ReferencedResponse) string {
	if response != nil {
		if responseObj := response.GetObject(); responseObj != nil {
			if desc := responseObj.GetDescription(); desc != "" {
				return desc
			}
		}
	}
	return "No description"
}

func getSchemaDescription(schema *oas3.JSONSchema[oas3.Referenceable]) string {
	if schema != nil {
		// Check if it's a schema object (Left side)
		if schema.IsLeft() {
			schemaObj := schema.GetLeft()
			if schemaObj != nil {
				if desc := schemaObj.Description; desc != nil && *desc != "" {
					return truncateText(*desc, 50)
				}
				if title := schemaObj.Title; title != nil && *title != "" {
					return *title
				}
				if schemaType := schemaObj.GetType(); len(schemaType) > 0 {
					var typeStrings []string
					for _, t := range schemaType {
						typeStrings = append(typeStrings, string(t))
					}
					return fmt.Sprintf("Type: %s", strings.Join(typeStrings, ", "))
				}
			}
		}

		// Check if it's a boolean schema (Right side)
		if schema.IsRight() {
			boolValue := schema.GetRight()
			if boolValue != nil {
				if *boolValue {
					return "Schema: true (allows anything)"
				}
				return "Schema: false (allows nothing)"
			}
		}
	}
	return "Schema"
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
