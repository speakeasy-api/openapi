package asyncapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	jsValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/speakeasy-api/openapi/internal/asyncapiversion"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const draft07MetaschemaURL = "http://json-schema.org/draft-07/schema"

var (
	draft07Once      sync.Once
	draft07Validator *jsValidator.Schema
	errDraft07       error

	defaultPrinter = message.NewPrinter(language.English)
)

// draft07Metaschema returns the compiled JSON Schema Draft 07 metaschema. It is bundled with the validator
// library, so no network access is required.
func draft07Metaschema() (*jsValidator.Schema, error) {
	draft07Once.Do(func() {
		draft07Validator, errDraft07 = jsValidator.NewCompiler().Compile(draft07MetaschemaURL)
	})
	return draft07Validator, errDraft07
}

// schemaFormatOption carries the schema format being validated through validation options.
type schemaFormatOption struct {
	format asyncapiversion.SchemaFormat
}

// WithSchemaFormat sets the schema format that schemas are validated as, for example the native AsyncAPI Schema
// Object format of the owning document (`application/vnd.aai.asyncapi+json;version=3.1.0`) or JSON Schema Draft 07
// (`application/schema+json;version=draft-07`). Without it, schemas are validated as native AsyncAPI schemas.
func WithSchemaFormat(schemaFormat string) validation.Option {
	return validation.WithContextObject(&schemaFormatOption{format: asyncapiversion.ParseSchemaFormat(schemaFormat)})
}

func schemaFormatFromOptions(o *validation.Options) asyncapiversion.SchemaFormat {
	if f := validation.GetContextObject[schemaFormatOption](o); f != nil {
		return f.format
	}
	return asyncapiversion.ParseSchemaFormat(asyncapiversion.DefaultSchemaFormat(""))
}

// Validate validates the schema, which may be a boolean schema.
func (j *JSONSchema) Validate(ctx context.Context, opts ...validation.Option) []error {
	if j == nil || !j.IsSchema() {
		return nil
	}
	return j.GetSchema().Validate(ctx, opts...)
}

// Validate validates the Schema Object and all its subschemas. The structure is checked against the JSON Schema
// Draft 07 metaschema. For native AsyncAPI schemas the AsyncAPI specific rules are also checked: `discriminator`
// must name a required property defined in the same schema, `default` must conform to the declared `type`,
// and `externalDocs` must be valid.
//
// A Reference Object is only checked for a valid reference. As in both AsyncAPI and Draft 07, its siblings are ignored.
func (s *Schema) Validate(ctx context.Context, opts ...validation.Option) []error {
	if s == nil {
		return nil
	}

	o := validation.NewOptions(opts...)
	format := schemaFormatFromOptions(o)
	core := s.GetCore()
	root := core.GetRootNode()

	errs := []error{}

	if root != nil {
		errs = append(errs, validateAgainstDraft07(root)...)
	}

	errs = append(errs, s.validateSemantics(ctx, format.IsDraft07(), opts...)...)

	s.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

func (s *Schema) validateSemantics(ctx context.Context, draft07 bool, opts ...validation.Option) []error {
	if s == nil {
		return nil
	}

	core := s.GetCore()
	errs := []error{}

	if s.IsReference() {
		if err := s.Ref.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, err, core, core.Ref))
		}
		// Siblings of `$ref` are ignored.
		return errs
	}

	if !draft07 {
		errs = append(errs, s.validateDiscriminator()...)
		errs = append(errs, s.validateDefault()...)
		if s.ExternalDocs != nil {
			errs = append(errs, s.ExternalDocs.Validate(ctx, opts...)...)
		}
	}

	for _, child := range s.subschemas() {
		errs = append(errs, child.GetSchema().validateSemantics(ctx, draft07, opts...)...)
	}

	return errs
}

func (s *Schema) validateDiscriminator() []error {
	if s.Discriminator == nil {
		return nil
	}
	core := s.GetCore()
	name := *s.Discriminator

	if name == "" {
		return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationEmptyValue, errors.New("schema.discriminator must not be empty"), core, core.Discriminator)}
	}

	errs := []error{}
	if s.Properties == nil || !s.Properties.Has(name) {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidTarget, fmt.Errorf("schema.discriminator property `%s` must be defined in the schema's properties", name), core, core.Discriminator))
	}
	if !slices.Contains(s.Required, name) {
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("schema.discriminator property `%s` must be listed in the schema's required properties", name), core, core.Discriminator))
	}
	return errs
}

func (s *Schema) validateDefault() []error {
	if s.Default == nil {
		return nil
	}
	types := s.GetType()
	if len(types) == 0 {
		return nil
	}

	for _, t := range types {
		if nodeMatchesType(s.Default, t) {
			return nil
		}
	}

	core := s.GetCore()
	return []error{validation.NewValueError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError("schema.default", "schema.default must conform to the schema type [`%s`]", strings.Join(types, "`, `")), core, core.Default)}
}

