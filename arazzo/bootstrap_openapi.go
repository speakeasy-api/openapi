package arazzo

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/speakeasy-api/openapi/arazzo/criterion"
	"github.com/speakeasy-api/openapi/expression"
	"github.com/speakeasy-api/openapi/jsonpointer"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

const (
	// DefaultBootstrapWorkflowID is the workflowId of the workflow generated for BootstrapOptions.OperationIDs.
	DefaultBootstrapWorkflowID = "main"
	// UntaggedWorkflowID is the workflowId of the workflow generated for operations without tags. It can be passed
	// in BootstrapOptions.Tags to select those operations.
	UntaggedWorkflowID = "default"
)

var (
	idRegex           = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)
	inputNameRegex    = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	idSeparatorsRegex = regexp.MustCompile(`[^A-Za-z0-9]+`)
	// linkExpressionTypes are the runtime expressions an OpenAPI link can use that read from the step that declares it.
	linkExpressionTypes = []expression.ExpressionType{
		expression.ExpressionTypeURL,
		expression.ExpressionTypeMethod,
		expression.ExpressionTypeStatusCode,
		expression.ExpressionTypeRequest,
		expression.ExpressionTypeResponse,
	}
	parameterLocations = []openapi.ParameterIn{
		openapi.ParameterInPath,
		openapi.ParameterInQuery,
		openapi.ParameterInQueryString,
		openapi.ParameterInHeader,
		openapi.ParameterInCookie,
	}
)

// BootstrapOptions configures BootstrapFromOpenAPI.
type BootstrapOptions struct {
	// SourceURL is the url of the OpenAPI document's sourceDescription, either absolute or relative to the generated
	// Arazzo document. Required.
	SourceURL string
	// SourceName is the name of the OpenAPI document's sourceDescription. Defaults to a name derived from the OpenAPI
	// document's title, or `api` if it doesn't have one.
	SourceName string
	// Tags limits the generated workflows to the workflows for these tags. Use UntaggedWorkflowID to select the
	// operations without tags. Can't be combined with OperationIDs.
	Tags []string
	// OperationIDs generates a single workflow calling these operations in order, instead of one workflow per tag.
	OperationIDs []string
	// WorkflowID is the workflowId of the workflow generated for OperationIDs. Defaults to DefaultBootstrapWorkflowID.
	WorkflowID string
}

