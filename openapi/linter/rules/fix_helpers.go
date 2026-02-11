package rules

import (
	"context"
	"fmt"
	"strconv"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

// addErrorResponseFix adds a skeleton error response to an operation's responses node.
type addErrorResponseFix struct {
	responsesNode *yaml.Node // the responses mapping node
	statusCode    string     // e.g. "401", "429", "500", "400"
	description   string     // e.g. "Unauthorized"
}

func (f *addErrorResponseFix) Description() string {
	return "Add " + f.statusCode + " response: " + f.description
}
func (f *addErrorResponseFix) Interactive() bool            { return false }
func (f *addErrorResponseFix) Prompts() []validation.Prompt { return nil }
func (f *addErrorResponseFix) SetInput([]string) error      { return nil }
func (f *addErrorResponseFix) Apply(doc any) error          { return nil }

func (f *addErrorResponseFix) ApplyNode(_ *yaml.Node) error {
	if f.responsesNode == nil || f.responsesNode.Kind != yaml.MappingNode {
		return nil
	}

	ctx := context.Background()

	// Idempotency: check if status code already exists
	_, _, found := yml.GetMapElementNodes(ctx, f.responsesNode, f.statusCode)
	if found {
		return nil
	}

	// Create: "statusCode": { description: "..." }
	responseNode := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("description"),
		yml.CreateStringNode(f.description),
	})

	yml.CreateOrUpdateMapNodeElement(ctx, f.statusCode, nil, responseNode, f.responsesNode)
	return nil
}

// addRetryAfterHeaderFix adds a Retry-After header to a 429 response node.
type addRetryAfterHeaderFix struct {
	responseNode *yaml.Node // the 429 response mapping node
}

func (f *addRetryAfterHeaderFix) Description() string {
	return "Add Retry-After header to 429 response"
}
func (f *addRetryAfterHeaderFix) Interactive() bool            { return false }
func (f *addRetryAfterHeaderFix) Prompts() []validation.Prompt { return nil }
func (f *addRetryAfterHeaderFix) SetInput([]string) error      { return nil }
func (f *addRetryAfterHeaderFix) Apply(doc any) error          { return nil }

func (f *addRetryAfterHeaderFix) ApplyNode(_ *yaml.Node) error {
	if f.responseNode == nil || f.responseNode.Kind != yaml.MappingNode {
		return nil
	}

	ctx := context.Background()

	// Check if headers already exists
	_, headersNode, found := yml.GetMapElementNodes(ctx, f.responseNode, "headers")
	if !found || headersNode == nil {
		// Create headers mapping
		headersNode = yml.CreateMapNode(ctx, nil)
		yml.CreateOrUpdateMapNodeElement(ctx, "headers", nil, headersNode, f.responseNode)
	}

	// Idempotency: check if Retry-After already exists
	_, _, found = yml.GetMapElementNodes(ctx, headersNode, "Retry-After")
	if found {
		return nil
	}

	// Create the Retry-After header:
	//   Retry-After:
	//     description: "Number of seconds to wait before retrying"
	//     schema:
	//       type: integer
	schemaNode := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("type"),
		yml.CreateStringNode("integer"),
	})
	headerNode := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("description"),
		yml.CreateStringNode("Number of seconds to wait before retrying"),
		yml.CreateStringNode("schema"),
		schemaNode,
	})

	yml.CreateOrUpdateMapNodeElement(ctx, "Retry-After", nil, headerNode, headersNode)
	return nil
}

// addDescriptionFix is an interactive fix that prompts for a description and sets it on a YAML mapping node.
type addDescriptionFix struct {
	targetNode  *yaml.Node // the mapping node to add/update "description" on
	targetLabel string     // human-readable label e.g. "tag 'users'", "operation GET /pets"
	description string     // filled by SetInput
}

func (f *addDescriptionFix) Description() string {
	return "Add description to " + f.targetLabel
}
func (f *addDescriptionFix) Interactive() bool { return true }
func (f *addDescriptionFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{
			Type:    validation.PromptFreeText,
			Message: "Enter description for " + f.targetLabel,
		},
	}
}