// nodeMatchesType reports whether a YAML value conforms to a JSON Schema primitive type.
func nodeMatchesType(node *yaml.Node, typ string) bool {
	n := yml.ResolveAlias(node)
	if n == nil {
		return false
	}

	switch n.Kind {
	case yaml.MappingNode:
		return typ == "object"
	case yaml.SequenceNode:
		return typ == "array"
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!str", "!!timestamp":
			// Unquoted YAML dates such as 2024-01-01 resolve to !!timestamp but are strings in JSON.
			return typ == "string"
		case "!!bool":
			return typ == "boolean"
		case "!!null":
			return typ == "null"
		case "!!int":
			return typ == "integer" || typ == "number"
		case "!!float":
			if typ == "number" {
				return true
			}
			if typ == "integer" {
				// Use an exact decimal so values like 1.0000000000000001 aren't rounded to an integer.
				r, ok := new(big.Rat).SetString(n.Value)
				return ok && r.IsInt()
			}
		}
	case yaml.DocumentNode, yaml.AliasNode:
	}

	return false
}

// subschemas returns the direct subschemas that are Schema Objects.
func (s *Schema) subschemas() []*JSONSchema {
	var out []*JSONSchema
	add := func(js ...*JSONSchema) {
		for _, j := range js {
			if j.IsSchema() {
				out = append(out, j)
			}
		}
	}

	add(s.AdditionalItems, s.Contains, s.AdditionalProperties, s.PropertyNames, s.If, s.Then, s.Else, s.Not)
	if s.Items != nil {
		add(s.Items.Schema)
		add(s.Items.Schemas...)
	}
	add(s.AllOf...)
	add(s.AnyOf...)
	add(s.OneOf...)
	if s.Definitions != nil {
		for _, v := range s.Definitions.All() {
			add(v)
		}
	}
	if s.Properties != nil {
		for _, v := range s.Properties.All() {
			add(v)
		}
	}
	if s.PatternProperties != nil {
		for _, v := range s.PatternProperties.All() {
			add(v)
		}
	}
	if s.Dependencies != nil {
		for _, v := range s.Dependencies.All() {
			if v != nil {
				add(v.Schema)
			}
		}
	}

	return out
}

// Validate validates the External Documentation Object.
func (e *ExternalDocumentation) Validate(_ context.Context, _ ...validation.Option) []error {
	if e == nil {
		return nil
	}

	core := e.GetCore()
	errs := []error{}

	switch {
	case e.Ref != nil:
		if err := e.Ref.Validate(); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidReference, err, core, core.Ref))
		}
	case e.URL == nil || *e.URL == "":
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`externalDocs.url` is required"), core, core.URL))
	default:
		if _, err := url.Parse(*e.URL); err != nil {
			errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("externalDocs.url is not a valid uri: %w", err), core, core.URL))
		}
	}

	e.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// Validate validates the Multi Format Schema Object. The schema is validated when its format is the native AsyncAPI
// Schema Object of a supported version or JSON Schema Draft 07. Other formats are preserved but not validated, and a
// warning with the RuleValidationUnsupportedFormat rule is reported so callers can tell the content was not checked.
func (m *MultiFormatSchema) Validate(ctx context.Context, opts ...validation.Option) []error {
	if m == nil {
		return nil
	}

	core := m.GetCore()
	errs := []error{}

	format := asyncapiversion.ParseSchemaFormat(m.SchemaFormat)

	switch {
	case m.SchemaFormat == "":
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationRequiredField, errors.New("`multiFormatSchema.schemaFormat` is required"), core, core.SchemaFormat))
	case format.Kind == asyncapiversion.SchemaFormatKindInvalid:
		errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationInvalidFormat, fmt.Errorf("multiFormatSchema.schemaFormat `%s` is not a valid media type", m.SchemaFormat), core, core.SchemaFormat))
	case format.IsJSONSchemaBased():
		if m.Schema != nil && m.Schema.Schema != nil {
			childOpts := append(slices.Clone(opts), WithSchemaFormat(m.SchemaFormat))
			errs = append(errs, m.Schema.Schema.Validate(ctx, childOpts...)...)
		}
	default:
		if format.Kind == asyncapiversion.SchemaFormatKindProtobuf && m.Schema != nil && !m.Schema.IsReference() {
			if raw := yml.ResolveAlias(m.Schema.Raw); raw != nil && (raw.Kind != yaml.ScalarNode || raw.ShortTag() != "!!str") {
				errs = append(errs, validation.NewValueError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError("multiFormatSchema.schema", "multiFormatSchema.schema for Protocol Buffers must be inlined as a string"), core, core.Schema))
			}
		}
		errs = append(errs, validation.NewValueError(validation.SeverityWarning, validation.RuleValidationUnsupportedFormat, fmt.Errorf("multiFormatSchema.schemaFormat `%s` is not supported for validation, the schema was not validated", m.SchemaFormat), core, core.SchemaFormat))
	}

	m.Valid = len(errs) == 0 && core.GetValid()

	return errs
}

