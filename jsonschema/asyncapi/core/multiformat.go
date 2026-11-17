package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/internal/asyncapiversion"
	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

type schemaFormatContextKey struct{}

func withSchemaFormat(ctx context.Context, format asyncapiversion.SchemaFormat) context.Context {
	return context.WithValue(ctx, schemaFormatContextKey{}, format)
}

func schemaFormatFromContext(ctx context.Context) asyncapiversion.SchemaFormat {
	if f, ok := ctx.Value(schemaFormatContextKey{}).(asyncapiversion.SchemaFormat); ok {
		return f
	}
	return asyncapiversion.ParseSchemaFormat(asyncapiversion.DefaultSchemaFormat(""))
}

// MultiFormatSchema is the backing model for an AsyncAPI Multi Format Schema Object.
type MultiFormatSchema struct {
	marshaller.CoreModel `model:"multiFormatSchema"`

	SchemaFormat marshaller.Node[string]            `key:"schemaFormat"`
	Schema       marshaller.Node[*SchemaDefinition] `key:"schema" required:"true"`

	Extensions core.Extensions `key:"extensions"`
}

var _ interfaces.CoreModel = (*MultiFormatSchema)(nil)

// Unmarshal determines the declared schema format before unmarshalling the object, so the `schema`
// value is interpreted using that format regardless of key order.
func (m *MultiFormatSchema) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	format := ""
	if resolved := yml.ResolveAlias(node); resolved != nil && resolved.Kind == yaml.MappingNode {
		if _, valueNode, found := yml.GetMapElementNodes(ctx, resolved, "schemaFormat"); found {
			if v := yml.ResolveAlias(valueNode); v != nil && v.Kind == yaml.ScalarNode {
				format = v.Value
			}
		}
	}

	return marshaller.UnmarshalModel(withSchemaFormat(ctx, asyncapiversion.ParseSchemaFormat(format)), node, m)
}

// SchemaDefinition is the backing model for the `schema` value of a Multi Format Schema Object.
// JSON Schema based formats (the native AsyncAPI Schema Object and JSON Schema Draft 07) are parsed into
// Schema, all other formats are retained as the raw node in Raw.
type SchemaDefinition struct {
	marshaller.CoreModel `model:"schemaDefinition"`

	Schema *values.EitherValue[Schema, bool]
	Raw    values.Value
}

var _ interfaces.CoreModel = (*SchemaDefinition)(nil)

func (d *SchemaDefinition) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	if yml.ResolveAlias(node) == nil {
		return nil, errors.New("node is nil")
	}

	d.SetRootNode(node)
	d.Schema = nil
	d.Raw = nil

	if !schemaFormatFromContext(ctx).IsJSONSchemaBased() {
		d.Raw = node
		d.SetValid(true, true)
		return nil, nil
	}

	validationErrs, err := marshaller.UnmarshalCore(ctx, parentName, node, &d.Schema)
	if err != nil {
		return nil, err
	}
	d.DetermineValidity(validationErrs)

	return validationErrs, nil
}

func (d *SchemaDefinition) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv, err := structValue(model, "SchemaDefinition")
	if err != nil {
		return nil, err
	}

	sf := mv.FieldByName("Schema")
	rf := mv.FieldByName("Raw")

	if sf.IsValid() && sf.Kind() == reflect.Ptr && !sf.IsNil() {
		// Don't reuse a raw node for a typed schema, as it may hold unrelated content
		if d.Raw != nil {
			valueNode = nil
		}
		d.Raw = nil

		sv, err := marshaller.SyncValue(ctx, sf.Interface(), &d.Schema, valueNode, false)
		if err != nil {
			return nil, err
		}
		d.SetRootNode(sv)
		return sv, nil
	}

	d.Schema = nil

	if rf.IsValid() && rf.Kind() == reflect.Ptr && !rf.IsNil() {
		raw, ok := rf.Interface().(*yaml.Node)
		if !ok {
			return nil, fmt.Errorf("SchemaDefinition.SyncChanges expected Raw to be a *yaml.Node, got `%s`", rf.Type())
		}
		d.Raw = raw
		d.SetRootNode(raw)
		return raw, nil
	}

	d.Raw = nil
	d.SetRootNode(nil)
	return nil, nil
}

