package extensions

import (
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
}

// New will create a new extensions set.
func New(elements ...*Element) *Extensions {
	ee := make([]*sequencedmap.Element[string, Extension], len(elements))
	for i, element := range elements {
		ee[i] = sequencedmap.NewElem(element.Key, element.Value)
	}

	return &Extensions{
		sequencedmap.New(ee...),
	}
}

// Init will initialize the extensions set.
func (e *Extensions) Init() {
	e.Map = sequencedmap.New[string, Extension]()
}