// Validate validates whichever schema the union holds.
func (u *SchemaUnion) Validate(ctx context.Context, opts ...validation.Option) []error {
	if u == nil {
		return nil
	}
	if u.MultiFormat != nil {
		return u.MultiFormat.Validate(ctx, opts...)
	}
	return u.Schema.Validate(ctx, opts...)
}

// validateAgainstDraft07 validates a schema node against the Draft 07 metaschema, reporting errors at the
// corresponding locations in node.
func validateAgainstDraft07(node *yaml.Node) []error {
	meta, err := draft07Metaschema()
	if err != nil {
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSchema, fmt.Errorf("failed to load the JSON Schema Draft 07 metaschema: %w", err), node)}
	}

	instance, err := schemaNodeToJSON(node)
	if err != nil {
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSyntax, fmt.Errorf("schema is not valid json: %w", err), node)}
	}

	err = meta.Validate(instance)
	if err == nil {
		return nil
	}

	var vErr *jsValidator.ValidationError
	if !errors.As(err, &vErr) {
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSchema, fmt.Errorf("schema invalid: %w", err), node)}
	}

	return dedupeErrors(rootCauses(vErr, node))
}

func rootCauses(err *jsValidator.ValidationError, root *yaml.Node) []error {
	if len(err.Causes) > 0 {
		var errs []error
		for _, cause := range filterBranchTypeMismatches(err) {
			errs = append(errs, rootCauses(cause, root)...)
		}
		return errs
	}

	target := nodeAtPath(root, err.InstanceLocation)
	parentName := "schema"
	if len(err.InstanceLocation) > 0 {
		parentName += "." + strings.Join(err.InstanceLocation, ".")
	}
	msg := err.ErrorKind.LocalizedString(defaultPrinter)

	switch t := err.ErrorKind.(type) {
	case *kind.Type:
		want := "`" + strings.Join(t.Want, "`, `") + "`"
		if len(t.Want) > 1 {
			want = "one of [" + want + "]"
		}
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationTypeMismatch, validation.NewTypeMismatchError(parentName, "%s expected %s, got `%s`", parentName, want, t.Got), target)}
	case *kind.Required:
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationRequiredField, fmt.Errorf("`%s` %s", parentName, strings.ReplaceAll(msg, "'", "`")), target)}
	default:
		return []error{validation.NewValidationError(validation.SeverityError, validation.RuleValidationInvalidSchema, fmt.Errorf("%s %s", parentName, msg), target)}
	}
}

// filterBranchTypeMismatches drops anyOf/oneOf branches that failed only because the value has a different type,
// when another branch failed for a more specific reason. For example the Draft 07 `type` keyword is
// anyOf [simpleTypes, array of simpleTypes], so `type: text` would otherwise also report "expected array".
func filterBranchTypeMismatches(err *jsValidator.ValidationError) []*jsValidator.ValidationError {
	switch err.ErrorKind.(type) {
	case *kind.AnyOf, *kind.OneOf:
	default:
		return err.Causes
	}

	filtered := make([]*jsValidator.ValidationError, 0, len(err.Causes))
	for _, cause := range err.Causes {
		if _, isType := cause.ErrorKind.(*kind.Type); isType && len(cause.Causes) == 0 {
			continue
		}
		filtered = append(filtered, cause)
	}
	if len(filtered) == 0 {
		return err.Causes
	}
	return filtered
}

func dedupeErrors(errs []error) []error {
	seen := make(map[string]struct{}, len(errs))
	out := make([]error, 0, len(errs))
	for _, err := range errs {
		key := err.Error()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, err)
	}
	return out
}

// nodeAtPath returns the deepest node found following path from root, falling back to the closest ancestor.
func nodeAtPath(root *yaml.Node, path []string) *yaml.Node {
	current := yml.ResolveAlias(root)
	for _, part := range path {
		if current == nil {
			return root
		}
		var next *yaml.Node
		switch current.Kind {
		case yaml.MappingNode:
			for i := 0; i+1 < len(current.Content); i += 2 {
				if k := yml.ResolveAlias(current.Content[i]); k != nil && k.Value == part {
					next = current.Content[i+1]
					break
				}
			}
		case yaml.SequenceNode:
			if idx, err := strconv.Atoi(part); err == nil && idx >= 0 && idx < len(current.Content) {
				next = current.Content[idx]
			}
		case yaml.DocumentNode, yaml.ScalarNode, yaml.AliasNode:
		}
		if next == nil {
			return current
		}
		current = yml.ResolveAlias(next)
	}
	if current == nil {
		return root
	}
	return current
}