// BootstrapFromOpenAPI generates a starter Arazzo document for an OpenAPI document.
//
// By default it generates one workflow per tag, calling the tag's operations in document order. Operations without
// tags are added to the workflow named UntaggedWorkflowID (along with any operations tagged with that name), and
// operations with several tags are added to the workflow for each of their tags. Each step:
//
//   - targets the operation by operationId, or by operationPath if the operation doesn't have one
//   - passes the operation's required parameters, and its request body if it's required
//   - takes those values from workflow inputs typed from the operation's schemas, or from the outputs of an earlier
//     step whose 2xx response declares an OpenAPI link to the operation
//   - succeeds on one of the operation's documented 2xx status codes, or any 2xx status code if it documents the
//     `2XX` range or no 2xx status codes
//
// References in the OpenAPI document should be resolved first (see openapi.OpenAPI.ResolveAllReferences), as
// parameters, request bodies and links that are unresolved references are skipped.
func BootstrapFromOpenAPI(doc *openapi.OpenAPI, opts BootstrapOptions) (*Arazzo, error) {
	if doc == nil {
		return nil, errors.New("an OpenAPI document is required")
	}
	if opts.SourceURL == "" {
		return nil, errors.New("a source URL for the OpenAPI document is required")
	}
	if len(opts.Tags) > 0 && len(opts.OperationIDs) > 0 {
		return nil, errors.New("tags and operationIds can't be used together")
	}

	sourceName := opts.SourceName
	if sourceName == "" {
		sourceName = toIdentifier(doc.Info.Title, "api")
	}
	if !idRegex.MatchString(sourceName) {
		return nil, fmt.Errorf("source name `%s` must match %s", sourceName, idRegex.String())
	}

	operations := collectBootstrapOperations(doc)
	if len(operations) == 0 {
		return nil, errors.New("the OpenAPI document has no operations")
	}

	b := &bootstrapper{sourceName: sourceName, workflowIDs: map[string]bool{}}

	var workflows Workflows
	if len(opts.OperationIDs) > 0 {
		selected, err := selectOperations(operations, opts.OperationIDs)
		if err != nil {
			return nil, err
		}
		workflowID := opts.WorkflowID
		if workflowID == "" {
			workflowID = DefaultBootstrapWorkflowID
		}
		if !idRegex.MatchString(workflowID) {
			return nil, fmt.Errorf("workflowId `%s` must match %s", workflowID, idRegex.String())
		}
		workflows = append(workflows, b.buildWorkflow(workflowID, "Calls "+strings.Join(opts.OperationIDs, ", ")+" in order", "", selected))
	} else {
		groups := groupOperationsByTag(doc, operations)
		for _, tag := range opts.Tags {
			if !slices.ContainsFunc(groups, func(g tagGroup) bool { return g.tag == tag }) {
				return nil, fmt.Errorf("no operations found for tag `%s`", tag)
			}
		}
		for _, g := range groups {
			if len(opts.Tags) > 0 && !slices.Contains(opts.Tags, g.tag) {
				continue
			}
			workflows = append(workflows, b.buildWorkflow(b.uniqueWorkflowID(g.tag), g.summary, g.description, g.operations))
		}
	}

	title := doc.Info.Title
	if title == "" {
		title = "API"
	}

	return &Arazzo{
		Arazzo: Version,
		Info: Info{
			Title:       title + " workflows",
			Description: pointer.From(fmt.Sprintf("Starter workflows generated from %s %s. Review each step's parameters, request body and success criteria, then add outputs or chain further steps as needed.", title, doc.Info.Version)),
			Version:     "1.0.0",
		},
		SourceDescriptions: SourceDescriptions{
			{Name: sourceName, URL: opts.SourceURL, Type: SourceDescriptionTypeOpenAPI},
		},
		Workflows: workflows,
	}, nil
}

// bootstrapOperation is an operation of the OpenAPI document.
type bootstrapOperation struct {
	path string
	// method is the HTTP method, or the key of an additional operation.
	method     string
	additional bool
	operation  *openapi.Operation
	pathItem   *openapi.PathItem
}

// pointer returns the JSON Pointer to the operation within the OpenAPI document.
func (o bootstrapOperation) pointer() string {
	p := "/paths/" + jsonpointer.EscapeString(o.path) + "/"
	if o.additional {
		p += "additionalOperations/"
	}
	return p + jsonpointer.EscapeString(o.method)
}

// matchesOperationRef returns true if the operationRef of a link references this operation within the same
// document.
func (o bootstrapOperation) matchesOperationRef(ref string) bool {
	document, fragment, found := strings.Cut(ref, "#")
	return found && document == "" && fragment == o.pointer()
}

func collectBootstrapOperations(doc *openapi.OpenAPI) []bootstrapOperation {
	operations := []bootstrapOperation{}
	if doc.Paths == nil {
		return operations
	}
	for path, referenced := range doc.Paths.All() {
		pathItem := referenced.GetObject()
		if pathItem == nil {
			continue
		}
		for method, operation := range pathItem.All() {
			if operation != nil {
				operations = append(operations, bootstrapOperation{path: path, method: string(method), operation: operation, pathItem: pathItem})
			}
		}
		for method, operation := range pathItem.GetAdditionalOperations().All() {
			if operation != nil {
				operations = append(operations, bootstrapOperation{path: path, method: method, additional: true, operation: operation, pathItem: pathItem})
			}
		}
	}
	return operations
}

func selectOperations(operations []bootstrapOperation, operationIDs []string) ([]bootstrapOperation, error) {
	selected := make([]bootstrapOperation, 0, len(operationIDs))
	for _, id := range operationIDs {
		i := slices.IndexFunc(operations, func(o bootstrapOperation) bool { return o.operation.GetOperationID() == id })
		if i < 0 {
			return nil, fmt.Errorf("operationId `%s` not found in the OpenAPI document", id)
		}
		selected = append(selected, operations[i])
	}
	return selected, nil
}