func (f *addDescriptionFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.description = responses[0]
	return nil
}

func (f *addDescriptionFix) Apply(doc any) error { return nil }

func (f *addDescriptionFix) ApplyNode(_ *yaml.Node) error {
	if f.targetNode == nil || f.targetNode.Kind != yaml.MappingNode || f.description == "" {
		return nil
	}
	ctx := context.Background()
	yml.CreateOrUpdateMapNodeElement(ctx, "description", nil, yml.CreateStringNode(f.description), f.targetNode)
	return nil
}

// addContactFix prompts for contact name, URL, and email and adds them to the info node.
type addContactFix struct {
	infoNode *yaml.Node
	name     string
	url      string
	email    string
}

func (f *addContactFix) Description() string { return "Add contact information to info section" }
func (f *addContactFix) Interactive() bool   { return true }
func (f *addContactFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "Contact name"},
		{Type: validation.PromptFreeText, Message: "Contact URL"},
		{Type: validation.PromptFreeText, Message: "Contact email"},
	}
}

func (f *addContactFix) SetInput(responses []string) error {
	if len(responses) != 3 {
		return fmt.Errorf("expected 3 responses, got %d", len(responses))
	}
	f.name = responses[0]
	f.url = responses[1]
	f.email = responses[2]
	return nil
}

func (f *addContactFix) Apply(doc any) error { return nil }

func (f *addContactFix) ApplyNode(_ *yaml.Node) error {
	if f.infoNode == nil || f.infoNode.Kind != yaml.MappingNode {
		return nil
	}
	ctx := context.Background()
	var content []*yaml.Node
	if f.name != "" {
		content = append(content, yml.CreateStringNode("name"), yml.CreateStringNode(f.name))
	}
	if f.url != "" {
		content = append(content, yml.CreateStringNode("url"), yml.CreateStringNode(f.url))
	}
	if f.email != "" {
		content = append(content, yml.CreateStringNode("email"), yml.CreateStringNode(f.email))
	}
	if len(content) == 0 {
		return nil
	}
	contactNode := yml.CreateMapNode(ctx, content)
	yml.CreateOrUpdateMapNodeElement(ctx, "contact", nil, contactNode, f.infoNode)
	return nil
}

// addLicenseFix prompts for license name and adds a license object to the info node.
type addLicenseFix struct {
	infoNode    *yaml.Node
	licenseName string
}

func (f *addLicenseFix) Description() string { return "Add license to info section" }
func (f *addLicenseFix) Interactive() bool   { return true }
func (f *addLicenseFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{
			Type:    validation.PromptChoice,
			Message: "License type",
			Choices: []string{"MIT", "Apache-2.0", "BSD-3-Clause", "Other"},
		},
	}
}

func (f *addLicenseFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.licenseName = responses[0]
	return nil
}

func (f *addLicenseFix) Apply(doc any) error { return nil }

func (f *addLicenseFix) ApplyNode(_ *yaml.Node) error {
	if f.infoNode == nil || f.infoNode.Kind != yaml.MappingNode || f.licenseName == "" {
		return nil
	}
	ctx := context.Background()
	licenseNode := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("name"),
		yml.CreateStringNode(f.licenseName),
	})
	yml.CreateOrUpdateMapNodeElement(ctx, "license", nil, licenseNode, f.infoNode)
	return nil
}

// addLicenseURLFix prompts for a license URL and sets it on the license node.
type addLicenseURLFix struct {
	licenseNode *yaml.Node
	url         string
}

func (f *addLicenseURLFix) Description() string { return "Add URL to license" }
func (f *addLicenseURLFix) Interactive() bool   { return true }
func (f *addLicenseURLFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "License URL"},
	}
}

func (f *addLicenseURLFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.url = responses[0]
	return nil
}

func (f *addLicenseURLFix) Apply(doc any) error { return nil }

