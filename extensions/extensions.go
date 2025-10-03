package extensions

import (
	"context"
	"fmt"
	"reflect"

	"github.com/speakeasy-api/openapi/errors"
	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

const (
	ErrNotFound = errors.Error("not found")
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

// Len returns the number of elements in the extensions map. nil safe.
func (e *Extensions) Len() int {
	if e == nil || e.Map == nil {
		return 0
	}

	return e.Map.Len()
}

func (e *Extensions) SetCore(core any) {
	c, ok := core.(*sequencedmap.Map[string, marshaller.Node[*yaml.Node]])
	if !ok {
		return
	}

	e.core = c
}

func (e *Extensions) GetCore() *sequencedmap.Map[string, marshaller.Node[*yaml.Node]] {
	return e.core
}

func (e *Extensions) Populate(source any) error {
	e.Init()

	se, ok := source.(*sequencedmap.Map[string, marshaller.Node[Extension]])
	if !ok {
		return fmt.Errorf("expected source to be *sequencedmap.Map[string, marshaller.Node[Extension]], got %s", reflect.TypeOf(source))
	}

	for key, value := range se.All() {
		e.Set(key, value.Value)
	}

	e.SetCore(se)

	return nil
}

// UnmarshalExtensionModel will unmarshal the extension into a model and its associated core model.
func UnmarshalExtensionModel[H any, L any](ctx context.Context, e *Extensions, ext string, m *H) ([]error, error) {
	if e == nil {
		return nil, ErrNotFound.Wrap(errors.New("extensions is nil"))
	}

	if !e.Has(ext) {
		return nil, ErrNotFound
	}

	c, validationErrs, err := core.UnmarshalExtensionModel[L](ctx, e.core, ext)
	if err != nil || c == nil {
		return nil, err
	}

	var mV H

	if err := marshaller.Populate(*c, &mV); err != nil {
		return nil, err
	}
	*m = mV

	return validationErrs, nil
}

// GetExtensionValue will return the value of the extension. Useful for scalar values or where a model without a core is required.
func GetExtensionValue[T any](e *Extensions, ext string) (*T, error) {
	var zero *T

	if e == nil {
		return zero, ErrNotFound.Wrap(errors.New("extensions is nil"))
	}

	node := e.GetOrZero(ext)
	if node == nil {
		return zero, ErrNotFound
	}

	var t T
	if err := node.Decode(&t); err != nil {
		return zero, err
	}

	return &t, nil
}

// IsEqual compares two Extensions instances for equality.
// Treats both empty and nil extensions as equal.
func (e *Extensions) IsEqual(other *Extensions) bool {
	if e == nil && other == nil {
		return true
	}

	// Treat nil and empty extensions as equal
	eLen := 0
	if e != nil {
		eLen = e.Len()
	}
	otherLen := 0
	if other != nil {
		otherLen = other.Len()
	}

	if eLen == 0 && otherLen == 0 {
		return true
	}

	if eLen != otherLen {
		return false
	}

	// Compare all extension key-value pairs
	for key, valueA := range e.All() {
		valueB, exists := other.Get(key)
		if !exists {
			return false
		}
		// Use the yml package's EqualNodes function for yaml.Node comparison
		if !yml.EqualNodes(valueA, valueB) {
			return false
		}
	}
	return true
}