// tagGroup is the operations of a tag.
type tagGroup struct {
	tag         string
	summary     string
	description string
	operations  []bootstrapOperation
}

// groupOperationsByTag groups operations by tag, ordering the tags declared by the document first, then other tags in
// the order they're used, then the untagged operations. Untagged operations are grouped with any operations tagged
// UntaggedWorkflowID, as both would otherwise produce a workflow of that name.
func groupOperationsByTag(doc *openapi.OpenAPI, operations []bootstrapOperation) []tagGroup {
	groups := []tagGroup{}
	index := map[string]int{}
	add := func(tag string, o *bootstrapOperation) *tagGroup {
		i, ok := index[tag]
		if !ok {
			i = len(groups)
			index[tag] = i
			groups = append(groups, tagGroup{tag: tag, summary: "Calls the `" + tag + "` operations in order"})
		}
		if o != nil {
			groups[i].operations = append(groups[i].operations, *o)
		}
		return &groups[i]
	}

	for _, tag := range doc.Tags {
		if tag == nil || tag.Name == "" {
			continue
		}
		g := add(tag.Name, nil)
		if summary := tag.GetSummary(); summary != "" {
			g.summary = summary
		}
		g.description = tag.GetDescription()
	}

	var untagged []bootstrapOperation
	for i := range operations {
		tags := operations[i].operation.GetTags()
		if len(tags) == 0 {
			untagged = append(untagged, operations[i])
		}
		seen := map[string]bool{}
		for _, tag := range tags {
			if !seen[tag] {
				seen[tag] = true
				add(tag, &operations[i])
			}
		}
	}

	if len(untagged) > 0 {
		_, tagged := index[UntaggedWorkflowID]
		g := add(UntaggedWorkflowID, nil)
		if !tagged {
			g.summary = "Calls the operations without tags in order"
		}
		g.operations = append(g.operations, untagged...)
	}

	// Declared tags without operations don't get a workflow.
	return slices.DeleteFunc(groups, func(g tagGroup) bool { return len(g.operations) == 0 })
}

// bootstrapper builds the workflows of a bootstrapped document.
type bootstrapper struct {
	sourceName  string
	workflowIDs map[string]bool
}

func (b *bootstrapper) uniqueWorkflowID(tag string) string {
	return uniqueName(toIdentifier(tag, "workflow"), b.workflowIDs)
}

// bootstrapStep is a step generated for an operation.
type bootstrapStep struct {
	operation bootstrapOperation
	step      *Step
	// outputs maps the expressions of the step's outputs to their names.
	outputs map[string]string
}

// workflowBuilder builds a single bootstrapped workflow.
type workflowBuilder struct {
	sourceName string
	stepIDs    map[string]bool
	steps      []*bootstrapStep
	inputs     *sequencedmap.Map[string, *oas3.JSONSchema[oas3.Referenceable]]
	required   []string
	// inputNames are the input names in use, and inputsByValue maps the values the inputs are for to their names.
	inputNames    map[string]bool
	inputsByValue map[string]string
}

func (b *bootstrapper) buildWorkflow(workflowID, summary, description string, operations []bootstrapOperation) *Workflow {
	wb := &workflowBuilder{
		sourceName:    b.sourceName,
		stepIDs:       map[string]bool{},
		inputs:        sequencedmap.New[string, *oas3.JSONSchema[oas3.Referenceable]](),
		inputNames:    map[string]bool{},
		inputsByValue: map[string]string{},
	}

	w := &Workflow{WorkflowID: workflowID, Summary: pointer.From(summary)}
	if description != "" {
		w.Description = pointer.From(description)
	}
	for _, o := range operations {
		w.Steps = append(w.Steps, wb.addStep(o))
	}

	if wb.inputs.Len() > 0 {
		w.Inputs = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&oas3.Schema{
			Type:       oas3.NewTypeFromString(oas3.SchemaTypeObject),
			Properties: wb.inputs,
			Required:   wb.required,
		})
	}

	return w
}