// Draft 07 keywords whose values are schemas, lists of schemas or maps of schemas. Used to only treat `$ref`
// as a reference where a schema is expected, rather than within values such as `default`, `enum` or a property named `$ref`.
var (
	schemaKeywords     = []string{"additionalItems", "contains", "additionalProperties", "propertyNames", "if", "then", "else", "not"}
	schemaListKeywords = []string{"allOf", "anyOf", "oneOf"}
	schemaMapKeywords  = []string{"definitions", "properties", "patternProperties"}
)

// schemaNodeToJSON converts a schema node into a JSON value for validation. Where a schema is a Reference Object,
// its siblings are dropped, as they are ignored by both AsyncAPI and Draft 07.
func schemaNodeToJSON(node *yaml.Node) (any, error) {
	n := yml.ResolveAlias(node)
	if n == nil {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return valueToJSON(n)
	}

	out := map[string]any{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := yml.ResolveAlias(n.Content[i])
		if k != nil && k.Value == "$ref" {
			v, err := valueToJSON(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			return map[string]any{"$ref": v}, nil
		}
	}

	for i := 0; i+1 < len(n.Content); i += 2 {
		k := yml.ResolveAlias(n.Content[i])
		if k == nil {
			continue
		}
		key := k.Value
		value := n.Content[i+1]
		resolved := yml.ResolveAlias(value)

		var v any
		var err error
		switch {
		case slices.Contains(schemaKeywords, key):
			v, err = schemaNodeToJSON(value)
		case key == "items" && resolved != nil && resolved.Kind == yaml.SequenceNode:
			v, err = schemaListToJSON(resolved)
		case key == "items":
			v, err = schemaNodeToJSON(value)
		case slices.Contains(schemaListKeywords, key) && resolved != nil && resolved.Kind == yaml.SequenceNode:
			v, err = schemaListToJSON(resolved)
		case (slices.Contains(schemaMapKeywords, key) || key == "dependencies") && resolved != nil && resolved.Kind == yaml.MappingNode:
			m := map[string]any{}
			for j := 0; j+1 < len(resolved.Content); j += 2 {
				mk := yml.ResolveAlias(resolved.Content[j])
				if mk == nil {
					continue
				}
				mv := resolved.Content[j+1]
				var cv any
				if mvr := yml.ResolveAlias(mv); key == "dependencies" && mvr != nil && mvr.Kind == yaml.SequenceNode {
					cv, err = valueToJSON(mv)
				} else {
					cv, err = schemaNodeToJSON(mv)
				}
				if err != nil {
					return nil, err
				}
				m[mk.Value] = cv
			}
			v = m
		default:
			v, err = valueToJSON(value)
		}
		if err != nil {
			return nil, err
		}
		out[key] = v
	}

	return out, nil
}

func schemaListToJSON(n *yaml.Node) (any, error) {
	out := make([]any, 0, len(n.Content))
	for _, c := range n.Content {
		v, err := schemaNodeToJSON(c)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// valueToJSON converts an arbitrary YAML value into a JSON value as expected by the validator.
func valueToJSON(node *yaml.Node) (any, error) {
	n := yml.ResolveAlias(node)
	if n == nil {
		return nil, nil
	}

	switch n.Kind {
	case yaml.MappingNode:
		out := map[string]any{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := yml.ResolveAlias(n.Content[i])
			if k == nil {
				continue
			}
			v, err := valueToJSON(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			out[k.Value] = v
		}
		return out, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := valueToJSON(c)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!null":
			return nil, nil
		case "!!bool":
			var b bool
			if err := n.Decode(&b); err != nil {
				return nil, err
			}
			return b, nil
		case "!!int":
			i, err := strconv.ParseInt(strings.ReplaceAll(n.Value, "_", ""), 0, 64)
			if err != nil {
				// Out of range for int64, keep the literal digits
				return json.Number(n.Value), nil //nolint:nilerr
			}
			return json.Number(strconv.FormatInt(i, 10)), nil
		case "!!float":
			f, err := strconv.ParseFloat(n.Value, 64)
			if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
				return nil, fmt.Errorf("`%s` is not a valid JSON number", n.Value)
			}
			return json.Number(strconv.FormatFloat(f, 'g', -1, 64)), nil
		default:
			return n.Value, nil
		}
	case yaml.DocumentNode, yaml.AliasNode:
	}

	return nil, fmt.Errorf("unsupported node kind `%s`", yml.NodeKindToString(n.Kind))
}
