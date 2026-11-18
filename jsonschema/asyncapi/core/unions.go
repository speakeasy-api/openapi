package core

import (
	"context"
	"errors"
	"reflect"

	"github.com/speakeasy-api/openapi/internal/interfaces"
	"github.com/speakeasy-api/openapi/marshaller"
	values "github.com/speakeasy-api/openapi/values/core"
	"github.com/speakeasy-api/openapi/yml"
	"go.yaml.in/yaml/v3"
)

// Items is the backing model for the `items` keyword: a list of schemas (tuple validation) or a single schema.
//
// It is a dedicated type rather than a generic EitherValue because it is part of the recursive Schema definition.
type Items struct {
	marshaller.CoreModel `model:"items"`

	Schema  *values.EitherValue[Schema, bool]
	Schemas []*values.EitherValue[Schema, bool]
	IsList  bool
}

var _ interfaces.CoreModel = (*Items)(nil)

func (i *Items) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	resolved := yml.ResolveAlias(node)
	if resolved == nil {
		return nil, errors.New("node is nil")
	}

	i.SetRootNode(node)
	i.Schema = nil
	i.Schemas = nil
	i.IsList = resolved.Kind == yaml.SequenceNode

	var validationErrs []error
	var err error
	if i.IsList {
		i.Schemas = []*values.EitherValue[Schema, bool]{}
		validationErrs, err = marshaller.UnmarshalCore(ctx, parentName, node, &i.Schemas)
	} else {
		validationErrs, err = marshaller.UnmarshalCore(ctx, parentName, node, &i.Schema)
	}
	if err != nil {
		return nil, err
	}

	i.DetermineValidity(validationErrs)
	return validationErrs, nil
}

func (i *Items) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv, err := structValue(model, "Items")
	if err != nil {
		return nil, err
	}

	lf := mv.FieldByName("Schemas")
	sf := mv.FieldByName("Schema")

	if lf.IsValid() && lf.Kind() == reflect.Slice && !lf.IsNil() {
		if !i.IsList {
			valueNode = nil
		}
		i.Schema = nil
		i.IsList = true

		lv, err := marshaller.SyncValue(ctx, lf.Interface(), &i.Schemas, valueNode, false)
		if err != nil {
			return nil, err
		}
		i.SetRootNode(lv)
		return lv, nil
	}

	if i.IsList {
		valueNode = nil
	}
	i.Schemas = nil
	i.IsList = false

	if sf.IsValid() && sf.Kind() == reflect.Ptr && !sf.IsNil() {
		sv, err := marshaller.SyncValue(ctx, sf.Interface(), &i.Schema, valueNode, false)
		if err != nil {
			return nil, err
		}
		i.SetRootNode(sv)
		return sv, nil
	}

	i.Schema = nil
	i.SetRootNode(nil)
	return nil, nil
}

// Dependency is the backing model for a value of the `dependencies` keyword: a list of property names or a schema.
//
// It is a dedicated type rather than a generic EitherValue because it is part of the recursive Schema definition.
type Dependency struct {
	marshaller.CoreModel `model:"dependency"`

	Schema     *values.EitherValue[Schema, bool]
	Properties []marshaller.Node[string]
	IsList     bool
}

var _ interfaces.CoreModel = (*Dependency)(nil)

func (d *Dependency) Unmarshal(ctx context.Context, parentName string, node *yaml.Node) ([]error, error) {
	resolved := yml.ResolveAlias(node)
	if resolved == nil {
		return nil, errors.New("node is nil")
	}

	d.SetRootNode(node)
	d.Schema = nil
	d.Properties = nil
	d.IsList = resolved.Kind == yaml.SequenceNode

	var validationErrs []error
	var err error
	if d.IsList {
		d.Properties = []marshaller.Node[string]{}
		validationErrs, err = marshaller.UnmarshalCore(ctx, parentName, node, &d.Properties)
	} else {
		validationErrs, err = marshaller.UnmarshalCore(ctx, parentName, node, &d.Schema)
	}
	if err != nil {
		return nil, err
	}

	d.DetermineValidity(validationErrs)
	return validationErrs, nil
}

func (d *Dependency) SyncChanges(ctx context.Context, model any, valueNode *yaml.Node) (*yaml.Node, error) {
	mv, err := structValue(model, "Dependency")
	if err != nil {
		return nil, err
	}

	lf := mv.FieldByName("Properties")
	sf := mv.FieldByName("Schema")

	if lf.IsValid() && lf.Kind() == reflect.Slice && !lf.IsNil() {
		if !d.IsList {
			valueNode = nil
		}
		d.Schema = nil
		d.IsList = true

		lv, err := marshaller.SyncValue(ctx, lf.Interface(), &d.Properties, valueNode, false)
		if err != nil {
			return nil, err
		}
		d.SetRootNode(lv)
		return lv, nil
	}

	if d.IsList {
		valueNode = nil
	}
	d.Properties = nil
	d.IsList = false

	if sf.IsValid() && sf.Kind() == reflect.Ptr && !sf.IsNil() {
		sv, err := marshaller.SyncValue(ctx, sf.Interface(), &d.Schema, valueNode, false)
		if err != nil {
			return nil, err
		}
		d.SetRootNode(sv)
		return sv, nil
	}

	d.Schema = nil
	d.SetRootNode(nil)
	return nil, nil
}