func (wb *workflowBuilder) addStep(o bootstrapOperation) *Step {
	operationID := o.operation.GetOperationID()

	base := operationID
	if base == "" {
		base = o.method + " " + o.path
	}
	stepID := uniqueName(toIdentifier(base, "step"), wb.stepIDs)

	s := &Step{StepID: stepID}
	if summary := o.operation.GetSummary(); summary != "" {
		s.Description = pointer.From(summary)
	}
	if operationID != "" {
		s.OperationID = pointer.From(expression.Expression(operationID))
	} else {
		s.OperationPath = pointer.From(expression.Expression("{$sourceDescriptions." + wb.sourceName + ".url}#" + o.pointer()))
	}

	links := wb.findLinks(o)

	for _, p := range bootstrapParameters(o) {
		if !p.GetRequired() && p.GetIn() != openapi.ParameterInPath {
			continue
		}
		value := links.parameterValue(p)
		if value == nil {
			value = wb.input(parameterKey(p.GetName(), p.GetIn()), toIdentifierMatching(p.GetName(), "input", inputNameRegex), parameterSchema(p), p.GetDescription())
		}
		s.Parameters = append(s.Parameters, &ReusableParameter{Object: &Parameter{
			Name:  p.GetName(),
			In:    pointer.From(In(p.GetIn())),
			Value: value,
		}})
	}

	if rb := o.operation.GetRequestBody().GetObject(); rb != nil && rb.Required != nil && *rb.Required {
		contentType, mediaType := preferredMediaType(rb.Content)
		payload := links.requestBody()
		if payload == nil {
			payload = wb.input("body:"+stepID, stepID+"Body", mediaType.GetSchema(), pointer.Value(rb.Description))
		}
		requestBody := &RequestBody{Payload: payload}
		if contentType != "" {
			requestBody.ContentType = pointer.From(contentType)
		}
		s.RequestBody = requestBody
	}

	s.SuccessCriteria = []*criterion.Criterion{successCriterion(o.operation)}

	wb.steps = append(wb.steps, &bootstrapStep{operation: o, step: s, outputs: map[string]string{}})

	return s
}

// input returns an expression for the workflow input providing a value, identified by key. Steps share the input for
// the same parameter (name and location). Other values get their own input, named after name with a numeric suffix if
// it's already used.
func (wb *workflowBuilder) input(key, name string, schema *oas3.JSONSchema[oas3.Referenceable], description string) *yaml.Node {
	inputName, ok := wb.inputsByValue[key]
	if !ok {
		inputName = uniqueName(name, wb.inputNames)
		wb.inputsByValue[key] = inputName
		wb.inputs.Set(inputName, inputSchema(schema, description))
		wb.required = append(wb.required, inputName)
	}
	return yml.CreateStringNode("$inputs." + inputName)
}

// stepLinks are the values provided for a step by the links of earlier steps, closest step first.
type stepLinks struct {
	parameters    []linkedValue
	requestBodies []linkedValue
}

// linkedValue is a value provided by a link in the response of an earlier step. For parameters, in is empty when
// the link doesn't qualify the parameter with a location.
type linkedValue struct {
	in   openapi.ParameterIn
	name string
	from *bootstrapStep
	node *yaml.Node
}

// parameterValue returns the value for a parameter from the first link that provides a usable one. Values are only
// converted when used, so steps don't get outputs that nothing reads.
func (l stepLinks) parameterValue(p *openapi.Parameter) *yaml.Node {
	for _, lv := range l.parameters {
		if (lv.in != "" && lv.in != p.GetIn()) || !parameterNamesMatch(p, lv.name) {
			continue
		}
		if value := lv.from.linkValue(lv.node, p.GetName()); value != nil {
			return value
		}
	}
	return nil
}

// requestBody returns the request body from the first link that provides a usable one.
func (l stepLinks) requestBody() *yaml.Node {
	for _, lv := range l.requestBodies {
		if value := lv.from.linkValue(lv.node, "requestBody"); value != nil {
			return value
		}
	}
	return nil
}

// parameterNamesMatch compares a parameter's name with the name used by a link. Header names are case-insensitive.
func parameterNamesMatch(p *openapi.Parameter, name string) bool {
	if p.GetIn() == openapi.ParameterInHeader {
		return strings.EqualFold(p.GetName(), name)
	}
	return p.GetName() == name
}