func (f *addLicenseURLFix) ApplyNode(_ *yaml.Node) error {
	if f.licenseNode == nil || f.licenseNode.Kind != yaml.MappingNode || f.url == "" {
		return nil
	}
	ctx := context.Background()
	yml.CreateOrUpdateMapNodeElement(ctx, "url", nil, yml.CreateStringNode(f.url), f.licenseNode)
	return nil
}

// addOperationTagFix prompts for a tag and adds it to an operation.
type addOperationTagFix struct {
	operationNode *yaml.Node
	tag           string
}

func (f *addOperationTagFix) Description() string { return "Add tag to operation" }
func (f *addOperationTagFix) Interactive() bool   { return true }
func (f *addOperationTagFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "Tag for this operation"},
	}
}

func (f *addOperationTagFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.tag = responses[0]
	return nil
}

func (f *addOperationTagFix) Apply(doc any) error { return nil }

func (f *addOperationTagFix) ApplyNode(_ *yaml.Node) error {
	if f.operationNode == nil || f.operationNode.Kind != yaml.MappingNode || f.tag == "" {
		return nil
	}
	ctx := context.Background()
	// Check if tags array exists
	_, tagsNode, found := yml.GetMapElementNodes(ctx, f.operationNode, "tags")
	if !found || tagsNode == nil {
		tagsNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		yml.CreateOrUpdateMapNodeElement(ctx, "tags", nil, tagsNode, f.operationNode)
	}
	tagsNode.Content = append(tagsNode.Content, yml.CreateStringNode(f.tag))
	return nil
}

// addContactPropertyFix prompts for a single missing contact property.
type addContactPropertyFix struct {
	contactNode *yaml.Node
	property    string // "name", "url", or "email"
	value       string
}

func (f *addContactPropertyFix) Description() string {
	return "Add " + f.property + " to contact"
}
func (f *addContactPropertyFix) Interactive() bool { return true }
func (f *addContactPropertyFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "Contact " + f.property},
	}
}

func (f *addContactPropertyFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.value = responses[0]
	return nil
}

func (f *addContactPropertyFix) Apply(doc any) error { return nil }

func (f *addContactPropertyFix) ApplyNode(_ *yaml.Node) error {
	if f.contactNode == nil || f.contactNode.Kind != yaml.MappingNode || f.value == "" {
		return nil
	}
	ctx := context.Background()
	yml.CreateOrUpdateMapNodeElement(ctx, f.property, nil, yml.CreateStringNode(f.value), f.contactNode)
	return nil
}

// replaceServerURLFix prompts for a replacement server URL.
type replaceServerURLFix struct {
	urlNode *yaml.Node
	newURL  string
}

func (f *replaceServerURLFix) Description() string { return "Replace server URL" }
func (f *replaceServerURLFix) Interactive() bool   { return true }
func (f *replaceServerURLFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "New server URL"},
	}
}

func (f *replaceServerURLFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.newURL = responses[0]
	return nil
}

func (f *replaceServerURLFix) Apply(doc any) error { return nil }

func (f *replaceServerURLFix) ApplyNode(_ *yaml.Node) error {
	if f.urlNode != nil && f.newURL != "" {
		f.urlNode.Value = f.newURL
	}
	return nil
}

// addServerFix prompts for a server URL and adds it to the document.
type addServerFix struct {
	doc *openapi.OpenAPI
	url string
}

func (f *addServerFix) Description() string { return "Add server URL" }
func (f *addServerFix) Interactive() bool   { return true }
func (f *addServerFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "Server URL"},
	}
}

func (f *addServerFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.url = responses[0]
	return nil
}

func (f *addServerFix) Apply(doc any) error {
	if f.url == "" {
		return nil
	}
	oasDoc, ok := doc.(*openapi.OpenAPI)
	if !ok {
		return fmt.Errorf("expected *openapi.OpenAPI, got %T", doc)
	}
	oasDoc.Servers = append(oasDoc.Servers, &openapi.Server{URL: f.url})
	return nil
}

// setIntegerFormatFix prompts for int32 or int64 and sets the format on a schema node.
type setIntegerFormatFix struct {
	schemaNode *yaml.Node
	format     string
}

