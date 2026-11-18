package asyncapi

import (
	"errors"
	"fmt"

	"github.com/speakeasy-api/openapi/jsonschema/asyncapi/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"go.yaml.in/yaml/v3"
)

// Items is the value of the `items` keyword: either a list of schemas (tuple validation) or a single schema.
// Set Schemas (including to an empty, non-nil slice) for the list form, otherwise Schema.
type Items struct {
	// Schema is the single schema applied to every item.
	Schema *JSONSchema
	// Schemas are the schemas applied to the items at the corresponding positions.
	Schemas []*JSONSchema

	core core.Items
}

var _ marshaller.Populator = (*Items)(nil)

// NewItemsFromSchema creates an `items` value applying the schema to every item.
func NewItemsFromSchema(schema *JSONSchema) *Items {
	return &Items{Schema: schema}
}

// NewItemsFromSchemas creates an `items` value applying the schemas positionally.
func NewItemsFromSchemas(schemas ...*JSONSchema) *Items {
	if schemas == nil {
		schemas = []*JSONSchema{}
	}
	return &Items{Schemas: schemas}
}

// IsList returns true if `items` is a list of schemas.
func (i *Items) IsList() bool {
	return i != nil && i.Schemas != nil
}

// GetCore returns the low level representation of the items value.
func (i *Items) GetCore() *core.Items {
	if i == nil {
		return nil
	}
	return &i.core
}

// GetRootNode returns the node the items value was parsed from.
func (i *Items) GetRootNode() *yaml.Node {
	if i == nil {
		return nil
	}
	return i.core.GetRootNode()
}

func (i *Items) Populate(source any) error {
	var c core.Items
	switch s := source.(type) {
	case core.Items:
		c = s
	case *core.Items:
		if s == nil {
			return nil
		}
		c = *s
	default:
		return fmt.Errorf("expected core.Items, got %T", source)
	}

	i.Schema = nil
	i.Schemas = nil
	i.core = c

	var errs []error
	if c.IsList {
		i.Schemas = make([]*JSONSchema, 0, len(c.Schemas))
		for _, cs := range c.Schemas {
			js := &JSONSchema{}
			if cs != nil {
				var err error
				js, err = populateNested[JSONSchema](cs)
				errs = append(errs, err)
			}
			i.Schemas = append(i.Schemas, js)
		}
	} else if c.Schema != nil {
		var err error
		i.Schema, err = populateNested[JSONSchema](c.Schema)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// populateNested populates a nested model from its core.
func populateNested[T any](source any) (*T, error) {
	target := new(T)
	return target, marshaller.PopulateWithContext(source, target, nil)
}

// Dependency is a value of the `dependencies` keyword: either a list of property names that are required when
// the property is present, or a schema the instance must validate against when the property is present.
// Set Properties (including to an empty, non-nil slice) for the list form, otherwise Schema.
type Dependency struct {
	// Schema is the schema the instance must validate against.
	Schema *JSONSchema
	// Properties are the property names that become required.
	Properties []string

	core core.Dependency
}

var _ marshaller.Populator = (*Dependency)(nil)

// NewDependencyFromSchema creates a schema dependency.
func NewDependencyFromSchema(schema *JSONSchema) *Dependency {
	return &Dependency{Schema: schema}
}

// NewDependencyFromProperties creates a property dependency.
func NewDependencyFromProperties(properties ...string) *Dependency {
	if properties == nil {
		properties = []string{}
	}
	return &Dependency{Properties: properties}
}

// IsList returns true if the dependency is a list of property names.
func (d *Dependency) IsList() bool {
	return d != nil && d.Properties != nil
}

// GetCore returns the low level representation of the dependency.
func (d *Dependency) GetCore() *core.Dependency {
	if d == nil {
		return nil
	}
	return &d.core
}

// GetRootNode returns the node the dependency was parsed from.
func (d *Dependency) GetRootNode() *yaml.Node {
	if d == nil {
		return nil
	}
	return d.core.GetRootNode()
}

func (d *Dependency) Populate(source any) error {
	var c core.Dependency
	switch s := source.(type) {
	case core.Dependency:
		c = s
	case *core.Dependency:
		if s == nil {
			return nil
		}
		c = *s
	default:
		return fmt.Errorf("expected core.Dependency, got %T", source)
	}

	d.Schema = nil
	d.Properties = nil
	d.core = c

	var err error
	if c.IsList {
		d.Properties = make([]string, 0, len(c.Properties))
		for _, p := range c.Properties {
			d.Properties = append(d.Properties, p.Value)
		}
	} else if c.Schema != nil {
		d.Schema, err = populateNested[JSONSchema](c.Schema)
	}
	return err
}