// findLinks finds the values provided for an operation by links in the 2xx responses of earlier steps, which are
// the responses their success criteria accept.
func (wb *workflowBuilder) findLinks(o bootstrapOperation) stepLinks {
	links := stepLinks{}
	for i := len(wb.steps) - 1; i >= 0; i-- {
		from := wb.steps[i]
		for code, response := range from.operation.operation.GetResponses().All() {
			if !strings.HasPrefix(code, "2") {
				continue
			}
			for _, referenced := range response.GetObject().GetLinks().All() {
				link := referenced.GetObject()
				if link == nil || !linksTo(link, o) {
					continue
				}
				for key, node := range link.GetParameters().All() {
					in, name := splitLinkParameter(key)
					links.parameters = append(links.parameters, linkedValue{in: in, name: name, from: from, node: node})
				}
				if body := link.GetRequestBody(); body != nil {
					links.requestBodies = append(links.requestBodies, linkedValue{from: from, node: body})
				}
			}
		}
	}
	return links
}

func linksTo(link *openapi.Link, o bootstrapOperation) bool {
	if id := link.GetOperationID(); id != "" {
		return id == o.operation.GetOperationID()
	}
	return o.matchesOperationRef(link.GetOperationRef())
}

// splitLinkParameter splits a link parameter key, which may be qualified with its location, such as `path.id`.
func splitLinkParameter(key string) (openapi.ParameterIn, string) {
	if in, name, found := strings.Cut(key, "."); found && slices.Contains(parameterLocations, openapi.ParameterIn(in)) {
		return openapi.ParameterIn(in), name
	}
	return "", key
}

// linkValue converts the value of a link into a value for the step it links to. Runtime expressions become
// outputs of this step and constants, including strings that start with `$` but aren't runtime expressions, are used
// as they are. Returns nil for values that can't be converted, such as invalid expressions, expressions that don't
// read from this step, and expressions embedded in strings or nested values.
func (bs *bootstrapStep) linkValue(node *yaml.Node, name string) *yaml.Node {
	if node == nil {
		return nil
	}

	e := expression.Expression(node.Value)
	if node.Kind != yaml.ScalarNode || !e.IsExpression(expression.WithArazzo11Grammar()) {
		if containsExpression(node) {
			return nil
		}
		return node
	}

	if e.Validate(expression.WithArazzo11Grammar()) != nil || !slices.Contains(linkExpressionTypes, e.GetType()) {
		return nil
	}

	outputName, ok := bs.outputs[node.Value]
	if !ok {
		used := map[string]bool{}
		for _, n := range bs.outputs {
			used[n] = true
		}
		outputName = uniqueName(toIdentifier(name, "value"), used)
		bs.outputs[node.Value] = outputName
		if bs.step.Outputs == nil {
			bs.step.Outputs = sequencedmap.New[string, *OutputValue]()
		}
		bs.step.Outputs.Set(outputName, NewOutputValueFromExpression(e))
	}

	return yml.CreateStringNode("$steps." + bs.step.StepID + ".outputs." + outputName)
}

// containsExpression returns true if a value is or contains a runtime expression, either on its own or embedded in a
// string such as `pet {$response.body#/id}`.
func containsExpression(node *yaml.Node) bool {
	if node.Kind == yaml.ScalarNode {
		return strings.Contains(node.Value, "{$") || expression.Expression(node.Value).IsExpression(expression.WithArazzo11Grammar())
	}
	return slices.ContainsFunc(node.Content, containsExpression)
}

// bootstrapParameters returns the resolved parameters of an operation, including those declared on its path item.
func bootstrapParameters(o bootstrapOperation) []*openapi.Parameter {
	params := []*openapi.Parameter{}
	index := map[string]int{}
	add := func(referenced []*openapi.ReferencedParameter) {
		for _, rp := range referenced {
			p := rp.GetObject()
			if p == nil {
				continue
			}
			key := parameterKey(p.Name, p.In)
			if i, ok := index[key]; ok {
				// Operation parameters override path item parameters with the same name and location.
				params[i] = p
				continue
			}
			index[key] = len(params)
			params = append(params, p)
		}
	}
	add(o.pathItem.GetParameters())
	add(o.operation.GetParameters())
	return params
}