func (f *setIntegerFormatFix) Description() string { return "Set integer format" }
func (f *setIntegerFormatFix) Interactive() bool   { return true }
func (f *setIntegerFormatFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{
			Type:    validation.PromptChoice,
			Message: "Integer format",
			Choices: []string{"int32", "int64"},
		},
	}
}

func (f *setIntegerFormatFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.format = responses[0]
	return nil
}

func (f *setIntegerFormatFix) Apply(doc any) error { return nil }

func (f *setIntegerFormatFix) ApplyNode(_ *yaml.Node) error {
	if f.schemaNode == nil || f.schemaNode.Kind != yaml.MappingNode || f.format == "" {
		return nil
	}
	ctx := context.Background()
	yml.CreateOrUpdateMapNodeElement(ctx, "format", nil, yml.CreateStringNode(f.format), f.schemaNode)
	return nil
}

// setNumericPropertyFix prompts for a numeric value and sets it as a property on a schema node.
type setNumericPropertyFix struct {
	schemaNode *yaml.Node
	property   string // e.g. "maxLength", "maxItems", "maxProperties"
	label      string // human-readable prompt label
	value      int64
}

func (f *setNumericPropertyFix) Description() string {
	return "Set " + f.property
}
func (f *setNumericPropertyFix) Interactive() bool { return true }
func (f *setNumericPropertyFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: f.label},
	}
}

func (f *setNumericPropertyFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	val, err := strconv.ParseInt(responses[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid number %q: %w", responses[0], err)
	}
	f.value = val
	return nil
}

func (f *setNumericPropertyFix) Apply(doc any) error { return nil }

func (f *setNumericPropertyFix) ApplyNode(_ *yaml.Node) error {
	if f.schemaNode == nil || f.schemaNode.Kind != yaml.MappingNode {
		return nil
	}
	ctx := context.Background()
	yml.CreateOrUpdateMapNodeElement(ctx, f.property, nil, yml.CreateIntNode(f.value), f.schemaNode)
	return nil
}

// setIntegerLimitsFix prompts for minimum and maximum values for integer schemas.
type setIntegerLimitsFix struct {
	schemaNode *yaml.Node
	minVal     int64
	maxVal     int64
}

func (f *setIntegerLimitsFix) Description() string { return "Set integer minimum and maximum" }
func (f *setIntegerLimitsFix) Interactive() bool   { return true }
func (f *setIntegerLimitsFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{Type: validation.PromptFreeText, Message: "Minimum value"},
		{Type: validation.PromptFreeText, Message: "Maximum value"},
	}
}

