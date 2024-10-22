package extensions

import (
	"context"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

// Extension represents a single extension to an object, in its raw form.
type Extension = *yaml.Node

// Element represents a key/value pair of a set of extensions.
type Element struct {
	*sequencedmap.Element[string, Extension]
}

// NewElem will create a new element for the extensions set.
func NewElem(key string, value *yaml.Node) *Element {
	return &Element{
		sequencedmap.NewElem(key, value),
	}
}

// Extensions represents a set of extensions to an object.
type Extensions struct {
	*sequencedmap.Map[string, Extension]

	core core.Extensions
}

// New will create a new extensions set.
func New(elements ...*Element) *Extensions {
	ee := make([]*sequencedmap.Element[string, Extension], len(elements))
	for i, element := range elements {
		ee[i] = sequencedmap.NewElem(element.Key, element.Value)
	}

	return &Extensions{
		Map: sequencedmap.New(ee...),
	}
}

// Init will initialize the extensions set.
func (e *Extensions) Init() {
	e.Map = sequencedmap.New[string, Extension]()
}

func (e *Extensions) SetCore(core any) {
	c, ok := core.(*sequencedmap.Map[string, marshaller.Node[*yaml.Node]])
	if !ok {
		return
	}

	e.core = c
}

func UnmarshalExtensionModel[H any, L any](ctx context.Context, e *Extensions, ext string, m *H) error {
	c, err := core.UnmarshalExtensionModel[L](ctx, e.core, ext)
	if err != nil {
		return err
	}

	var mV H

	if err := marshaller.PopulateModel(*c, &mV); err != nil {
		return err
	}
	*m = mV

	return nil
}