// parameterSchema returns the schema of a parameter, which is either its schema or the schema of its content.
func parameterSchema(p *openapi.Parameter) *oas3.JSONSchema[oas3.Referenceable] {
	if schema := p.GetSchema(); schema != nil {
		return schema
	}
	_, mediaType := preferredMediaType(p.Content)
	return mediaType.GetSchema()
}

// preferredMediaType returns the JSON media type if there is one, otherwise the first.
func preferredMediaType(content *sequencedmap.Map[string, *openapi.MediaType]) (string, *openapi.MediaType) {
	var firstType string
	var first *openapi.MediaType
	for contentType, mediaType := range content.All() {
		mt, _, _ := strings.Cut(contentType, ";")
		mt = strings.ToLower(strings.TrimSpace(mt))
		if mt == "application/json" || strings.HasSuffix(mt, "+json") {
			return contentType, mediaType
		}
		if firstType == "" {
			firstType, first = contentType, mediaType
		}
	}
	return firstType, first
}

// inputSchema returns a workflow input schema for a value described by an OpenAPI schema. Only the top level type,
// format and enum are copied, so the input doesn't depend on references into the OpenAPI document.
func inputSchema(schema *oas3.JSONSchema[oas3.Referenceable], description string) *oas3.JSONSchema[oas3.Referenceable] {
	input := &oas3.Schema{}

	if resolved := schema.GetResolvedSchema(); resolved != nil && resolved.IsSchema() {
		s := resolved.GetSchema()
		input.Type = s.Type
		input.Format = s.Format
		input.Enum = s.Enum
		if description == "" {
			description = s.GetDescription()
		}
	}
	if description != "" {
		input.Description = pointer.From(description)
	}

	return oas3.NewJSONSchemaFromSchema[oas3.Referenceable](input)
}

// successCriterion accepts the 2xx status codes the operation documents, or any 2xx status code if it documents the
// `2XX` range or no 2xx status codes.
func successCriterion(o *openapi.Operation) *criterion.Criterion {
	codes := []string{}
	for code := range o.GetResponses().All() {
		if strings.EqualFold(code, "2XX") {
			codes = nil
			break
		}
		if n, err := strconv.Atoi(code); err == nil && n >= 200 && n < 300 {
			codes = append(codes, code)
		}
	}

	if len(codes) == 1 {
		return &criterion.Criterion{Condition: "$statusCode == " + codes[0]}
	}

	condition := `^2\d\d$`
	if len(codes) > 1 {
		condition = "^(" + strings.Join(codes, "|") + ")$"
	}
	return &criterion.Criterion{
		Context:   pointer.From(expression.Expression("$statusCode")),
		Condition: condition,
		Type:      criterion.CriterionTypeUnion{Type: pointer.From(criterion.CriterionTypeRegex)},
	}
}

// toIdentifier converts a value into an identifier matching `[A-Za-z0-9_\-]+`. Values that already match are used
// as they are, others are converted to camelCase. Returns fallback if the value has no usable characters.
func toIdentifier(value, fallback string) string {
	return toIdentifierMatching(value, fallback, idRegex)
}

// toIdentifierMatching returns value if it matches valid, otherwise value converted to alphanumeric camelCase.
// Returns fallback if the value has no alphanumeric characters.
func toIdentifierMatching(value, fallback string, valid *regexp.Regexp) string {
	if valid.MatchString(value) {
		return value
	}

	var sb strings.Builder
	for _, word := range idSeparatorsRegex.Split(value, -1) {
		if word == "" {
			continue
		}
		r := []rune(word)
		switch {
		case sb.Len() == 0 && strings.ToUpper(word) == word:
			r = []rune(strings.ToLower(word))
		case sb.Len() == 0:
			r[0] = unicode.ToLower(r[0])
		default:
			r[0] = unicode.ToUpper(r[0])
		}
		sb.WriteString(string(r))
	}

	if sb.Len() == 0 {
		return fallback
	}
	return sb.String()
}

// uniqueName returns name, or name with a numeric suffix if it's already used, and marks it as used.
func uniqueName(name string, used map[string]bool) string {
	unique := name
	for i := 2; used[unique]; i++ {
		unique = name + strconv.Itoa(i)
	}
	used[unique] = true
	return unique
}