func (f *setIntegerLimitsFix) SetInput(responses []string) error {
	if len(responses) != 2 {
		return fmt.Errorf("expected 2 responses, got %d", len(responses))
	}
	minV, err := strconv.ParseInt(responses[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid minimum %q: %w", responses[0], err)
	}
	maxV, err := strconv.ParseInt(responses[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid maximum %q: %w", responses[1], err)
	}
	f.minVal = minV
	f.maxVal = maxV
	return nil
}

func (f *setIntegerLimitsFix) Apply(doc any) error { return nil }

func (f *setIntegerLimitsFix) ApplyNode(_ *yaml.Node) error {
	if f.schemaNode == nil || f.schemaNode.Kind != yaml.MappingNode {
		return nil
	}
	ctx := context.Background()
	yml.CreateOrUpdateMapNodeElement(ctx, "minimum", nil, yml.CreateIntNode(f.minVal), f.schemaNode)
	yml.CreateOrUpdateMapNodeElement(ctx, "maximum", nil, yml.CreateIntNode(f.maxVal), f.schemaNode)
	return nil
}

// removeUnusedComponentFix is an interactive fix that removes an unused component entry.
type removeUnusedComponentFix struct {
	parentMapNode *yaml.Node // the component type's mapping node (e.g., schemas map)
	componentName string     // the key to remove
	componentRef  string     // human-readable ref e.g. "#/components/schemas/Pet"
	confirmed     bool
}

func (f *removeUnusedComponentFix) Description() string {
	return "Remove unused component " + f.componentRef
}
func (f *removeUnusedComponentFix) Interactive() bool { return true }
func (f *removeUnusedComponentFix) Prompts() []validation.Prompt {
	return []validation.Prompt{
		{
			Type:    validation.PromptChoice,
			Message: "Remove unused component " + f.componentRef + "?",
			Choices: []string{"Yes", "No"},
		},
	}
}

func (f *removeUnusedComponentFix) SetInput(responses []string) error {
	if len(responses) != 1 {
		return fmt.Errorf("expected 1 response, got %d", len(responses))
	}
	f.confirmed = responses[0] == "Yes"
	return nil
}

func (f *removeUnusedComponentFix) Apply(doc any) error { return nil }

func (f *removeUnusedComponentFix) ApplyNode(_ *yaml.Node) error {
	if !f.confirmed || f.parentMapNode == nil || f.parentMapNode.Kind != yaml.MappingNode {
		return nil
	}
	ctx := context.Background()
	yml.DeleteMapNodeElement(ctx, f.componentName, f.parentMapNode)
	return nil
}

// addPathParameterFix adds a missing path parameter definition to an operation.
type addPathParameterFix struct {
	operationNode *yaml.Node // the operation mapping node
	paramName     string     // e.g. "userId"
	schemaType    string     // "integer" or "string"
	schemaFormat  string     // e.g. "uuid" or ""
}

func (f *addPathParameterFix) Description() string {
	desc := "Add missing path parameter '" + f.paramName + "'"
	if f.schemaFormat != "" {
		desc += " (type: " + f.schemaType + ", format: " + f.schemaFormat + ")"
	} else {
		desc += " (type: " + f.schemaType + ")"
	}
	return desc
}
func (f *addPathParameterFix) Interactive() bool            { return false }
func (f *addPathParameterFix) Prompts() []validation.Prompt { return nil }
func (f *addPathParameterFix) SetInput([]string) error      { return nil }
func (f *addPathParameterFix) Apply(doc any) error          { return nil }

func (f *addPathParameterFix) ApplyNode(_ *yaml.Node) error {
	if f.operationNode == nil || f.operationNode.Kind != yaml.MappingNode {
		return nil
	}

	ctx := context.Background()

	// Get or create parameters sequence
	_, paramsNode, found := yml.GetMapElementNodes(ctx, f.operationNode, "parameters")
	if !found || paramsNode == nil || paramsNode.Kind != yaml.SequenceNode {
		paramsNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		yml.CreateOrUpdateMapNodeElement(ctx, "parameters", nil, paramsNode, f.operationNode)
	}

	// Idempotency: check if parameter already exists
	for _, elem := range paramsNode.Content {
		if elem.Kind != yaml.MappingNode {
			continue
		}
		_, nameNode, nameFound := yml.GetMapElementNodes(ctx, elem, "name")
		_, inNode, inFound := yml.GetMapElementNodes(ctx, elem, "in")
		if nameFound && inFound && nameNode.Value == f.paramName && inNode.Value == "path" {
			return nil // already exists
		}
	}

	// Build schema node
	schemaContent := []*yaml.Node{
		yml.CreateStringNode("type"),
		yml.CreateStringNode(f.schemaType),
	}
	if f.schemaFormat != "" {
		schemaContent = append(schemaContent,
			yml.CreateStringNode("format"),
			yml.CreateStringNode(f.schemaFormat))
	}
	schemaNode := yml.CreateMapNode(ctx, schemaContent)

	// Build parameter node
	paramNode := yml.CreateMapNode(ctx, []*yaml.Node{
		yml.CreateStringNode("name"),
		yml.CreateStringNode(f.paramName),
		yml.CreateStringNode("in"),
		yml.CreateStringNode("path"),
		yml.CreateStringNode("required"),
		yml.CreateBoolNode(true),
		yml.CreateStringNode("schema"),
		schemaNode,
	})

	paramsNode.Content = append(paramsNode.Content, paramNode)
	return nil
}