// SchemaUnion is the backing model for the values AsyncAPI accepts wherever a schema is expected:
// a Schema Object, a boolean schema, a Reference Object or a Multi Format Schema Object.
//
// A mapping containing `$ref` is a Reference Object (unmarshalled into Schema). Otherwise a mapping
// containing `schemaFormat` is a Multi Format Schema Object. Anything else is a Schema Object.
type SchemaUnion struct {
	marshaller.CoreModel `model:"schemaUnion"`

	Schema      *values.EitherValue[Schema, bool]
	MultiFormat *MultiFormatSchema
}

var _ interfaces.CoreModel = (*SchemaUnion)(nil)

// IsMultiFormatNode returns true if the node is a Multi Format Schema Object according to the SchemaUnion selection rules.
func IsMultiFormatNode(ctx context.Context, node *yaml.Node) bool {
	resolved := yml.ResolveAlias(node)
	if resolved == nil || resolved.Kind != yaml.MappingNode {
		return false
	}
	if _, _, isRef := yml.GetMapElementNodes(ctx, resolved, "$ref"); isRef {
		return false
	}
	_, _, hasFormat := yml.GetMapElementNodes(ctx, resolved, "schemaFormat")
	return hasFormat
}

func (u *SchemaUnion) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	if yml.ResolveAlias(node) == nil {
		return nil, errors.New("node is nil")
	}

	u.SetRootNode(node)
	u.Schema = nil
	u.MultiFormat = nil

	var validationErrs []error
	var err error

	if IsMultiFormatNode(ctx, node) {
		validationErrs, err = marshaller.UnmarshalCore(ctx, parentName, node, &u.MultiFormat)
	} else {
		validationErrs, err = marshaller.UnmarshalCore(ctx, parentName, node, &u.Schema)
	}
	if err != nil {
		return nil, err
	}

	u.DetermineValidity(validationErrs)

	return validationErrs, nil
}

func (u *SchemaUnion) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv, err := structValue(model, "SchemaUnion")
	if err != nil {
		return nil, err
	}

	sf := mv.FieldByName("Schema")
	mf := mv.FieldByName("MultiFormat")

	if sf.IsValid() && sf.Kind() == reflect.Ptr && !sf.IsNil() {
		if u.MultiFormat != nil {
			valueNode = nil
		}
		u.MultiFormat = nil

		sv, err := marshaller.SyncValue(ctx, sf.Interface(), &u.Schema, valueNode, false)
		if err != nil {
			return nil, err
		}
		u.SetRootNode(sv)
		return sv, nil
	}

	if u.Schema != nil {
		// Don't reuse the node of a native schema for a multi format schema, as its keys would leak into the output
		valueNode = nil
	}
	u.Schema = nil

	if mf.IsValid() && mf.Kind() == reflect.Ptr && !mf.IsNil() {
		mfv, err := marshaller.SyncValue(ctx, mf.Interface(), &u.MultiFormat, valueNode, false)
		if err != nil {
			return nil, err
		}
		u.SetRootNode(mfv)
		return mfv, nil
	}

	u.MultiFormat = nil
	u.SetRootNode(nil)
	return nil, nil
}

func structValue(model any, name string) (reflect.Value, error) {
	mv := reflect.ValueOf(model)
	for mv.Kind() == reflect.Ptr {
		if mv.IsNil() {
			return reflect.Value{}, fmt.Errorf("%s.SyncChanges expected a non-nil model", name)
		}
		mv = mv.Elem()
	}
	if mv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("%s.SyncChanges expected a struct, got `%s`", name, mv.Kind())
	}
	return mv, nil
}
